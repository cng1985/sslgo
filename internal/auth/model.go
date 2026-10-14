package auth

import "sslplatform/internal/entity"

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

type Tenant struct {
	entity.Base
	Name   string `gorm:"size:128;not null" json:"name"`
	Code   string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Status string `gorm:"size:32;not null" json:"status"`
}

func (Tenant) TableName() string { return "tenants" }

type User struct {
	entity.Base
	TenantID           uint   `gorm:"index;not null" json:"tenantId"`
	Username           string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash       string `gorm:"size:255;not null" json:"-"`
	DisplayName        string `gorm:"size:128" json:"displayName"`
	Status             string `gorm:"size:32;not null" json:"status"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func (User) TableName() string { return "users" }

type Role struct {
	entity.Base
	TenantID    uint   `gorm:"not null;uniqueIndex:idx_role_code" json:"tenantId"`
	Code        string `gorm:"size:64;not null;uniqueIndex:idx_role_code" json:"code"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	BuiltIn     bool   `json:"builtIn"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Resource string `gorm:"size:64;not null;uniqueIndex:idx_perm" json:"resource"`
	Action   string `gorm:"size:64;not null;uniqueIndex:idx_perm" json:"action"`
	Name     string `gorm:"size:128;not null" json:"name"`
}

func (Permission) TableName() string { return "permissions" }

func (p Permission) Key() string { return p.Resource + ":" + p.Action }

type UserRole struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null;uniqueIndex:idx_user_role" json:"userId"`
	RoleID uint `gorm:"not null;uniqueIndex:idx_user_role" json:"roleId"`
}

func (UserRole) TableName() string { return "user_roles" }

type RolePermission struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	RoleID       uint `gorm:"not null;uniqueIndex:idx_role_perm" json:"roleId"`
	PermissionID uint `gorm:"not null;uniqueIndex:idx_role_perm" json:"permissionId"`
}

func (RolePermission) TableName() string { return "role_permissions" }

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type UserCreateRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	RoleIDs     []uint `json:"roleIds"`
}

type UserUpdateRequest struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	Password    string `json:"password"`
}

type UserDeleteRequest struct {
	ID uint `json:"id"`
}

type UserListRequest struct {
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Filters UserListFilter `json:"filters"`
}

type UserListFilter struct {
	Keyword string `json:"keyword"`
	Status  string `json:"status"`
}

type UserAssignRoleRequest struct {
	UserID  uint   `json:"userId"`
	RoleIDs []uint `json:"roleIds"`
}

type RoleCreateRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleUpdateRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleListRequest struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

type RoleDetailRequest struct {
	ID uint `json:"id"`
}

type RoleGrantRequest struct {
	RoleID        uint   `json:"roleId"`
	PermissionIDs []uint `json:"permissionIds"`
}

type RoleBrief struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type UserView struct {
	ID                 uint        `json:"id"`
	Username           string      `json:"username"`
	DisplayName        string      `json:"displayName"`
	Status             string      `json:"status"`
	MustChangePassword bool        `json:"mustChangePassword"`
	Roles              []RoleBrief `json:"roles"`
	CreatedAt          any         `json:"createdAt"`
}

type RoleView struct {
	ID          uint         `json:"id"`
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	BuiltIn     bool         `json:"builtIn"`
	Permissions []Permission `json:"permissions"`
}

type ProfileView struct {
	User        UserView    `json:"user"`
	Tenant      Tenant      `json:"tenant"`
	Roles       []RoleBrief `json:"roles"`
	Permissions []string    `json:"permissions"`
	TokenTTLSec int64       `json:"tokenTtlSeconds"`
}

type LoginView struct {
	Token    string      `json:"token"`
	ExpireAt string      `json:"expireAt"`
	Profile  ProfileView `json:"profile"`
}
