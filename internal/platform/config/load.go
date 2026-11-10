package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultJWTSecret     = "change-me-please-32chars!!"
	defaultAdminPassword = "Admin@123456"
)

type location struct {
	baseDir   string
	configDir string
	dataDir   string
	profile   string
	legacy    string
}

type fileConfig struct {
	App struct {
		Name      string `yaml:"name"`
		Profile   string `yaml:"profile"`
		BaseDir   string `yaml:"base_dir"`
		ConfigDir string `yaml:"config_dir"`
		DataDir   string `yaml:"data_dir"`
	} `yaml:"app"`
	Server struct {
		Addr              string   `yaml:"addr"`
		Mode              string   `yaml:"mode"`
		Host              string   `yaml:"host"`
		Port              int      `yaml:"port"`
		ReadHeaderTimeout string   `yaml:"read_header_timeout"`
		IdleTimeout       string   `yaml:"idle_timeout"`
		MaxHeaderBytes    int      `yaml:"max_header_bytes"`
		ShutdownTimeout   string   `yaml:"shutdown_timeout"`
		RequestTimeout    string   `yaml:"request_timeout"`
		SSETimeout        string   `yaml:"sse_timeout"`
		MaxBodyBytes      int64    `yaml:"max_body_bytes"`
		TrustedProxies    []string `yaml:"trusted_proxies"`
	} `yaml:"server"`
	Database struct {
		Enabled         *bool  `yaml:"enabled"`
		Driver          string `yaml:"driver"`
		Path            string `yaml:"path"`
		DSN             string `yaml:"dsn"`
		MaxOpenConns    int    `yaml:"max_open_conns"`
		MaxIdleConns    int    `yaml:"max_idle_conns"`
		ConnMaxLifetime string `yaml:"conn_max_lifetime"`
		BatchSize       int    `yaml:"batch_size"`
		SQLite          struct {
			Path        string `yaml:"path"`
			ForeignKeys *bool  `yaml:"foreign_keys"`
			BusyTimeout string `yaml:"busy_timeout"`
			JournalMode string `yaml:"journal_mode"`
		} `yaml:"sqlite"`
	} `yaml:"database"`
	Auth struct {
		Enabled    *bool  `yaml:"enabled"`
		JWTSecret  string `yaml:"jwtSecret"`
		JWTSecret2 string `yaml:"jwt_secret"`
		TokenTTL   string `yaml:"tokenTTL"`
		SessionTTL string `yaml:"session_ttl"`
	} `yaml:"auth"`
	ACME struct {
		DirectoryURL  string `yaml:"directoryURL"`
		DirectoryURL2 string `yaml:"directory_url"`
		Timeout       string `yaml:"timeout"`
	} `yaml:"acme"`
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
	Seed struct {
		AdminUser      string `yaml:"admin_user"`
		AdminPassword  string `yaml:"admin_password"`
		ACMEEmail      string `yaml:"acme_email"`
		BootstrapNginx *bool  `yaml:"bootstrap_nginx"`
	} `yaml:"seed"`
}

// Load 按「定位参数 → 基础配置 → profile → 环境变量 → 命令行」读取配置。
func Load() (*Config, error) {
	loc, err := resolveLocation()
	if err != nil {
		return nil, err
	}
	cfg := defaults(loc)
	sources := make([]string, 0, 4)
	if loc.legacy != "" {
		if err := applyFile(&cfg, loc.legacy); err != nil {
			return nil, err
		}
		sources = append(sources, loc.legacy)
	} else {
		base := filepath.Join(loc.configDir, "application.yaml")
		if _, err := os.Stat(base); err == nil {
			if err := applyFile(&cfg, base); err != nil {
				return nil, err
			}
			sources = append(sources, base)
		} else if legacy := filepath.Join(loc.configDir, "config.yaml"); fileExists(legacy) {
			if err := applyFile(&cfg, legacy); err != nil {
				return nil, err
			}
			sources = append(sources, legacy)
		}
		if loc.profile != "" {
			profileFile := filepath.Join(loc.configDir, "application-"+loc.profile+".yaml")
			if _, err := os.Stat(profileFile); err != nil {
				return nil, fmt.Errorf("缺少环境配置 %s", profileFile)
			}
			if err := applyFile(&cfg, profileFile); err != nil {
				return nil, err
			}
			sources = append(sources, profileFile)
		}
	}
	if err := applyEnv(&cfg); err != nil {
		return nil, err
	}
	if err := applyFlags(&cfg); err != nil {
		return nil, err
	}
	cfg.App.BaseDir = loc.baseDir
	cfg.App.ConfigDir = loc.configDir
	cfg.App.Profile = loc.profile
	if err := cfg.resolvePaths(loc); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.Sources = sources
	return &cfg, nil
}

