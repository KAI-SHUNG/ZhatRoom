package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func setServerDefaults(v *viper.Viper) {
	v.SetDefault("socket", "/tmp/zhatroom.sock")
	v.SetDefault("max_clients", 100)
	v.SetDefault("db.host", "127.0.0.1")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.user", "postgres")
	v.SetDefault("db.name", "zhat_db")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("db.timezone", "Asia/Shanghai")
}

func setClientDefaults(v *viper.Viper) {
	v.SetDefault("socket", "/tmp/zhatroom.sock")
	v.SetDefault("username", "Anonymous")
}

func Load(path string, target any) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ZHATROOM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	switch target.(type) {
	case *ServerConfig:
		setServerDefaults(v)
	case *ClientConfig:
		setClientDefaults(v)
	}

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}
