package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alist-org/alist/v3/internal/driver"
	alistModel "github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"

	fp "github.com/kawai-network/fileprocessor"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InitFileprocBridge() {
	dsn := os.Getenv("FILEPROC_PG_DSN")
	if dsn == "" {
		return
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		slog.Warn("fileproc: pgxpool.New failed (disabled)", "err", err)
		return
	}
	dim := embeddingDim()
	vec, err := fp.NewPgVectorStoreWithPool(context.Background(), pool, dim, "", nil)
	if err != nil {
		slog.Warn("fileproc: NewPgVectorStoreWithPool failed (disabled)", "err", err)
		return
	}
	emb := fp.NewEmbeddingCache(&remoteEmbedder{
		url:   getEnv("FILEPROC_EMBEDDING_URL", "https://openrouter.ai/api/v1/embeddings"),
		key:   getEnv("FILEPROC_EMBEDDING_KEY", os.Getenv("OPENROUTER_API_KEY")),
		model: getEnv("FILEPROC_EMBEDDING_MODEL", "text-embedding-3-small"),
	}, nil)
	op.RegisterFileUploadedHook(func(ctx context.Context, st driver.Driver, parent string, file alistModel.Obj) {
		if file.IsDir() {
			return
		}
		uid, _ := ctx.Value("kratos_identity_id").(string)
		if uid == "" {
			return
		}
		store, err := fp.NewPostgresFileStoreWithPool(pool, fp.PostgresFileStoreOwner{UserID: uid})
		if err != nil {
			slog.Error("fileproc: NewPostgresFileStoreWithPool", "err", err)
			return
		}
		rag := fp.NewRAGProcessor(store.ChunkStore(), vec, emb, nil)
		proc, err := fp.New(fp.Config{FileStore: store, RAGProcessor: rag})
		if err != nil {
			slog.Error("fileproc: New", "err", err)
			return
		}
		p := filepath.Join(st.GetStorage().MountPath, parent, file.GetName())
		c, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		if _, err := proc.ProcessFile(c, fp.Request{
			FilePath: p, Filename: file.GetName(), Source: "alist://" + uid, EnableRAG: true,
		}); err != nil {
			slog.Error("fileproc: ProcessFile", "err", err, "path", p)
		}
	})
	slog.Info("fileproc: registered", "dim", dim)
}

func embeddingDim() int {
	s := os.Getenv("FILEPROC_EMBEDDING_DIM")
	if s == "" {
		return 1024
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 1024
	}
	return n
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type remoteEmbedder struct{ url, key, model string }

func (e *remoteEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	b, _ := json.Marshal(map[string]any{"input": texts, "model": e.model})
	req, err := http.NewRequestWithContext(ctx, "POST", e.url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("create req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.key != "" {
		req.Header.Set("Authorization", "Bearer "+e.key)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("api %s: HTTP %d: %s", e.url, r.StatusCode, string(raw))
	}
	var api struct {
		Data []struct{ Embedding []float64 `json:"embedding"` } `json:"data"`
	}
	if err := json.Unmarshal(raw, &api); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	out := make([][]float32, len(api.Data))
	for i, d := range api.Data {
		v := make([]float32, len(d.Embedding))
		for j, f := range d.Embedding {
			v[j] = float32(f)
		}
		out[i] = v
	}
	return out, nil
}

func (e *remoteEmbedder) Dimension() int {
	if d := embeddingDim(); d > 0 {
		return d
	}
	return 1024
}
