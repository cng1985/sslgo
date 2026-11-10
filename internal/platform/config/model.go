package config

import "time"

// Config 是进程启动后的只读配置。路径字段都已解析为最终值。
type Config struct {
	App      App
	Server   Server
	Database Database
	Auth     Auth
	ACME     ACME
	Logging  Logging
	Seed     Seed
	Sources  []string
}

type App struct {
	Name      string
	Profile   string
	BaseDir   string
	ConfigDir string
	DataDir   string
}

type Server struct {
	Host              string
	Port              int
	Addr              string
	Mode              string
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
	RequestTimeout    time.Duration
	SSETimeout        time.Duration
	MaxBodyBytes      int64
	TrustedProxies    []string
}

type Database struct {
	Enabled         bool
	Driver          string
	Path            string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	BatchSize       int
	SQLite          SQLite
}

type SQLite struct {
	Path        string
	ForeignKeys bool
	BusyTimeout time.Duration
	JournalMode string
}

type Auth struct {
	Enabled   bool
	JWTSecret string
	TokenTTL  time.Duration
}

type ACME struct {
	DirectoryURL string
	Timeout      time.Duration
}

type Logging struct {
	Level  string
	Format string
}

type Seed struct {
	AdminUser      string
	AdminPassword  string
	BootstrapNginx bool
	ACMEEmail      string
}

func (c Config) UsingDefaultSecret() bool {
	return c.Auth.JWTSecret == defaultJWTSecret
}

func (c Config) UsingDefaultPassword() bool {
	return c.Seed.AdminPassword == defaultAdminPassword
}
