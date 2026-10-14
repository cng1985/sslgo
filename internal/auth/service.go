package auth

import (
	"context"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/config"
	"sslplatform/internal/paging"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{2,31}$`)

type Service struct {
	repo *Repository
	cfg  *config.Config
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

type claims struct {
	UserID   uint   `json:"uid"`
	TenantID uint   `json:"tid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginView, error) {
	if req.Username == "" || req.Password == "" {
		return nil, apperror.New(400, "请填写用户名和密码")
	}
	user, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, apperror.New(401, "用户名或密码错误")
	}
	if user.Status != StatusActive {
		return nil, apperror.New(403, "账号已停用")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, apperror.New(401, "用户名或密码错误")
	}
	tenant, err := s.repo.GetTenant(ctx, user.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status != StatusActive {
		return nil, apperror.New(403, "租户已停用")
	}
	profile, err := s.profileOf(ctx, user, tenant)
	if err != nil {
		return nil, err
	}
	expire := time.Now().Add(s.cfg.Auth.TokenTTL)
	token, err := s.sign(user, expire)
	if err != nil {
		return nil, err
	}
	return &LoginView{Token: token, ExpireAt: expire.UTC().Format(time.RFC3339), Profile: *profile}, nil
}

func (s *Service) Profile(ctx context.Context, userID, tenantID uint) (*ProfileView, error) {
	user, err := s.repo.GetUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.profileOf(ctx, user, tenant)
}

func (s *Service) ChangePassword(ctx context.Context, userID, tenantID uint, req ChangePasswordRequest) error {
	if len(req.NewPassword) < 8 {
		return apperror.New(400, "新密码至少 8 位")
	}
	if req.NewPassword == req.OldPassword {
		return apperror.New(400, "新密码不能与旧密码相同")
	}
	user, err := s.repo.GetUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		return apperror.New(400, "旧密码不正确")
	}
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.MustChangePassword = false
	return s.repo.SaveUser(ctx, user)
}

func (s *Service) CreateUser(ctx context.Context, tenantID uint, req UserCreateRequest) (*UserView, error) {
	if !usernameRe.MatchString(req.Username) {
		return nil, apperror.New(400, "用户名需以字母开头，长度 3 到 32，只能包含字母数字._-")
	}
	if len(req.Password) < 8 {
		return nil, apperror.New(400, "密码至少 8 位")
	}
	roles, err := s.mustRoles(ctx, tenantID, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &User{
		TenantID:     tenantID,
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  defaultName(req.DisplayName, req.Username),
		Status:       StatusActive,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, user.ID, req.RoleIDs); err != nil {
		return nil, err
	}
	return toUserView(user, roles), nil
}

func (s *Service) UpdateUser(ctx context.Context, actorID, tenantID uint, req UserUpdateRequest) (*UserView, error) {
	user, err := s.repo.GetUser(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if req.Status != "" && req.Status != StatusActive && req.Status != StatusDisabled {
		return nil, apperror.New(400, "状态只能是 active 或 disabled")
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Password != "" {
		if len(req.Password) < 8 {
			return nil, apperror.New(400, "密码至少 8 位")
		}
		hash, err := hashPassword(req.Password)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
		user.MustChangePassword = false
	}
	if req.Status != "" && req.Status != user.Status {
		if err := s.guardLastAdmin(ctx, tenantID, user, req.Status != StatusActive); err != nil {
			return nil, err
		}
		user.Status = req.Status
	}
	if actorID == user.ID && user.Status != StatusActive {
		return nil, apperror.New(400, "不能停用当前登录账号")
	}
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, err
	}
	roles, err := s.repo.RolesOfUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return toUserView(user, roles), nil
}

func (s *Service) DeleteUser(ctx context.Context, actorID, tenantID, id uint) error {
	if actorID == id {
		return apperror.New(400, "不能删除当前登录账号")
	}
	user, err := s.repo.GetUser(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := s.guardLastAdmin(ctx, tenantID, user, true); err != nil {
		return err
	}
	return s.repo.DeleteUser(ctx, tenantID, id)
}

func (s *Service) ListUsers(ctx context.Context, tenantID uint, req UserListRequest) ([]UserView, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.ListUsers(ctx, tenantID, page, size, req.Filters.Keyword, req.Filters.Status)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]UserView, 0, len(rows))
	for i := range rows {
		roles, err := s.repo.RolesOfUser(ctx, rows[i].ID)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		views = append(views, *toUserView(&rows[i], roles))
	}
	return views, page, size, total, nil
}

func (s *Service) AssignRoles(ctx context.Context, tenantID uint, req UserAssignRoleRequest) (*UserView, error) {
	user, err := s.repo.GetUser(ctx, tenantID, req.UserID)
	if err != nil {
		return nil, err
	}
	roles, err := s.mustRoles(ctx, tenantID, req.RoleIDs)
	if err != nil {
		return nil, err
	}
	keepsAdmin := false
	for _, role := range roles {
		if role.Code == RoleAdmin {
			keepsAdmin = true
		}
	}
	if !keepsAdmin {
		if err := s.guardLastAdmin(ctx, tenantID, user, true); err != nil {
			return nil, err
		}
	}
	if err := s.repo.ReplaceUserRoles(ctx, user.ID, req.RoleIDs); err != nil {
		return nil, err
	}
	return toUserView(user, roles), nil
}

func (s *Service) CreateRole(ctx context.Context, tenantID uint, req RoleCreateRequest) (*RoleView, error) {
	if !usernameRe.MatchString(req.Code) {
		return nil, apperror.New(400, "角色编码需以字母开头，长度 3 到 32")
	}
	if isBuiltIn(req.Code) {
		return nil, apperror.New(409, "不能使用内置角色编码")
	}
	if req.Name == "" {
		return nil, apperror.New(400, "请填写角色名称")
	}
	role := &Role{TenantID: tenantID, Code: req.Code, Name: req.Name, Description: req.Description}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	return &RoleView{ID: role.ID, Code: role.Code, Name: role.Name, Description: role.Description, Permissions: []Permission{}}, nil
}

func (s *Service) UpdateRole(ctx context.Context, tenantID uint, req RoleUpdateRequest) (*RoleView, error) {
	role, err := s.repo.GetRole(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	role.Description = req.Description
	if err := s.repo.SaveRole(ctx, role); err != nil {
		return nil, err
	}
	return s.roleView(ctx, role)
}

func (s *Service) ListRoles(ctx context.Context, tenantID uint, req RoleListRequest) ([]RoleView, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.ListRoles(ctx, tenantID, page, size)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]RoleView, 0, len(rows))
	for i := range rows {
		view, err := s.roleView(ctx, &rows[i])
		if err != nil {
			return nil, 0, 0, 0, err
		}
		views = append(views, *view)
	}
	return views, page, size, total, nil
}

