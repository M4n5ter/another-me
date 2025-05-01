package db

import (
	"log/slog"

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

// SurrealDBConfig 配置
type SurrealDBConfig struct {
	DBURL     string `json:"db_url"`    // 数据库 URL
	Namespace string `json:"namespace"` // 命名空间
	Database  string `json:"database"`  // 数据库
	Username  string `json:"username"`  // 用户名
	Password  string `json:"password"`  // 密码
}
