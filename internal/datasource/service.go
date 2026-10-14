package datasource

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/common/proc"
	"sslplatform/internal/config"
	"sslplatform/internal/paging"
)

type Service struct {
	repo *Repository
	cfg  *config.Config
	http *http.Client
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{
		repo: repo,
		cfg:  cfg,
		http: &http.Client{Timeout: 8 * time.Second},
	}
}

func (s *Service) Create(ctx context.Context, tenantID uint, req CreateRequest) (*View, error) {
	raw, err := s.normalize(req.Type, req.Config, "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.New(400, "请填写数据源名称")
	}
	item := &DataSource{
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Type:        req.Type,
		Description: req.Description,
		ConfigJSON:  raw,
		Status:      StatusActive,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return toView(item)
}

func (s *Service) Update(ctx context.Context, tenantID uint, req UpdateRequest) (*View, error) {
	item, err := s.repo.Get(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		item.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" || req.Description == "" {
		item.Description = req.Description
	}
	if req.Status != "" {
		if req.Status != StatusActive && req.Status != StatusDisabled {
			return nil, apperror.New(400, "状态只能是 active 或 disabled")
		}
		item.Status = req.Status
	}
	if req.Config != nil {
		raw, err := s.normalize(item.Type, req.Config, item.ConfigJSON)
		if err != nil {
			return nil, err
		}
		item.ConfigJSON = raw
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	return toView(item)
}

func (s *Service) Delete(ctx context.Context, tenantID, id uint) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) Detail(ctx context.Context, tenantID, id uint) (*View, error) {
	item, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return toView(item)
}

func (s *Service) List(ctx context.Context, tenantID uint, req ListRequest) ([]View, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.List(ctx, tenantID, page, size, req.Filters)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]View, 0, len(rows))
	for i := range rows {
		view, err := toView(&rows[i])
		if err != nil {
			return nil, 0, 0, 0, err
		}
		views = append(views, *view)
	}
	return views, page, size, total, nil
}

func (s *Service) Test(ctx context.Context, tenantID, id uint) (*TestView, error) {
	item, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	var view *TestView
	switch item.Type {
	case TypeACME:
		cfg, err := ParseACME(item.ConfigJSON)
		if err != nil {
			return nil, err
		}
		view, err = s.testACME(ctx, cfg)
		if err != nil {
			return nil, err
		}
	case TypeNginx:
		cfg, err := ParseNginx(item.ConfigJSON)
		if err != nil {
			return nil, err
		}
		view, err = testNginx(ctx, cfg)
		if err != nil {
			return nil, err
		}
	default:
		return nil, apperror.New(400, "未知数据源类型")
	}
	now := time.Now()
	item.LastTestAt = &now
	item.LastTestOK = view.OK
	item.LastTestMsg = view.Message
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) testACME(ctx context.Context, cfg ACMEConfig) (*TestView, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.DirectoryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return &TestView{OK: false, Message: "无法连接 ACME 目录: " + err.Error()}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "newAccount") {
		return &TestView{OK: false, Message: "ACME 目录响应异常"}, nil
	}
	msg := "ACME 目录可用"
	if strings.Contains(cfg.DirectoryURL, "staging") {
		msg += "（测试环境，浏览器不会信任该证书）"
	}
	return &TestView{OK: true, Message: msg}, nil
}

func testNginx(ctx context.Context, cfg NginxConfig) (*TestView, error) {
	if _, err := proc.Resolve(cfg.NginxBin); err != nil {
		return &TestView{OK: false, Message: err.Error()}, nil
	}
	info, err := os.Stat(cfg.Webroot)
	if err != nil || !info.IsDir() {
		return &TestView{OK: false, Message: "webroot 不存在或不是目录"}, nil
	}
	if info, err = os.Stat(cfg.CertDir); err != nil || !info.IsDir() {
		return &TestView{OK: false, Message: "证书目录不存在或不是目录"}, nil
	}
	if cfg.ServiceManager == ManagerSystemd {
		if _, err := proc.Resolve("systemctl"); err != nil {
			return &TestView{OK: false, Message: err.Error()}, nil
		}
	}
	if cfg.DryRun {
		return &TestView{OK: true, Message: "配置有效，当前为演练模式，不会重启 Nginx"}, nil
	}
	if cfg.TestBefore {
		if _, err := proc.Run(ctx, cfg.NginxBin, "-t"); err != nil {
			return &TestView{OK: false, Message: err.Error()}, nil
		}
	}
	return &TestView{OK: true, Message: "Nginx 配置检测通过"}, nil
}

