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
	vec, err := fp.NewPublicEmbeddingsStoreWithPool(context.Background(), pool, nil)
	if err != nil {
		slog.Warn("fileproc: NewPublicEmbeddingsStoreWithPool failed (disabled)", "err", err)
		return
	}
	// Embedder config mirrors egent-jobs/embeddings + egent-lobehub/knowledge so
	// every writer/reader of public.embeddings shares one provider + dimension.
	// A real API key is REQUIRED — with an empty key the embed call 401s and
	// ingest silently produces no vectors.
	embedURL := os.Getenv("OPENAI_EMBEDDINGS_URL")
	if embedURL == "" {
		embedURL = "https://api.openai.com/v1/embeddings"
	}
	embedModel := os.Getenv("OPENAI_EMBEDDINGS_MODEL")
	if embedModel == "" {
		embedModel = "text-embedding-3-small"
	}
	embedKey := os.Getenv("OPENAI_API_KEY")
	if embedKey == "" {
		embedKey = os.Getenv("MODEL_API_KEY")
	}
	if embedKey == "" {
		slog.Warn("fileproc: no embedder API key (OPENAI_API_KEY/MODEL_API_KEY) — RAG ingest will fail to embed")
	}
	emb := fp.NewEmbeddingCache(
		fp.NewOpenAIEmbedder(embedURL, embedKey, embedModel, dim),
		nil,
	)

	// Vision-language config for image understanding. When FILEPROC_VL_URL (or
	// PLANO_LLM_GATEWAY, which planoctl sets) is present, uploaded images get a
	// VL description + OCR cleanup via an OpenAI-compatible chat endpoint —
	// typically Plano's internal ingress (:12010) with the "kawai-vision" alias.
	// With no URL the VL provider stays nil and images fall back to OCR-only.
	vlURL := os.Getenv("FILEPROC_VL_URL")
	if vlURL == "" {
		vlURL = os.Getenv("PLANO_LLM_GATEWAY")
	}
	vlModel := os.Getenv("FILEPROC_VL_MODEL")
	if vlModel == "" {
		vlModel = "kawai-vision"
	}
	vlKey := os.Getenv("FILEPROC_VL_API_KEY")
	if vlKey == "" {
		vlKey = os.Getenv("MODEL_API_KEY")
	}
	vlInternalKey := os.Getenv("PLANO_INTERNAL_KEY")
	if vlURL != "" {
		slog.Info("fileproc: VL enabled", "model", vlModel, "url", vlURL)
	}

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
		cfg := fp.Config{FileStore: store, RAGProcessor: rag}

		// Build the VL client per-upload so we can stamp x-arch-actor-id with the
		// uploading user for Plano's internal-ingress billing. One client serves
		// both roles: image description (VLProvider) + OCR cleanup (LanguageModel).
		if vlURL != "" {
			vc := fp.NewOpenAIChatClient(vlURL, vlKey, vlModel)
			if vlInternalKey != "" {
				vc.SetHeader("x-arch-internal-key", vlInternalKey)
				vc.SetHeader("x-arch-actor-id", uid)
			}
			cfg.VLProvider = vc
			cfg.LanguageModel = vc
		}

		proc, err := fp.New(cfg)
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