func defaults(loc location) Config {
	return Config{
		App: App{
			Name:      "sslplatform",
			Profile:   loc.profile,
			BaseDir:   loc.baseDir,
			ConfigDir: loc.configDir,
			DataDir:   loc.dataDir,
		},
		Server: Server{
			Host:              "127.0.0.1",
			Port:              8080,
			Addr:              "127.0.0.1:8080",
			Mode:              "release",
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1048576,
			ShutdownTimeout:   30 * time.Second,
			RequestTimeout:    120 * time.Second,
			SSETimeout:        15 * time.Minute,
			MaxBodyBytes:      2097152,
			TrustedProxies:    []string{},
		},
		Database: Database{
			Enabled:         true,
			Driver:          "sqlite",
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: 30 * time.Minute,
			BatchSize:       100,
			SQLite: SQLite{
				Path:        "sslplatform.db",
				ForeignKeys: true,
				BusyTimeout: 5 * time.Second,
				JournalMode: "WAL",
			},
		},
		Auth: Auth{
			Enabled:   true,
			JWTSecret: defaultJWTSecret,
			TokenTTL:  24 * time.Hour,
		},
		ACME: ACME{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
			Timeout:      120 * time.Second,
		},
		Logging: Logging{Level: "info", Format: "text"},
		Seed: Seed{
			AdminUser:     "admin",
			AdminPassword: defaultAdminPassword,
			ACMEEmail:     "admin@example.com",
		},
	}
}

