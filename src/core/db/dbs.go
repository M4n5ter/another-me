package db

import (
	"log/slog"
	"os"
	"sync"
)

var dbs *DBs
var once sync.Once

type DBs struct {
	// SurrealDB
	surreal      *SurrealDBClient
	surrealMutex sync.Mutex

	// SQLite(sqlite-vec)
	// sqlite      *SQLiteClient
	// sqliteMutex sync.Mutex
}

func SetDBs(config *SurrealDBConfig) {
	once.Do(func() {
		surreal, err := NewSurrealDBClient(config)
		if err != nil {
			slog.Error("failed to create global db client", "error", err)
			return
		}
		dbs = &DBs{surreal: surreal}
	})
}

// Surreal 返回 SurrealDB 客户端单例
func Surreal() *SurrealDBClient {
	dbs.surrealMutex.Lock()
	defer dbs.surrealMutex.Unlock()
	surreal := dbs.surreal
	if surreal == nil {
		slog.Error("surreal db client is not set")
		os.Exit(1)
	}

	return surreal
}
