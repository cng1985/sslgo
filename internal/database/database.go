package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sslplatform/internal/config"
)

// Open 打开 SQLite，并限制连接数以避免写锁。
func Open(cfg *config.Config) (*gorm.DB, error) {
	path := cfg.Database.Path
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("创建数据库目录: %w", err)
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.New(slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn), logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := os.Chmod(path, 0o600); err != nil {
		slog.Warn("设置数据库文件权限失败", "err", err)
	}
	return db, nil
}
