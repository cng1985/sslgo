package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sslplatform/internal/platform/config"
)

// Ready 表示迁移和初始化已经完成。HTTP 与调度器依赖它，避免先对外服务。
type Ready struct{}

// Open 打开配置指定的数据库。未知驱动或连接失败直接返回错误，不回退到 SQLite。
func Open(lc fx.Lifecycle, cfg *config.Config) (*gorm.DB, error) {
	if !cfg.Database.Enabled {
		return nil, fmt.Errorf("数据库未启用")
	}
	gormCfg := &gorm.Config{
		Logger: logger.New(slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn), logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	}
	var dialector gorm.Dialector
	switch cfg.Database.Driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o750); err != nil {
			return nil, fmt.Errorf("创建数据库目录: %w", err)
		}
		dialector = sqlite.Open(sqliteDSN(cfg))
	case "mysql", "postgres", "sqlserver":
		return nil, fmt.Errorf("驱动 %s 已列入计划支持，当前发行版只验证了 SQLite", cfg.Database.Driver)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动 %s", cfg.Database.Driver)
	}
	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	if cfg.Database.Driver == "sqlite" {
		if err := os.Chmod(cfg.Database.Path, 0o600); err != nil && !os.IsNotExist(err) {
			slog.Warn("设置数据库文件权限失败", "err", err)
		}
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		conn, err := db.DB()
		if err != nil {
			return err
		}
		return conn.Close()
	}})
	return db, nil
}

func sqliteDSN(cfg *config.Config) string {
	busy := cfg.Database.SQLite.BusyTimeout.Milliseconds()
	if busy <= 0 {
		busy = 5000
	}
	fk := 0
	if cfg.Database.SQLite.ForeignKeys {
		fk = 1
	}
	mode := strings.ToUpper(cfg.Database.SQLite.JournalMode)
	if mode == "" {
		mode = "WAL"
	}
	return fmt.Sprintf("file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(%s)&_pragma=foreign_keys(%d)", cfg.Database.Path, busy, mode, fk)
}

// AutoMigrate 执行注册模型的结构迁移。
func AutoMigrate(db *gorm.DB, models []any) error {
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`).Error
}

// MarkSchema 记录已经应用的结构版本，重复执行不覆盖原时间。
func MarkSchema(db *gorm.DB, version string) error {
	return db.Exec(`INSERT INTO schema_migrations(version, applied_at)
		SELECT ?, CURRENT_TIMESTAMP
		WHERE NOT EXISTS (SELECT 1 FROM schema_migrations WHERE version = ?)`, version, version).Error
}
