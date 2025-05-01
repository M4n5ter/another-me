package conf

import (
	"log/slog"
	"os"
	"strings"

	"github.com/m4n5ter/another-me/src/core/db"
	"github.com/spf13/viper"
)

var required = []string{
	"surreal.db_url",
}

func init() {
	viper.SetEnvPrefix("ANOTHER_ME")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetConfigName("another-me")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	// Unix-like systems
	viper.AddConfigPath("$HOME/.config/another-me")
	viper.AddConfigPath("/etc/another-me")
	viper.AddConfigPath("/usr/local/etc/another-me")
	// Windows
	viper.AddConfigPath("$HOME\\.config\\another-me")
	viper.AddConfigPath("C:\\ProgramData\\another-me")
	viper.AddConfigPath("C:\\Users\\Administrator\\AppData\\Local\\another-me")

	err := viper.ReadInConfig()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	for _, key := range required {
		if !viper.IsSet(key) {
			slog.Error("required config key not set", "key", key)
			os.Exit(1)
		}
	}
}

func GetSurrealDBConfig() *db.SurrealDBConfig {
	return &db.SurrealDBConfig{
		DBURL:     viper.GetString("surreal.db_url"),
		Namespace: viper.GetString("surreal.namespace"),
		Database:  viper.GetString("surreal.database"),
		Username:  viper.GetString("surreal.username"),
		Password:  viper.GetString("surreal.password"),
	}
}

func GetLocale() string {
	return viper.GetString("locale")
}
