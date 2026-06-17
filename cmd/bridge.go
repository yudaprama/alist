package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
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
	dim := fp.DefaultEmbeddingDim
	vec, err := fp.NewPgVectorStoreWithPool(context.Background(), pool, dim, "", nil)
	if err != nil {
		slog.Warn("fileproc: NewPgVectorStoreWithPool failed (disabled)", "err", err)
		return
	}
	emb := fp.NewEmbeddingCache(
		fp.NewOpenAIEmbedder(
			"https://openrouter.ai/api/v1/embeddings",
			"",
			"text-embedding-3-small",
			dim,
		),
		nil,
	)
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
