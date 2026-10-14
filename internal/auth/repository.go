package auth

import (
	"context"

	"gorm.io/gorm"

	"sslplatform/internal/common/apperror"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(user).Error, "用户不存在")
}

func (r *Repository) SaveUser(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *Repository) DeleteUser(ctx context.Context, tenantID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&User{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperror.New(404, "用户不存在")
		}
		return nil
	})
}

func (r *Repository) GetUser(ctx context.Context, tenantID, id uint) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&user).Error
	if err != nil {
		return nil, apperror.FromDB(err, "用户不存在")
	}
	return &user, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, apperror.FromDB(err, "用户不存在")
	}
	return &user, nil
}

func (r *Repository) ListUsers(ctx context.Context, tenantID uint, page, size int, keyword, status string) ([]User, int64, error) {
	q := r.db.WithContext(ctx).Model(&User{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR display_name LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []User
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) GetTenant(ctx context.Context, id uint) (*Tenant, error) {
	var tenant Tenant
	err := r.db.WithContext(ctx).First(&tenant, id).Error
	if err != nil {
		return nil, apperror.FromDB(err, "租户不存在")
	}
	return &tenant, nil
}

func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(role).Error, "角色不存在")
}

func (r *Repository) SaveRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *Repository) GetRole(ctx context.Context, tenantID, id uint) (*Role, error) {
	var role Role
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&role).Error
	if err != nil {
		return nil, apperror.FromDB(err, "角色不存在")
	}
	return &role, nil
}

func (r *Repository) ListRoles(ctx context.Context, tenantID uint, page, size int) ([]Role, int64, error) {
	q := r.db.WithContext(ctx).Model(&Role{}).Where("tenant_id = ?", tenantID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Role
	err := q.Order("id asc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) RolesByIDs(ctx context.Context, tenantID uint, ids []uint) ([]Role, error) {
	if len(ids) == 0 {
		return []Role{}, nil
	}
	var rows []Role
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id IN ?", tenantID, ids).Find(&rows).Error
	return rows, err
}

func (r *Repository) RolesOfUser(ctx context.Context, userID uint) ([]Role, error) {
	var rows []Role
	err := r.db.WithContext(ctx).
		Table("roles").
		Select("roles.*").
		Joins("join user_roles on user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.id asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		for _, id := range roleIDs {
			if err := tx.Create(&UserRole{UserID: userID, RoleID: id}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) CountAdminUsers(ctx context.Context, tenantID, exceptUserID uint) (int64, error) {
	q := r.db.WithContext(ctx).
		Table("users").
		Joins("join user_roles on user_roles.user_id = users.id").
		Joins("join roles on roles.id = user_roles.role_id").
		Where("users.tenant_id = ? AND roles.code = ? AND users.status = ?", tenantID, RoleAdmin, StatusActive)
	if exceptUserID > 0 {
		q = q.Where("users.id <> ?", exceptUserID)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	var rows []Permission
	err := r.db.WithContext(ctx).Order("resource asc, action asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) PermissionsByIDs(ctx context.Context, ids []uint) ([]Permission, error) {
	if len(ids) == 0 {
		return []Permission{}, nil
	}
	var rows []Permission
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *Repository) PermissionsOfRole(ctx context.Context, roleID uint) ([]Permission, error) {
	var rows []Permission
	err := r.db.WithContext(ctx).
		Table("permissions").
		Select("permissions.*").
		Joins("join role_permissions on role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.resource asc, permissions.action asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ReplaceRolePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		for _, id := range permissionIDs {
			if err := tx.Create(&RolePermission{RoleID: roleID, PermissionID: id}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) PermissionKeys(ctx context.Context, userID uint) (map[string]struct{}, bool, error) {
	type row struct {
		Resource string
		Action   string
		Code     string
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("permissions.resource as resource, permissions.action as action, roles.code as code").
		Joins("join roles on roles.id = user_roles.role_id").
		Joins("left join role_permissions on role_permissions.role_id = roles.id").
		Joins("left join permissions on permissions.id = role_permissions.permission_id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, false, err
	}
	set := make(map[string]struct{})
	admin := false
	for _, item := range rows {
		if item.Code == RoleAdmin {
			admin = true
		}
		if item.Resource != "" && item.Action != "" {
			set[item.Resource+":"+item.Action] = struct{}{}
		}
	}
	return set, admin, nil
}
