package datasource

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"sslplatform/internal/acme"
	"sslplatform/internal/common/apperror"
	"sslplatform/internal/common/proc"
	"sslplatform/internal/config"
	"sslplatform/internal/paging"
)

type Service struct {
	repo     *Repository
	cfg      *config.Config
	http     *http.Client
	register func(context.Context, acme.RegisterRequest) (*acme.RegisterResult, error)
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{
		repo:     repo,
		cfg:      cfg,
		http:     &http.Client{Timeout: 8 * time.Second},
		register: acme.Register,
	}
}

func (s *Service) Create(ctx context.Context, tenantID uint, req CreateRequest) (*View, error) {
	raw, register, err := s.normalize(req.Type, req.Config, "")
	if err != nil {
		return nil, err
	}
	if register {
		raw, err = s.registerACME(ctx, raw)
		if err != nil {
			return nil, err
		}
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
		raw, register, err := s.normalize(item.Type, req.Config, item.ConfigJSON)
		if err != nil {
			return nil, err
		}
		if register {
			raw, err = s.registerACME(ctx, raw)
			if err != nil {
				return nil, err
			}
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
	if strings.Contains(cfg.DirectoryURL, "staging") || strings.Contains(cfg.DirectoryURL, "test4.buypass") {
		msg += "（测试环境，浏览器不会信任该证书）"
	}
	if cfg.RegistrationURI != "" {
		msg += "，账户已注册"
	} else {
		msg += "，账户尚未注册"
	}
	if cfg.DNSProvider != "" {
		msg += "，已绑定" + acme.DNSName(cfg.DNSProvider)
	}
	return &TestView{OK: true, Message: msg}, nil
}

// Register 向证书机构注册已保存的 ACME 账户。
func (s *Service) Register(ctx context.Context, tenantID uint, req RegisterAccountRequest) (*View, error) {
	item, err := s.repo.Get(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if item.Type != TypeACME {
		return nil, apperror.New(400, "只有 ACME 数据源可以注册账户")
	}
	cfg, err := ParseACME(item.ConfigJSON)
	if err != nil {
		return nil, err
	}
	if req.AgreeTOS {
		cfg.TermsAgreed = true
	}
	if !cfg.TermsAgreed {
		return nil, apperror.New(400, "必须同意 ACME 服务条款")
	}
	if cfg.RegistrationURI != "" {
		return toView(item)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	rawText, err := s.registerACME(ctx, string(raw))
	if err != nil {
		return nil, err
	}
	item.ConfigJSON = rawText
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	return toView(item)
}

func (s *Service) registerACME(ctx context.Context, raw string) (string, error) {
	cfg, err := ParseACME(raw)
	if err != nil {
		return "", err
	}
	if !cfg.TermsAgreed {
		return "", apperror.New(400, "注册 ACME 账户前必须同意服务条款")
	}
	if acme.RequiresEAB(cfg.CA, cfg.DirectoryURL) && (strings.TrimSpace(cfg.EABKID) == "" || strings.TrimSpace(cfg.EABHMACKey) == "") {
		return "", apperror.New(400, "该证书机构需要外部账号绑定，请填写 EAB Kid 和 HMAC Key")
	}
	if s.register == nil {
		return "", apperror.New(500, "ACME 注册器未配置")
	}
	result, err := s.register(ctx, acme.RegisterRequest{
		Email:         cfg.Email,
		DirectoryURL:  cfg.DirectoryURL,
		AccountKeyPEM: cfg.AccountKeyPEM,
		EABKID:        cfg.EABKID,
		EABHMACKey:    cfg.EABHMACKey,
		Timeout:       s.cfg.ACME.Timeout,
	})
	if err != nil {
		return "", apperror.New(502, "注册 ACME 账户失败："+err.Error())
	}
	cfg.AccountKeyPEM = result.AccountKeyPEM
	cfg.RegistrationURI = result.RegistrationURI
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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

func (s *Service) normalize(kind string, config map[string]any, previous string) (string, bool, error) {
	if config == nil {
		return "", false, apperror.New(400, "请填写 config")
	}
	switch kind {
	case TypeACME:
		cfg, register, err := s.normalizeACME(config, previous)
		if err != nil {
			return "", false, err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", false, err
		}
		return string(raw), register, nil
	case TypeNginx:
		cfg, err := decodeNginx(config)
		if err != nil {
			return "", false, err
		}
		cfg, err = validateNginx(cfg)
		if err != nil {
			return "", false, err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return "", false, err
		}
		return string(raw), false, nil
	default:
		return "", false, apperror.New(400, "数据源类型只能是 acme 或 nginx")
	}
}

func (s *Service) normalizeACME(config map[string]any, previous string) (ACMEConfig, bool, error) {
	register := boolValue(config["register"])
	delete(config, "register")
	_, providerSet := config["dnsProvider"]
	cfg, err := decodeACME(config)
	if err != nil {
		return ACMEConfig{}, false, err
	}
	var old ACMEConfig
	if previous != "" {
		if old, err = ParseACME(previous); err != nil {
			return ACMEConfig{}, false, err
		}
		cfg = mergeACME(cfg, old, providerSet)
	}
	ca, directory, _, err := acme.ResolveCA(cfg.CA, cfg.DirectoryURL, s.cfg.ACME.DirectoryURL)
	if err != nil {
		return ACMEConfig{}, false, apperror.New(400, err.Error())
	}
	cfg.CA = ca
	cfg.DirectoryURL = directory
	if previous != "" && strings.TrimRight(old.DirectoryURL, "/") != strings.TrimRight(cfg.DirectoryURL, "/") {
		cfg.RegistrationURI = ""
	}
	cfg.Email = strings.TrimSpace(cfg.Email)
	if !strings.Contains(cfg.Email, "@") || strings.ContainsAny(cfg.Email, " \t\r\n") {
		return ACMEConfig{}, false, apperror.New(400, "请填写有效的 ACME 邮箱")
	}
	cfg.DNSProvider = strings.TrimSpace(cfg.DNSProvider)
	if cfg.DNSProvider == "none" {
		cfg.DNSProvider = ""
	}
	cfg.DNS = trimDNS(cfg.DNS)
	if cfg.DNSProvider == "" {
		cfg.DNS = nil
	}
	if err := acme.ValidateDNS(cfg.DNSProvider, cfg.DNS); err != nil {
		return ACMEConfig{}, false, apperror.New(400, err.Error())
	}
	if strings.TrimSpace(cfg.AccountKeyPEM) == "" {
		key, err := acme.GenerateAccountKeyPEM()
		if err != nil {
			return ACMEConfig{}, false, err
		}
		cfg.AccountKeyPEM = key
	}
	cfg.EABKID = strings.TrimSpace(cfg.EABKID)
	cfg.EABHMACKey = strings.TrimSpace(cfg.EABHMACKey)
	return cfg, register, nil
}

func mergeACME(cfg, old ACMEConfig, providerSet bool) ACMEConfig {
	if strings.TrimSpace(cfg.Email) == "" {
		cfg.Email = old.Email
	}
	if strings.TrimSpace(cfg.CA) == "" && strings.TrimSpace(cfg.DirectoryURL) == "" {
		cfg.CA = old.CA
		cfg.DirectoryURL = old.DirectoryURL
	}
	newKey := strings.TrimSpace(cfg.AccountKeyPEM)
	oldKey := strings.TrimSpace(old.AccountKeyPEM)
	keyChanged := newKey != "" && oldKey != "" && newKey != oldKey
	if newKey == "" {
		cfg.AccountKeyPEM = old.AccountKeyPEM
	}
	if strings.TrimSpace(cfg.EABHMACKey) == "" {
		cfg.EABHMACKey = old.EABHMACKey
	}
	if strings.TrimSpace(cfg.EABKID) == "" {
		cfg.EABKID = old.EABKID
	}
	if !cfg.TermsAgreed {
		cfg.TermsAgreed = old.TermsAgreed
	}
	if !providerSet {
		cfg.DNSProvider = old.DNSProvider
		if len(cfg.DNS) == 0 {
			cfg.DNS = old.DNS
		}
	}
	if cfg.DNSProvider == old.DNSProvider && cfg.DNSProvider != "" {
		if cfg.DNS == nil {
			cfg.DNS = map[string]string{}
		}
		for _, name := range acme.SecretFields(cfg.DNSProvider) {
			if strings.TrimSpace(cfg.DNS[name]) == "" && old.DNS != nil {
				cfg.DNS[name] = old.DNS[name]
			}
		}
	}
	directoryChanged := strings.TrimSpace(cfg.DirectoryURL) != "" &&
		strings.TrimRight(strings.TrimSpace(cfg.DirectoryURL), "/") != strings.TrimRight(strings.TrimSpace(old.DirectoryURL), "/")
	if keyChanged || directoryChanged {
		cfg.RegistrationURI = ""
	} else if cfg.RegistrationURI == "" {
		cfg.RegistrationURI = old.RegistrationURI
	}
	return cfg
}

func trimDNS(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func boolValue(v any) bool {
	flag, ok := v.(bool)
	return ok && flag
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
	// 部署只覆盖证书目录中的域名文件，不写入 Nginx 配置。
	cfg.SnippetDir = ""
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
	cfg, err := publicConfig(item.ConfigJSON)
	if err != nil {
		return nil, err
	}
	return &View{
		ID: item.ID, Name: item.Name, Type: item.Type, Description: item.Description,
		Status: item.Status, Config: cfg, LastTestOK: item.LastTestOK, LastTestMsg: item.LastTestMsg,
		LastTestAt: item.LastTestAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

// SanitizeConfigJSON 给通用查询去掉账户私钥和云厂商密钥。
func SanitizeConfigJSON(raw string) string {
	cfg, err := publicConfig(raw)
	if err != nil {
		return raw
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return raw
	}
	return string(b)
}

func publicConfig(raw string) (map[string]any, error) {
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if pem, _ := cfg["accountKeyPEM"].(string); pem != "" {
		cfg["accountKeyStored"] = true
		delete(cfg, "accountKeyPEM")
	}
	if hmacKey, _ := cfg["eabHmacKey"].(string); hmacKey != "" {
		cfg["eabHmacStored"] = true
		delete(cfg, "eabHmacKey")
	}
	if uri, _ := cfg["registrationURI"].(string); uri != "" {
		cfg["accountRegistered"] = true
	}
	provider, _ := cfg["dnsProvider"].(string)
	dns, _ := cfg["dns"].(map[string]any)
	if dns != nil {
		stored := make([]string, 0)
		for _, name := range acme.SecretFields(provider) {
			if value, _ := dns[name].(string); value != "" {
				stored = append(stored, name)
				delete(dns, name)
			}
		}
		if len(stored) > 0 {
			cfg["dnsSecretStored"] = stored
		}
		if len(dns) == 0 {
			delete(cfg, "dns")
		}
	}
	return cfg, nil
}
