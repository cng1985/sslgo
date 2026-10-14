package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultJWTSecret = "change-me-please-32chars!!"

type fileConfig struct {
	Server struct {
		Addr string `yaml:"addr"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Auth struct {
		JWTSecret string `yaml:"jwtSecret"`
		TokenTTL  string `yaml:"tokenTTL"`
	} `yaml:"auth"`
	ACME struct {
		DirectoryURL string `yaml:"directoryURL"`
		Timeout      string `yaml:"timeout"`
	} `yaml:"acme"`
}

// Load 读取配置文件，再用环境变量覆盖。main 不解析配置。
func Load() (*Config, error) {
	cfg := defaults()
	path := os.Getenv("SSL_CONFIG")
	if path == "" {
		if _, err := os.Stat("configs/config.yaml"); err == nil {
			path = "configs/config.yaml"
		}
	}
	if path != "" {
		if err := applyFile(&cfg, path); err != nil {
			return nil, err
		}
	}
	if err := applyEnv(&cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func defaults() Config {
	return Config{
		Server:   Server{Addr: ":8080", Mode: "release"},
		Database: Database{Path: "data/sslplatform.db"},
		Auth:     Auth{JWTSecret: defaultJWTSecret, TokenTTL: 24 * time.Hour},
		ACME: ACME{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
			Timeout:      120 * time.Second,
		},
		Seed: Seed{
			AdminUser:     "admin",
			AdminPassword: "Admin@123456",
			ACMEEmail:     "admin@example.com",
		},
	}
}

func applyFile(cfg *Config, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置 %s: %w", path, err)
	}
	var file fileConfig
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("解析配置 %s: %w", path, err)
	}
	if file.Server.Addr != "" {
		cfg.Server.Addr = file.Server.Addr
	}
	if file.Server.Mode != "" {
		cfg.Server.Mode = file.Server.Mode
	}
	if file.Database.Path != "" {
		cfg.Database.Path = file.Database.Path
	}
	if file.Auth.JWTSecret != "" {
		cfg.Auth.JWTSecret = file.Auth.JWTSecret
	}
	if file.Auth.TokenTTL != "" {
		d, err := time.ParseDuration(file.Auth.TokenTTL)
		if err != nil {
			return fmt.Errorf("auth.tokenTTL: %w", err)
		}
		cfg.Auth.TokenTTL = d
	}
	if file.ACME.DirectoryURL != "" {
		cfg.ACME.DirectoryURL = file.ACME.DirectoryURL
	}
	if file.ACME.Timeout != "" {
		d, err := time.ParseDuration(file.ACME.Timeout)
		if err != nil {
			return fmt.Errorf("acme.timeout: %w", err)
		}
		cfg.ACME.Timeout = d
	}
	return nil
}

func applyEnv(cfg *Config) error {
	set := func(key string, dst *string) {
		if v := os.Getenv(key); v != "" {
			*dst = v
		}
	}
	set("SSL_SERVER_ADDR", &cfg.Server.Addr)
	set("SSL_SERVER_MODE", &cfg.Server.Mode)
	set("SSL_DATABASE_PATH", &cfg.Database.Path)
	set("SSL_JWT_SECRET", &cfg.Auth.JWTSecret)
	set("SSL_ACME_DIRECTORY", &cfg.ACME.DirectoryURL)
	set("SSL_ADMIN_USER", &cfg.Seed.AdminUser)
	set("SSL_ADMIN_PASSWORD", &cfg.Seed.AdminPassword)
	set("SSL_ACME_EMAIL", &cfg.Seed.ACMEEmail)
	if v := os.Getenv("SSL_TOKEN_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("SSL_TOKEN_TTL: %w", err)
		}
		cfg.Auth.TokenTTL = d
	}
	if v := os.Getenv("SSL_ACME_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("SSL_ACME_TIMEOUT: %w", err)
		}
		cfg.ACME.Timeout = d
	}
	if v := os.Getenv("SSL_BOOTSTRAP_NGINX"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("SSL_BOOTSTRAP_NGINX: %w", err)
		}
		cfg.Seed.BootstrapNginx = b
	}
	return nil
}

func (c Config) validate() error {
	if c.Server.Addr == "" {
		return fmt.Errorf("server.addr 不能为空")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path 不能为空")
	}
	if len(c.Auth.JWTSecret) < 16 {
		return fmt.Errorf("jwt secret 至少 16 个字符")
	}
	if c.Auth.TokenTTL <= 0 {
		return fmt.Errorf("tokenTTL 必须大于 0")
	}
	if c.Seed.AdminUser == "" || len(c.Seed.AdminPassword) < 8 {
		return fmt.Errorf("初始管理员账号或密码无效")
	}
	return nil
}
