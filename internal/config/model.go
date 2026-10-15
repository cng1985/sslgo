package config

import "time"

// Config 是进程启动后的只读配置。
type Config struct {
	Server   Server
	Database Database
	Auth     Auth
	ACME     ACME
	Seed     Seed
}

type Server struct {
	Addr string
	Mode string
}

type Database struct {
	Path string
}

type Auth struct {
	JWTSecret string
	TokenTTL  time.Duration
}

type ACME struct {
	DirectoryURL string
	Timeout      time.Duration
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