func resolveLocation() (location, error) {
	wd, err := os.Getwd()
	if err != nil {
		return location{}, err
	}
	base := absPath(wd)
	configDir := "configs"
	dataDir := "./data"
	profile := ""
	legacy := ""
	if !testingBinary() {
		fs := flag.NewFlagSet("sslplatform", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		baseFlag := fs.String("base-dir", "", "")
		configFlag := fs.String("config-dir", "", "")
		dataFlag := fs.String("data-dir", "", "")
		profileFlag := fs.String("profile", "", "")
		legacyFlag := fs.String("config", "", "")
		_ = fs.Parse(os.Args[1:])
		if *baseFlag != "" {
			base = *baseFlag
		}
		if *configFlag != "" {
			configDir = *configFlag
		}
		if *dataFlag != "" {
			dataDir = *dataFlag
		}
		if *profileFlag != "" {
			profile = *profileFlag
		}
		if *legacyFlag != "" {
			legacy = *legacyFlag
		}
	}
	if v := os.Getenv("APP_BASE_DIR"); v != "" {
		base = v
	}
	if v := os.Getenv("APP_CONFIG_DIR"); v != "" {
		configDir = v
	}
	if v := os.Getenv("APP_DATA_DIR"); v != "" {
		dataDir = v
	}
	if v := os.Getenv("APP_PROFILE"); v != "" {
		profile = v
	}
	if v := os.Getenv("SSL_CONFIG"); v != "" {
		legacy = v
	}
	base = absPath(base)
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(base, configDir)
	}
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(base, dataDir)
	}
	switch profile {
	case "", "dev", "pro":
	default:
		return location{}, fmt.Errorf("未知 profile %q，只允许 dev 或 pro", profile)
	}
	return location{baseDir: base, configDir: configDir, dataDir: dataDir, profile: profile, legacy: legacy}, nil
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
	if file.App.Name != "" {
		cfg.App.Name = file.App.Name
	}
	if file.App.DataDir != "" {
		cfg.App.DataDir = file.App.DataDir
	}
	if file.Server.Mode != "" {
		cfg.Server.Mode = file.Server.Mode
	}
	if file.Server.Host != "" {
		cfg.Server.Host = file.Server.Host
	}
	if file.Server.Port != 0 {
		cfg.Server.Port = file.Server.Port
	}
	if file.Server.Addr != "" {
		cfg.Server.Addr = file.Server.Addr
	} else if file.Server.Host != "" || file.Server.Port != 0 {
		cfg.Server.Addr = ""
	}
	if err := applyDuration(file.Server.ReadHeaderTimeout, &cfg.Server.ReadHeaderTimeout, "server.read_header_timeout"); err != nil {
		return err
	}
	if err := applyDuration(file.Server.IdleTimeout, &cfg.Server.IdleTimeout, "server.idle_timeout"); err != nil {
		return err
	}
	if file.Server.MaxHeaderBytes > 0 {
		cfg.Server.MaxHeaderBytes = file.Server.MaxHeaderBytes
	}
	if err := applyDuration(file.Server.ShutdownTimeout, &cfg.Server.ShutdownTimeout, "server.shutdown_timeout"); err != nil {
		return err
	}
	if err := applyDuration(file.Server.RequestTimeout, &cfg.Server.RequestTimeout, "server.request_timeout"); err != nil {
		return err
	}
	if err := applyDuration(file.Server.SSETimeout, &cfg.Server.SSETimeout, "server.sse_timeout"); err != nil {
		return err
	}
	if file.Server.MaxBodyBytes > 0 {
		cfg.Server.MaxBodyBytes = file.Server.MaxBodyBytes
	}
	if file.Server.TrustedProxies != nil {
		cfg.Server.TrustedProxies = file.Server.TrustedProxies
	}
	if file.Database.Enabled != nil {
		cfg.Database.Enabled = *file.Database.Enabled
	}
	if file.Database.Driver != "" {
		cfg.Database.Driver = file.Database.Driver
	}
	if file.Database.Path != "" {
		cfg.Database.Path = file.Database.Path
	}
	if file.Database.DSN != "" {
		cfg.Database.DSN = file.Database.DSN
	}
	if file.Database.MaxOpenConns > 0 {
		cfg.Database.MaxOpenConns = file.Database.MaxOpenConns
	}
	if file.Database.MaxIdleConns > 0 {
		cfg.Database.MaxIdleConns = file.Database.MaxIdleConns
	}
	if err := applyDuration(file.Database.ConnMaxLifetime, &cfg.Database.ConnMaxLifetime, "database.conn_max_lifetime"); err != nil {
		return err
	}
	if file.Database.BatchSize > 0 {
		cfg.Database.BatchSize = file.Database.BatchSize
	}
	if file.Database.SQLite.Path != "" {
		cfg.Database.SQLite.Path = file.Database.SQLite.Path
	}
	if file.Database.SQLite.ForeignKeys != nil {
		cfg.Database.SQLite.ForeignKeys = *file.Database.SQLite.ForeignKeys
	}
	if err := applyDuration(file.Database.SQLite.BusyTimeout, &cfg.Database.SQLite.BusyTimeout, "database.sqlite.busy_timeout"); err != nil {
		return err
	}
	if file.Database.SQLite.JournalMode != "" {
		cfg.Database.SQLite.JournalMode = file.Database.SQLite.JournalMode
	}
	if file.Auth.Enabled != nil {
		cfg.Auth.Enabled = *file.Auth.Enabled
	}
	if secret := firstNonEmpty(file.Auth.JWTSecret, file.Auth.JWTSecret2); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if ttl := firstNonEmpty(file.Auth.TokenTTL, file.Auth.SessionTTL); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return fmt.Errorf("auth.session_ttl: %w", err)
		}
		cfg.Auth.TokenTTL = d
	}
	if dir := firstNonEmpty(file.ACME.DirectoryURL, file.ACME.DirectoryURL2); dir != "" {
		cfg.ACME.DirectoryURL = dir
	}
	if err := applyDuration(file.ACME.Timeout, &cfg.ACME.Timeout, "acme.timeout"); err != nil {
		return err
	}
	if file.Logging.Level != "" {
		cfg.Logging.Level = file.Logging.Level
	}
	if file.Logging.Format != "" {
		cfg.Logging.Format = file.Logging.Format
	}
	if file.Seed.AdminUser != "" {
		cfg.Seed.AdminUser = file.Seed.AdminUser
	}
	if file.Seed.AdminPassword != "" {
		cfg.Seed.AdminPassword = file.Seed.AdminPassword
	}
	if file.Seed.ACMEEmail != "" {
		cfg.Seed.ACMEEmail = file.Seed.ACMEEmail
	}
	if file.Seed.BootstrapNginx != nil {
		cfg.Seed.BootstrapNginx = *file.Seed.BootstrapNginx
	}
	return nil
}

