package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/surrealdb/surrealdb.go"
)

type SurrealDBClient struct {
	db         *surrealdb.DB
	invalidate func() error // 注销 token

	logger *slog.Logger
}

// NewSurrealDBClient 创建一个 SurrealDB 客户端
func NewSurrealDBClient(config *SurrealDBConfig) (*SurrealDBClient, error) {
	logger := slog.Default().WithGroup("surrealdb")

	db, err := surrealdb.New(config.DBURL)
	if err != nil {
		return nil, err
	}

	// 使用默认的命名空间和数据库
	if config.Namespace != "" && config.Database != "" {
		err = db.Use(config.Namespace, config.Database)
		if err != nil {
			return nil, err
		}
	}

	authData := &surrealdb.Auth{
		Username:  config.Username,
		Password:  config.Password,
		Database:  config.Database,
		Namespace: config.Namespace,
	}
	token, err := db.SignIn(authData)
	if err != nil {
		return nil, err
	}

	err = db.Authenticate(token)
	if err != nil {
		return nil, err
	}

	return &SurrealDBClient{
		db: db,
		invalidate: func() error {
			return db.Invalidate()
		},
		logger: logger,
	}, nil
}

// Terminate 注销 token 并关闭连接
func (c *SurrealDBClient) Terminate() {
	err := c.invalidate()
	if err != nil {
		c.logger.Error("failed to invalidate", "error", err)
	}

	err = c.db.Close()
	if err != nil {
		c.logger.Error("failed to close", "error", err)
	}
}

// Use 使用命名空间和数据库
func (c *SurrealDBClient) Use(namespace, database string) error {
	return c.db.Use(namespace, database)
}

// DB 返回 SurrealDB 实例
func (c *SurrealDBClient) DB() *surrealdb.DB {
	return c.db
}

const (
	typ                   = "SurrealDB"
	defaultCollection     = "another_me_collection"
	defaultVectorField    = "vector"
	defaultTopK           = 5
	defaultScoreThreshold = 0
)

// SurrealDBConfig 配置
type SurrealDBConfig struct {
	DBURL     string `json:"db_url"`    // 数据库 URL
	Namespace string `json:"namespace"` // 命名空间
	Database  string `json:"database"`  // 数据库
	Username  string `json:"username"`  // 用户名
	Password  string `json:"password"`  // 密码
}

// ---- 向量检索器 ----

// SurrealDBRetriever 向量检索器，T 为查询结果的类型
type SurrealDBRetriever[T any] struct {
	db     *surrealdb.DB
	config SurrealDBRetrieverConfig[T]
}

type SurrealDBRetrieverConfig[T any] struct {
	// Default Retriever config
	// Collection is the collection name in the surrealdb database
	// Optional, and the default value is "another_me_collection"
	Collection string
	// VectorField is the vector field name in the collection
	// Optional, and the default value is "vector"
	VectorField string
	// DocumentConverter is the function to convert the search result to schema.Document
	// Optional, and the default value is defaultDocumentConverter
	DocumentConverter func(ctx context.Context, doc surrealdb.QueryResult[T]) ([]*schema.Document, error)
	// TopK is the top k results to be returned
	// Optional, and the default value is 5
	TopK int
	// ScoreThreshold is the threshold for the search result
	// Optional, and the default value is 0
	ScoreThreshold float64

	// Embedding is the embedding vectorization method for values needs to be embedded from schema.Document's content.
	// Required
	Embedding embedding.Embedder
}

func NewSurrealDBRetriever[T any](db *surrealdb.DB, config SurrealDBRetrieverConfig[T]) *SurrealDBRetriever[T] {
	if config.Collection == "" {
		config.Collection = defaultCollection
	}
	if config.VectorField == "" {
		config.VectorField = defaultVectorField
	}
	if config.TopK == 0 {
		config.TopK = defaultTopK
	}
	if config.ScoreThreshold == 0 {
		config.ScoreThreshold = defaultScoreThreshold
	}
	if config.DocumentConverter == nil {
		defaultDocumentConverter := func(ctx context.Context, doc surrealdb.QueryResult[T]) ([]*schema.Document, error) {
			content, err := json.Marshal(doc.Result)
			if err != nil {
				return nil, err
			}
			return []*schema.Document{
				{ID: uuid.New().String(), Content: string(content), MetaData: make(map[string]any)},
			}, nil
		}
		config.DocumentConverter = defaultDocumentConverter
	}
	return &SurrealDBRetriever[T]{
		db:     db,
		config: config,
	}
}

var _ retriever.Retriever = (*SurrealDBRetriever[any])(nil)