func (s *Service) normalize(kind string, config map[string]any, previous string) (string, error) {
	if config == nil {
		return "", apperror.New(400, "请填写 config")
	}
	switch kind {
	case TypeACME:
		cfg, err := decodeACME(config)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(cfg.AccountKeyPEM) == "" && previous != "" {
			old, err := ParseACME(previous)
			if err == nil {
				cfg.AccountKeyPEM = old.AccountKeyPEM
				if cfg.RegistrationURI == "" {
					cfg.RegistrationURI = old.RegistrationURI
				}
			}
		}
		if err := validateACME(cfg, s.cfg.ACME.DirectoryURL); err != nil {
			return "", err
		}
		if cfg.DirectoryURL == "" {
			cfg.DirectoryURL = s.cfg.ACME.DirectoryURL
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	case TypeNginx:
		cfg, err := decodeNginx(config)
		if err != nil {
			return "", err
		}
		cfg, err = validateNginx(cfg)
		if err != nil {
			return "", err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	default:
		return "", apperror.New(400, "数据源类型只能是 acme 或 nginx")
	}
}

func decodeACME(config map[string]any) (ACMEConfig, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return ACMEConfig{}, err
	}
	var cfg ACMEConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ACMEConfig{}, apperror.New(400, "ACME 配置无效")
	}
	return cfg, nil
}

func decodeNginx(config map[string]any) (NginxConfig, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return NginxConfig{}, err
	}
	var cfg NginxConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return NginxConfig{}, apperror.New(400, "Nginx 配置无效")
	}
	return cfg, nil
}

func validateACME(cfg ACMEConfig, fallback string) error {
	if !strings.Contains(cfg.Email, "@") || strings.ContainsAny(cfg.Email, " \t\r\n") {
		return apperror.New(400, "请填写有效的 ACME 邮箱")
	}
	directory := cfg.DirectoryURL
	if directory == "" {
		directory = fallback
	}
	u, err := url.Parse(directory)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return apperror.New(400, "ACME 目录地址无效")
	}
	return nil
}

func validateNginx(cfg NginxConfig) (NginxConfig, error) {
	resolved, err := proc.Resolve(cfg.NginxBin)
	if err != nil {
		return cfg, apperror.New(400, err.Error())
	}
	cfg.NginxBin = resolved
	if cfg.Webroot, err = proc.SafeDir(cfg.Webroot); err != nil {
		return cfg, apperror.New(400, "webroot "+err.Error())
	}
	if cfg.CertDir, err = proc.SafeDir(cfg.CertDir); err != nil {
		return cfg, apperror.New(400, "certDir "+err.Error())
	}
	if cfg.SnippetDir != "" {
		if cfg.SnippetDir, err = proc.SafeDir(cfg.SnippetDir); err != nil {
			return cfg, apperror.New(400, "snippetDir "+err.Error())
		}
	}
	switch cfg.ServiceManager {
	case ManagerSystemd:
		if !proc.ValidService(cfg.ServiceName) {
			return cfg, apperror.New(400, "systemd 服务名无效")
		}
	case ManagerSignal:
	default:
		return cfg, apperror.New(400, "serviceManager 只能是 systemd 或 signal")
	}
	if cfg.Action != ActionRestart && cfg.Action != ActionReload {
		return cfg, apperror.New(400, "action 只能是 restart 或 reload")
	}
	return cfg, nil
}

// ParseACME 解析已保存的 ACME 配置。
func ParseACME(raw string) (ACMEConfig, error) {
	var cfg ACMEConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ACMEConfig{}, apperror.New(400, "ACME 配置无法解析")
	}
	return cfg, nil
}

// ParseNginx 解析已保存的 Nginx 配置。
func ParseNginx(raw string) (NginxConfig, error) {
	var cfg NginxConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return NginxConfig{}, apperror.New(400, "Nginx 配置无法解析")
	}
	return cfg, nil
}

func toView(item *DataSource) (*View, error) {
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(item.ConfigJSON), &cfg); err != nil {
		return nil, err
	}
	if _, ok := cfg["accountKeyPEM"]; ok {
		cfg["accountKeyStored"] = true
		delete(cfg, "accountKeyPEM")
	}
	return &View{
		ID: item.ID, Name: item.Name, Type: item.Type, Description: item.Description,
		Status: item.Status, Config: cfg, LastTestOK: item.LastTestOK, LastTestMsg: item.LastTestMsg,
		LastTestAt: item.LastTestAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

// SanitizeConfigJSON 给通用查询去掉账户私钥。
func SanitizeConfigJSON(raw string) string {
	var cfg map[string]any
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return raw
	}
	if _, ok := cfg["accountKeyPEM"]; ok {
		cfg["accountKeyStored"] = true
		delete(cfg, "accountKeyPEM")
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return raw
	}
	return string(b)
}