func applyEnv(cfg *Config) error {
	set := func(key string, dst *string) {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			*dst = v
		}
	}
	set("APP_NAME", &cfg.App.Name)
	set("APP_SERVER_HOST", &cfg.Server.Host)
	set("SSL_SERVER_MODE", &cfg.Server.Mode)
	set("APP_SERVER_MODE", &cfg.Server.Mode)
	set("APP_DATABASE_DRIVER", &cfg.Database.Driver)
	set("APP_DATABASE_DSN", &cfg.Database.DSN)
	set("APP_DATABASE_SQLITE_PATH", &cfg.Database.SQLite.Path)
	set("APP_DATABASE_SQLITE_JOURNAL_MODE", &cfg.Database.SQLite.JournalMode)
	set("APP_LOGGING_LEVEL", &cfg.Logging.Level)
	set("APP_LOGGING_FORMAT", &cfg.Logging.Format)
	set("SSL_JWT_SECRET", &cfg.Auth.JWTSecret)
	set("APP_AUTH_JWT_SECRET", &cfg.Auth.JWTSecret)
	set("SSL_ACME_DIRECTORY", &cfg.ACME.DirectoryURL)
	set("APP_ACME_DIRECTORY", &cfg.ACME.DirectoryURL)
	set("SSL_ADMIN_USER", &cfg.Seed.AdminUser)
	set("APP_SEED_ADMIN_USER", &cfg.Seed.AdminUser)
	set("SSL_ADMIN_PASSWORD", &cfg.Seed.AdminPassword)
	set("APP_SEED_ADMIN_PASSWORD", &cfg.Seed.AdminPassword)
	set("SSL_ACME_EMAIL", &cfg.Seed.ACMEEmail)
	set("APP_SEED_ACME_EMAIL", &cfg.Seed.ACMEEmail)
	if v := os.Getenv("SSL_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("APP_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("SSL_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("APP_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("APP_SERVER_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("APP_SERVER_PORT: %w", err)
		}
		cfg.Server.Port = n
		cfg.Server.Addr = ""
	}
	if err := envDuration("APP_SERVER_READ_HEADER_TIMEOUT", &cfg.Server.ReadHeaderTimeout); err != nil {
		return err
	}
	if err := envDuration("APP_SERVER_IDLE_TIMEOUT", &cfg.Server.IdleTimeout); err != nil {
		return err
	}
	if err := envDuration("APP_SERVER_SHUTDOWN_TIMEOUT", &cfg.Server.ShutdownTimeout); err != nil {
		return err
	}
	if err := envDuration("APP_SERVER_REQUEST_TIMEOUT", &cfg.Server.RequestTimeout); err != nil {
		return err
	}
	if err := envDuration("APP_SERVER_SSE_TIMEOUT", &cfg.Server.SSETimeout); err != nil {
		return err
	}
	if err := envDuration("APP_AUTH_SESSION_TTL", &cfg.Auth.TokenTTL); err != nil {
		return err
	}
	if err := envDuration("SSL_TOKEN_TTL", &cfg.Auth.TokenTTL); err != nil {
		return err
	}
	if err := envDuration("SSL_ACME_TIMEOUT", &cfg.ACME.Timeout); err != nil {
		return err
	}
	if err := envDuration("APP_ACME_TIMEOUT", &cfg.ACME.Timeout); err != nil {
		return err
	}
	if err := envDuration("APP_DATABASE_CONN_MAX_LIFETIME", &cfg.Database.ConnMaxLifetime); err != nil {
		return err
	}
	if err := envDuration("APP_DATABASE_SQLITE_BUSY_TIMEOUT", &cfg.Database.SQLite.BusyTimeout); err != nil {
		return err
	}
	if v := os.Getenv("APP_SERVER_MAX_HEADER_BYTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("APP_SERVER_MAX_HEADER_BYTES: %w", err)
		}
		cfg.Server.MaxHeaderBytes = n
	}
	if v := os.Getenv("APP_SERVER_MAX_BODY_BYTES"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("APP_SERVER_MAX_BODY_BYTES: %w", err)
		}
		cfg.Server.MaxBodyBytes = n
	}
	if v := os.Getenv("APP_SERVER_TRUSTED_PROXIES"); v != "" {
		var items []string
		if err := json.Unmarshal([]byte(v), &items); err != nil {
			return fmt.Errorf("APP_SERVER_TRUSTED_PROXIES 需要 JSON 数组: %w", err)
		}
		cfg.Server.TrustedProxies = items
	}
	if err := envBool("APP_DATABASE_ENABLED", &cfg.Database.Enabled); err != nil {
		return err
	}
	if err := envBool("APP_DATABASE_SQLITE_FOREIGN_KEYS", &cfg.Database.SQLite.ForeignKeys); err != nil {
		return err
	}
	if err := envBool("APP_AUTH_ENABLED", &cfg.Auth.Enabled); err != nil {
		return err
	}
	if err := envBool("SSL_BOOTSTRAP_NGINX", &cfg.Seed.BootstrapNginx); err != nil {
		return err
	}
	if err := envBool("APP_SEED_BOOTSTRAP_NGINX", &cfg.Seed.BootstrapNginx); err != nil {
		return err
	}
	if v := os.Getenv("APP_DATABASE_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("APP_DATABASE_MAX_OPEN_CONNS: %w", err)
		}
		cfg.Database.MaxOpenConns = n
	}
	if v := os.Getenv("APP_DATABASE_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("APP_DATABASE_MAX_IDLE_CONNS: %w", err)
		}
		cfg.Database.MaxIdleConns = n
	}
	if v := os.Getenv("APP_DATABASE_BATCH_SIZE"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("APP_DATABASE_BATCH_SIZE: %w", err)
		}
		cfg.Database.BatchSize = n
	}
	return nil
}