// Retrieve implements retriever.Retriever.
func (r *SurrealDBRetriever[T]) Retrieve(ctx context.Context, query string, opts ...retriever.Option) (docs []*schema.Document, err error) {
	// get common options
	co := retriever.GetCommonOptions(&retriever.Options{
		Index:          &r.config.VectorField,
		TopK:           &r.config.TopK,
		ScoreThreshold: &r.config.ScoreThreshold,
		Embedding:      r.config.Embedding,
	}, opts...)
	// get impl specific options
	io := retriever.GetImplSpecificOptions(&ImplOptions{}, opts...)

	ctx = callbacks.EnsureRunInfo(ctx, r.GetType(), components.ComponentOfRetriever)
	// callback info on start
	ctx = callbacks.OnStart(ctx, &retriever.CallbackInput{
		Query:          query,
		TopK:           *co.TopK,
		Filter:         io.Filter,
		ScoreThreshold: co.ScoreThreshold,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	// get the embedding vector
	emb := co.Embedding
	if emb == nil {
		return nil, fmt.Errorf("[surrealdb retriever] embedding not provided")
	}

	// embedding the query
	vectors, err := emb.EmbedStrings(r.makeEmbeddingCtx(ctx, emb), []string{query})
	if err != nil {
		return nil, fmt.Errorf("[surrealdb retriever] embedding has error: %w", err)
	}
	// check the embedding result
	if len(vectors) != 1 {
		return nil, fmt.Errorf("[surrealdb retriever] invalid return length of vector, got=%d, expected=1", len(vectors))
	}

	results, err := surrealdb.Query[T](r.db,
		"SELECT * FROM ( SELECT *, vector::similarity::cosine($vector_field, $query_embeddings) AS score FROM $collection WHERE $vector_field <|$k|> $query_embeddings ) WHERE score > $score_threshold ORDER BY score DESC",
		map[string]any{"collection": r.config.Collection, "vector_field": r.config.VectorField, "query_embeddings": vectors[0], "k": r.config.TopK, "score_threshold": r.config.ScoreThreshold})
	if err != nil {
		return nil, fmt.Errorf("[surrealdb retriever] vector search has error: %w", err)
	}
	// check the vector search result
	if len(*results) == 0 {
		return nil, fmt.Errorf("[surrealdb retriever] no results found")
	}

	// convert the vector search result to schema.Document
	documents := make([]*schema.Document, 0, len(*results))
	for _, result := range *results {
		if strings.ToLower(result.Status) != "ok" {
			return nil, fmt.Errorf("[surrealdb retriever] vector search has error: %s", result.Status)
		}

		document, err := r.config.DocumentConverter(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("[surrealdb retriever] failed to convert vector search result to schema.Document: %w", err)
		}
		documents = append(documents, document...)
	}

	// callback info on end
	callbacks.OnEnd(ctx, &retriever.CallbackOutput{Docs: documents})

	return documents, nil
}

func (r *SurrealDBRetriever[T]) GetType() string {
	return typ
}

// makeEmbeddingCtx makes the embedding context
func (r *SurrealDBRetriever[T]) makeEmbeddingCtx(ctx context.Context, emb embedding.Embedder) context.Context {
	runInfo := &callbacks.RunInfo{
		Component: components.ComponentOfEmbedding,
	}

	if embType, ok := components.GetType(emb); ok {
		runInfo.Type = embType
	}

	runInfo.Name = runInfo.Type + string(runInfo.Component)

	return callbacks.ReuseHandlers(ctx, runInfo)
}

type ImplOptions struct {
	// Filter is the filter for the search
	// Optional, and the default value is empty
	Filter string
}

func WithFilter(filter string) retriever.Option {
	return retriever.WrapImplSpecificOptFn(func(o *ImplOptions) {
		o.Filter = filter
	})
}

// TODO: ---- 向量索引器 ----
type SurrealDBIndexer struct {
	db     *surrealdb.DB
	config SurrealDBIndexerConfig
}

type SurrealDBIndexerConfig struct {
	Collection string

	Embedding embedding.Embedder
}

func NewSurrealDBIndexer(db *surrealdb.DB, config SurrealDBIndexerConfig) *SurrealDBIndexer {
	if config.Collection == "" {
		config.Collection = defaultCollection
	}

	return &SurrealDBIndexer{db: db, config: config}
}

func (i *SurrealDBIndexer) GetType() string {
	return typ
}

var _ indexer.Indexer = (*SurrealDBIndexer)(nil)

func (i *SurrealDBIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	return nil, nil
}
