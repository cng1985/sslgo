package database

import (
	"encoding/json"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"sslplatform/internal/auth"
	"sslplatform/internal/config"
	"sslplatform/internal/datasource"
	"sslplatform/internal/etl"
	"sslplatform/internal/scheduler"
	"sslplatform/internal/task"
)

// Ready 表示表结构已经迁移完成。启动日志依赖它，保证迁移行先打出来。
type Ready struct{}

// MigrateAndSeed 创建表结构、权限点和首个管理员。
func MigrateAndSeed(db *gorm.DB, cfg *config.Config) (Ready, error) {
	if err := db.AutoMigrate(
		&auth.Tenant{},
		&auth.User{},
		&auth.Role{},
		&auth.Permission{},
		&auth.UserRole{},
		&auth.RolePermission{},
		&datasource.DataSource{},
		&etl.Certificate{},
		&etl.ETLJob{},
		&task.Task{},
		&scheduler.Schedule{},
	); err != nil {
		return Ready{}, err
	}
	slog.Info("数据库迁移完成")
	if err := ensurePermissions(db); err != nil {
		return Ready{}, err
	}
	if err := syncBuiltinGrants(db); err != nil {
		return Ready{}, err
	}
	var users int64
	if err := db.Model(&auth.User{}).Count(&users).Error; err != nil {
		return Ready{}, err
	}
	if users == 0 {
		if err := seedAdmin(db, cfg); err != nil {
			return Ready{}, err
		}
	}
	if cfg.Seed.BootstrapNginx {
		if err := bootstrapNginx(db, cfg); err != nil {
			return Ready{}, err
		}
	}
	return Ready{}, nil
}

func ensurePermissions(db *gorm.DB) error {
	for _, item := range auth.PermissionCatalog() {
		var existing auth.Permission
		err := db.Where("resource = ? AND action = ?", item.Resource, item.Action).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&item).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// syncBuiltinGrants 给已有的内置角色补上新版本增加的权限。
func syncBuiltinGrants(db *gorm.DB) error {
	var roles []auth.Role
	if err := db.Where("built_in = ?", true).Find(&roles).Error; err != nil {
		return err
	}
	var perms []auth.Permission
	if err := db.Find(&perms).Error; err != nil {
		return err
	}
	byKey := map[string]uint{}
	for _, perm := range perms {
		byKey[perm.Key()] = perm.ID
	}
	for _, role := range roles {
		var existing []auth.RolePermission
		if err := db.Where("role_id = ?", role.ID).Find(&existing).Error; err != nil {
			return err
		}
		have := map[uint]struct{}{}
		for _, item := range existing {
			have[item.PermissionID] = struct{}{}
		}
		for key := range auth.BuiltinGrants(role.Code) {
			pid, ok := byKey[key]
			if !ok {
				continue
			}
			if _, ok := have[pid]; ok {
				continue
			}
			if err := db.Create(&auth.RolePermission{RoleID: role.ID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedAdmin(db *gorm.DB, cfg *config.Config) error {
	return db.Transaction(func(tx *gorm.DB) error {
		tenant := auth.Tenant{Name: "默认租户", Code: "default", Status: auth.StatusActive}
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		var perms []auth.Permission
		if err := tx.Find(&perms).Error; err != nil {
			return err
		}
		byKey := map[string]uint{}
		for _, p := range perms {
			byKey[p.Key()] = p.ID
		}
		roles := []auth.Role{
			{TenantID: tenant.ID, Code: auth.RoleAdmin, Name: "管理员", Description: "当前租户全部权限", BuiltIn: true},
			{TenantID: tenant.ID, Code: auth.RoleOperator, Name: "运维", Description: "单独申请或部署证书，并管理调度", BuiltIn: true},
			{TenantID: tenant.ID, Code: auth.RoleViewer, Name: "只读", Description: "查看证书、任务和流水线", BuiltIn: true},
		}
		for i := range roles {
			if err := tx.Create(&roles[i]).Error; err != nil {
				return err
			}
			grants := auth.BuiltinGrants(roles[i].Code)
			for key := range grants {
				pid, ok := byKey[key]
				if !ok {
					continue
				}
				if err := tx.Create(&auth.RolePermission{RoleID: roles[i].ID, PermissionID: pid}).Error; err != nil {
					return err
				}
			}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Seed.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := auth.User{
			TenantID:           tenant.ID,
			Username:           cfg.Seed.AdminUser,
			PasswordHash:       string(hash),
			DisplayName:        "管理员",
			Status:             auth.StatusActive,
			MustChangePassword: true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&auth.UserRole{UserID: user.ID, RoleID: roles[0].ID}).Error; err != nil {
			return err
		}
		slog.Info("已初始化默认租户和管理员", "username", user.Username, "tenant", tenant.Code)
		return nil
	})
}

func bootstrapNginx(db *gorm.DB, cfg *config.Config) error {
	var n int64
	if err := db.Model(&datasource.DataSource{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var tenant auth.Tenant
	if err := db.Where("code = ?", "default").First(&tenant).Error; err != nil {
		return err
	}
	acmeRaw, _ := json.Marshal(datasource.ACMEConfig{
		Email:        cfg.Seed.ACMEEmail,
		DirectoryURL: cfg.ACME.DirectoryURL,
	})
	nginxRaw, _ := json.Marshal(datasource.NginxConfig{
		NginxBin:       "/usr/sbin/nginx",
		ServiceManager: "signal",
		ServiceName:    "nginx",
		Action:         "restart",
		Webroot:        "/var/www/acme",
		CertDir:        "/etc/nginx/ssl",
		SnippetDir:     "/etc/nginx/conf.d",
		TestBefore:     true,
	})
	rows := []datasource.DataSource{
		{TenantID: tenant.ID, Name: "Let's Encrypt", Type: datasource.TypeACME, Description: "容器内默认 ACME 账户", ConfigJSON: string(acmeRaw), Status: datasource.StatusActive},
		{TenantID: tenant.ID, Name: "本机 Nginx", Type: datasource.TypeNginx, Description: "容器内 Nginx，签发后重启服务", ConfigJSON: string(nginxRaw), Status: datasource.StatusActive},
	}
	return db.Create(&rows).Error
}