func applyFlags(cfg *Config) error {
	if testingBinary() {
		return nil
	}
	fs := flag.NewFlagSet("sslplatform-override", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("server-host", "", "")
	port := fs.Int("server-port", 0, "")
	_ = fs.Parse(os.Args[1:])
	if *host != "" {
		cfg.Server.Host = *host
		cfg.Server.Addr = ""
	}
	if *port != 0 {
		cfg.Server.Port = *port
		cfg.Server.Addr = ""
	}
	return nil
}

func (c *Config) resolvePaths(loc location) error {
	if c.App.DataDir == "" || c.App.DataDir == "./data" || !filepath.IsAbs(c.App.DataDir) {
		if filepath.IsAbs(c.App.DataDir) {
			c.App.DataDir = filepath.Clean(c.App.DataDir)
		} else if env := os.Getenv("APP_DATA_DIR"); env != "" {
			c.App.DataDir = loc.dataDir
		} else if c.App.DataDir == "" {
			c.App.DataDir = loc.dataDir
		} else {
			c.App.DataDir = absPath(filepath.Join(loc.baseDir, c.App.DataDir))
		}
	}
	if c.Database.Path != "" {
		if !filepath.IsAbs(c.Database.Path) {
			c.Database.Path = filepath.Join(loc.baseDir, c.Database.Path)
		}
		c.Database.Path = filepath.Clean(c.Database.Path)
	} else {
		sqlitePath := c.Database.SQLite.Path
		if sqlitePath == "" {
			sqlitePath = "sslplatform.db"
		}
		if filepath.IsAbs(sqlitePath) {
			c.Database.Path = filepath.Clean(sqlitePath)
		} else {
			c.Database.Path = filepath.Join(c.App.DataDir, sqlitePath)
		}
	}
	if c.Server.Addr == "" {
		c.Server.Addr = fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
	}
	return nil
}

func (c Config) validate() error {
	if c.Server.Addr == "" {
		return fmt.Errorf("server.addr 不能为空")
	}
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port 超出范围")
	}
	if !c.Database.Enabled {
		return fmt.Errorf("当前版本需要数据库")
	}
	switch c.Database.Driver {
	case "sqlite", "mysql", "postgres", "sqlserver":
	default:
		return fmt.Errorf("不支持的数据库驱动 %s", c.Database.Driver)
	}
	if c.Database.Driver == "sqlite" && c.Database.Path == "" {
		return fmt.Errorf("database.path 不能为空")
	}
	if c.Database.Driver != "sqlite" && strings.TrimSpace(c.Database.DSN) == "" {
		return fmt.Errorf("网络数据库必须设置 database.dsn")
	}
	if c.Database.MaxOpenConns < 1 || c.Database.MaxIdleConns < 1 || c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("数据库连接池参数无效")
	}
	if c.Database.BatchSize < 1 {
		return fmt.Errorf("database.batch_size 必须大于 0")
	}
	switch strings.ToUpper(c.Database.SQLite.JournalMode) {
	case "DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF":
	default:
		return fmt.Errorf("不支持的 journal_mode %s", c.Database.SQLite.JournalMode)
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
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("不支持的日志级别 %s", c.Logging.Level)
	}
	switch c.Logging.Format {
	case "text", "json":
	default:
		return fmt.Errorf("不支持的日志格式 %s", c.Logging.Format)
	}
	if c.Server.MaxBodyBytes < 1 || c.Server.ReadHeaderTimeout <= 0 || c.Server.IdleTimeout <= 0 {
		return fmt.Errorf("HTTP 超时或请求体限制无效")
	}
	return nil
}

func applyDuration(raw string, dst *time.Duration, name string) error {
	if raw == "" {
		return nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dst = d
	return nil
}

func envDuration(key string, dst *time.Duration) error {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	*dst = d
	return nil
}

func envBool(key string, dst *bool) error {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	*dst = b
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func testingBinary() bool {
	return strings.HasSuffix(os.Args[0], ".test") || strings.Contains(os.Args[0], "/_test/")
}
