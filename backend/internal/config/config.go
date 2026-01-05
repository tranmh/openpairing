package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	CORS        CORSConfig
	Federation  FederationConfig
	Pairing     PairingConfig
	RateLimit   RateLimitConfig
	Email       EmailConfig
	App         AppConfig
	WebPush     WebPushConfig
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
	SkipVerify   bool
}

type AppConfig struct {
	BaseURL string
}

type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret           string
	AccessExpiry     time.Duration
	RefreshExpiry    time.Duration
	Issuer           string
	RefreshSecret    string
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

type FederationConfig struct {
	CacheDir         string
	EnableFIDESync   bool
	EnableDSBSync    bool
	EnableOESBSync   bool
	EnableSSBSync    bool
	FideSyncInterval time.Duration
	DSBSyncInterval  time.Duration
	OESBSyncInterval time.Duration
	SSBSyncInterval  time.Duration
	DSBAPIKey        string
	OESBAPIKey       string
	SSBAPIKey        string

	// Portal64 integration (alternative DSB data source)
	DSBProvider        string        // "portal64" or "dewis" (default: "portal64")
	Portal64BaseURL    string        // Portal64 API base URL
	Portal64Timeout    time.Duration // HTTP timeout
	Portal64CacheTTL   time.Duration // Player cache TTL
	Portal64SearchTTL  time.Duration // Search results cache TTL
}

type PairingConfig struct {
	JaVaFoPath string
	BBPPath    string
	Timeout    time.Duration
}

type RateLimitConfig struct {
	Enabled bool
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", ""),
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "openpairing"),
			Password:        getEnv("DB_PASSWORD", "openpairing"),
			Name:            getEnv("DB_NAME", "openpairing"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "change-me-in-production"),
			AccessExpiry:  getDurationEnv("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
			Issuer:        getEnv("JWT_ISSUER", "openpairing"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-in-production"),
		},
		CORS: CORSConfig{
			AllowedOrigins:   []string{getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
			AllowCredentials: true,
		},
		Federation: FederationConfig{
			CacheDir:         getEnv("FEDERATION_CACHE_DIR", "/tmp/openpairing_federation"),
			EnableFIDESync:   getBoolEnv("FEDERATION_ENABLE_FIDE", false),
			EnableDSBSync:    getBoolEnv("FEDERATION_ENABLE_DSB", false),
			EnableOESBSync:   getBoolEnv("FEDERATION_ENABLE_OESB", false),
			EnableSSBSync:    getBoolEnv("FEDERATION_ENABLE_SSB", false),
			FideSyncInterval: getDurationEnv("FEDERATION_FIDE_INTERVAL", 24*time.Hour),
			DSBSyncInterval:  getDurationEnv("FEDERATION_DSB_INTERVAL", 7*24*time.Hour),
			OESBSyncInterval: getDurationEnv("FEDERATION_OESB_INTERVAL", 7*24*time.Hour),
			SSBSyncInterval:  getDurationEnv("FEDERATION_SSB_INTERVAL", 7*24*time.Hour),
			DSBAPIKey:        getEnv("FEDERATION_DSB_API_KEY", ""),
			OESBAPIKey:       getEnv("FEDERATION_OESB_API_KEY", ""),
			SSBAPIKey:        getEnv("FEDERATION_SSB_API_KEY", ""),

			// Portal64 integration
			DSBProvider:       getEnv("FEDERATION_DSB_PROVIDER", "portal64"),
			Portal64BaseURL:   getEnv("FEDERATION_PORTAL64_BASE_URL", "https://test.svw.info:8080"),
			Portal64Timeout:   getDurationEnv("FEDERATION_PORTAL64_TIMEOUT", 30*time.Second),
			Portal64CacheTTL:  getDurationEnv("FEDERATION_PORTAL64_CACHE_TTL", 1*time.Hour),
			Portal64SearchTTL: getDurationEnv("FEDERATION_PORTAL64_SEARCH_TTL", 15*time.Minute),
		},
		Pairing: PairingConfig{
			JaVaFoPath: getEnv("JAVAFO_PATH", ""),
			BBPPath:    getEnv("BBP_PATH", ""),
			Timeout:    getDurationEnv("PAIRING_TIMEOUT", 5*time.Minute),
		},
		RateLimit: RateLimitConfig{
			Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "localhost"),
			SMTPPort:     getIntEnv("SMTP_PORT", 1025),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("SMTP_FROM_EMAIL", "noreply@openpairing.com"),
			FromName:     getEnv("SMTP_FROM_NAME", "OpenPairing"),
			UseTLS:       getBoolEnv("SMTP_USE_TLS", false),
			SkipVerify:   getBoolEnv("SMTP_SKIP_VERIFY", false),
		},
		App: AppConfig{
			BaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),
		},
		WebPush: WebPushConfig{
			VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
			VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
			VAPIDSubject:    getEnv("VAPID_SUBJECT", "mailto:admin@openpairing.com"),
		},
	}
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.Name + "?sslmode=" + c.SSLMode
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