func (s *Service) RoleDetail(ctx context.Context, tenantID, id uint) (*RoleView, error) {
	role, err := s.repo.GetRole(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.roleView(ctx, role)
}

func (s *Service) Grant(ctx context.Context, tenantID uint, req RoleGrantRequest) (*RoleView, error) {
	role, err := s.repo.GetRole(ctx, tenantID, req.RoleID)
	if err != nil {
		return nil, err
	}
	if role.Code == RoleAdmin {
		return nil, apperror.New(400, "管理员角色始终拥有全部权限")
	}
	perms, err := s.repo.PermissionsByIDs(ctx, req.PermissionIDs)
	if err != nil {
		return nil, err
	}
	if len(perms) != len(uniqueUints(req.PermissionIDs)) {
		return nil, apperror.New(400, "包含不存在的权限")
	}
	if err := s.repo.ReplaceRolePermissions(ctx, role.ID, req.PermissionIDs); err != nil {
		return nil, err
	}
	return s.roleView(ctx, role)
}

func (s *Service) ListPermissionCatalog(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) Allowed(ctx context.Context, userID uint, resource, action string) (bool, error) {
	set, admin, err := s.repo.PermissionKeys(ctx, userID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	_, ok := set[resource+":"+action]
	return ok, nil
}

func (s *Service) sign(user *User, expire time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			ExpiresAt: jwt.NewNumericDate(expire),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func (s *Service) parse(token string) (*claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, apperror.New(401, "登录态无效")
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, apperror.New(401, "登录态无效或已过期")
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid || c.UserID == 0 || c.TenantID == 0 {
		return nil, apperror.New(401, "登录态无效")
	}
	return c, nil
}

func (s *Service) profileOf(ctx context.Context, user *User, tenant *Tenant) (*ProfileView, error) {
	roles, err := s.repo.RolesOfUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	set, admin, err := s.repo.PermissionKeys(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	perms := make([]string, 0, len(set))
	if admin {
		for _, item := range PermissionCatalog() {
			perms = append(perms, item.Key())
		}
	} else {
		for key := range set {
			perms = append(perms, key)
		}
	}
	return &ProfileView{
		User:        *toUserView(user, roles),
		Tenant:      *tenant,
		Roles:       briefs(roles),
		Permissions: perms,
		TokenTTLSec: int64(s.cfg.Auth.TokenTTL.Seconds()),
	}, nil
}

func (s *Service) roleView(ctx context.Context, role *Role) (*RoleView, error) {
	var perms []Permission
	var err error
	if role.Code == RoleAdmin {
		perms, err = s.repo.ListPermissions(ctx)
	} else {
		perms, err = s.repo.PermissionsOfRole(ctx, role.ID)
	}
	if err != nil {
		return nil, err
	}
	if perms == nil {
		perms = []Permission{}
	}
	return &RoleView{
		ID: role.ID, Code: role.Code, Name: role.Name, Description: role.Description,
		BuiltIn: role.BuiltIn, Permissions: perms,
	}, nil
}

func (s *Service) mustRoles(ctx context.Context, tenantID uint, ids []uint) ([]Role, error) {
	ids = uniqueUints(ids)
	roles, err := s.repo.RolesByIDs(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(ids) {
		return nil, apperror.New(400, "包含不存在的角色")
	}
	return roles, nil
}

func (s *Service) guardLastAdmin(ctx context.Context, tenantID uint, user *User, removing bool) error {
	if !removing {
		return nil
	}
	roles, err := s.repo.RolesOfUser(ctx, user.ID)
	if err != nil {
		return err
	}
	isAdmin := false
	for _, role := range roles {
		if role.Code == RoleAdmin {
			isAdmin = true
		}
	}
	if !isAdmin || user.Status != StatusActive {
		return nil
	}
	left, err := s.repo.CountAdminUsers(ctx, tenantID, user.ID)
	if err != nil {
		return err
	}
	if left == 0 {
		return apperror.New(400, "不能移除租户内最后一个管理员")
	}
	return nil
}

func toUserView(user *User, roles []Role) *UserView {
	return &UserView{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Status: user.Status, MustChangePassword: user.MustChangePassword,
		Roles: briefs(roles), CreatedAt: user.CreatedAt,
	}
}

func briefs(roles []Role) []RoleBrief {
	out := make([]RoleBrief, 0, len(roles))
	for _, role := range roles {
		out = append(out, RoleBrief{ID: role.ID, Code: role.Code, Name: role.Name})
	}
	return out
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func defaultName(name, username string) string {
	if name == "" {
		return username
	}
	return name
}

func uniqueUints(ids []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
