package config

import "fmt"

type DBConfig struct {
	Host     string `yaml:"host"      mapstructure:"host"`
	Port     int    `yaml:"port"      mapstructure:"port"`
	User     string `yaml:"user"      mapstructure:"user"`
	Password string `yaml:"password"  mapstructure:"password"`
	Name     string `yaml:"name"      mapstructure:"name"`
	SSLMode  string `yaml:"sslmode"   mapstructure:"sslmode"`
	TZ       string `yaml:"timezone"  mapstructure:"timezone"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%d sslmode=%s TimeZone=%s",
		d.Host, d.User, d.Name, d.Password, d.Port, d.SSLMode, d.TZ)
}

type ServerConfig struct {
	Socket     string   `yaml:"socket"       mapstructure:"socket"`
	MaxClients int      `yaml:"max_clients"  mapstructure:"max_clients"`
	DB         DBConfig `yaml:"db"           mapstructure:"db"`
}

type ClientConfig struct {
	Socket   string `yaml:"socket"   mapstructure:"socket"`
	Username string `yaml:"username" mapstructure:"username"`
}
