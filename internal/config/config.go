package config

type ServerConfig struct {
	Config      Config `yaml:",inline"`
	AuthEnabled bool   `yaml:"auth_enabled"`
	MAXClient   int    `yaml:"max_client"`
}

type ClientConfig struct {
	Config   Config `yaml:",inline"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Addr string `yaml:"addr"`
}

// AuthConfig holds authentication related configuration.
type AuthConfig struct {
	// Type can be "none", "jwt", or "session"
	Type string `yaml:"type"`
	// Secret used to sign JWT tokens (HMAC)
	JWTSecret string `yaml:"jwt_secret"`
	// Access token TTL in seconds
	AccessTTLSeconds int `yaml:"access_ttl_seconds"`
	// Refresh token TTL in seconds (optional)
	RefreshTTLSeconds int `yaml:"refresh_ttl_seconds"`
	// Session store: "memory" or "redis"
	SessionStore string `yaml:"session_store"`
	// Redis address for session store (optional)
	RedisAddr string `yaml:"redis_addr"`
	// Password hash algorithm (e.g., "bcrypt")
	PasswordHashAlgo string `yaml:"password_hash_algo"`
}
