package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadServerConfig(t *testing.T) {
	path := writeTemp(t, `
socket: /run/zhatroom.sock
max_clients: 50
db:
  host: 10.0.0.1
  port: 5433
  user: admin
  password: secret
  name: mydb
  sslmode: require
  timezone: UTC
`)
	var cfg ServerConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Socket != "/run/zhatroom.sock" {
		t.Errorf("socket = %q, want /run/zhatroom.sock", cfg.Socket)
	}
	if cfg.MaxClients != 50 {
		t.Errorf("max_clients = %d, want 50", cfg.MaxClients)
	}
	if cfg.DB.Host != "10.0.0.1" {
		t.Errorf("db.host = %q, want 10.0.0.1", cfg.DB.Host)
	}
	if cfg.DB.Port != 5433 {
		t.Errorf("db.port = %d, want 5433", cfg.DB.Port)
	}
	if cfg.DB.Password != "secret" {
		t.Errorf("db.password = %q, want secret", cfg.DB.Password)
	}
}

func TestLoadClientConfig(t *testing.T) {
	path := writeTemp(t, `
socket: /run/zhatroom.sock
username: alice
`)
	var cfg ClientConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Socket != "/run/zhatroom.sock" {
		t.Errorf("socket = %q, want /run/zhatroom.sock", cfg.Socket)
	}
	if cfg.Username != "alice" {
		t.Errorf("username = %q, want alice", cfg.Username)
	}
}

func TestDefaultsApply(t *testing.T) {
	path := writeTemp(t, `
db:
  password: zhatroom
`)
	var cfg ServerConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Socket != "/tmp/zhatroom.sock" {
		t.Errorf("default socket = %q, want /tmp/zhatroom.sock", cfg.Socket)
	}
	if cfg.MaxClients != 100 {
		t.Errorf("default max_clients = %d, want 100", cfg.MaxClients)
	}
	if cfg.DB.Host != "127.0.0.1" {
		t.Errorf("default db.host = %q, want 127.0.0.1", cfg.DB.Host)
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("default db.port = %d, want 5432", cfg.DB.Port)
	}
	if cfg.DB.User != "postgres" {
		t.Errorf("default db.user = %q, want postgres", cfg.DB.User)
	}
	if cfg.DB.SSLMode != "disable" {
		t.Errorf("default db.sslmode = %q, want disable", cfg.DB.SSLMode)
	}
	if cfg.DB.TZ != "Asia/Shanghai" {
		t.Errorf("default db.timezone = %q, want Asia/Shanghai", cfg.DB.TZ)
	}
}

func TestEnvVarOverride(t *testing.T) {
	path := writeTemp(t, `
db:
  password: zhatroom
`)
	t.Setenv("ZHATROOM_DB_HOST", "192.168.1.100")
	t.Setenv("ZHATROOM_MAX_CLIENTS", "200")

	var cfg ServerConfig
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "192.168.1.100" {
		t.Errorf("db.host = %q, want 192.168.1.100 (env override)", cfg.DB.Host)
	}
	if cfg.MaxClients != 200 {
		t.Errorf("max_clients = %d, want 200 (env override)", cfg.MaxClients)
	}
}

func TestMissingConfigFile(t *testing.T) {
	var cfg ServerConfig
	err := Load("/nonexistent/config.yaml", &cfg)
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestDBConfigDSN(t *testing.T) {
	db := DBConfig{
		Host:     "10.0.0.1",
		Port:     5433,
		User:     "admin",
		Password: "secret",
		Name:     "mydb",
		SSLMode:  "require",
		TZ:       "UTC",
	}
	want := "host=10.0.0.1 user=admin dbname=mydb password=secret port=5433 sslmode=require TimeZone=UTC"
	if got := db.DSN(); got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}
