package datasource

import (
	"time"

	"sslplatform/internal/entity"
)

const (
	TypeACME  = "acme"
	TypeNginx = "nginx"

	StatusActive   = "active"
	StatusDisabled = "disabled"

	ManagerSystemd = "systemd"
	ManagerSignal  = "signal"

	ActionRestart = "restart"
	ActionReload  = "reload"
)

type DataSource struct {
	entity.Base
	TenantID    uint       `gorm:"not null;index;uniqueIndex:idx_ds_name" json:"tenantId"`
	Name        string     `gorm:"size:128;not null;uniqueIndex:idx_ds_name" json:"name"`
	Type        string     `gorm:"size:32;not null" json:"type"`
	Description string     `gorm:"size:255" json:"description"`
	ConfigJSON  string     `gorm:"type:text;not null" json:"-"`
	Status      string     `gorm:"size:32;not null" json:"status"`
	LastTestAt  *time.Time `json:"lastTestAt,omitempty"`
	LastTestOK  bool       `json:"lastTestOk"`
	LastTestMsg string     `gorm:"size:1024" json:"lastTestMsg"`
}

func (DataSource) TableName() string { return "datasources" }

// ACMEConfig 是 ACME 账户配置。账户私钥和云厂商密钥只写入数据库，接口响应中会去掉。
type ACMEConfig struct {
	Email           string            `json:"email"`
	CA              string            `json:"ca,omitempty"`
	DirectoryURL    string            `json:"directoryURL"`
	AccountKeyPEM   string            `json:"accountKeyPEM,omitempty"`
	RegistrationURI string            `json:"registrationURI,omitempty"`
	EABKID          string            `json:"eabKid,omitempty"`
	EABHMACKey      string            `json:"eabHmacKey,omitempty"`
	TermsAgreed     bool              `json:"termsAgreed,omitempty"`
	DNSProvider     string            `json:"dnsProvider,omitempty"`
	DNS             map[string]string `json:"dns,omitempty"`
}

// NginxConfig 描述证书落地目录，以及签发后如何重启 Nginx。
// 部署只覆盖 CertDir 里已有的「域名.pem」和「域名.key」，例如 /usr/local/nginx/conf/certs/nbsaas.com.pem。
// SnippetDir 只为兼容旧配置保留，部署时不会读取，也不会修改 Nginx 配置文件。
type NginxConfig struct {
	NginxBin       string `json:"nginxBin"`
	ServiceManager string `json:"serviceManager"`
	ServiceName    string `json:"serviceName"`
	Action         string `json:"action"`
	Webroot        string `json:"webroot"`
	CertDir        string `json:"certDir"`
	SnippetDir     string `json:"snippetDir,omitempty"`
	TestBefore     bool   `json:"testBefore"`
	DryRun         bool   `json:"dryRun"`
}

type CreateRequest struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config"`
}

type UpdateRequest struct {
	ID          uint           `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	Config      map[string]any `json:"config"`
}

type IDRequest struct {
	ID uint `json:"id"`
}

// RegisterAccountRequest 把已保存的 ACME 配置注册到证书机构。
type RegisterAccountRequest struct {
	ID       uint `json:"id"`
	AgreeTOS bool `json:"agreeTos"`
}

type ListRequest struct {
	Page    int        `json:"page"`
	Size    int        `json:"size"`
	Filters ListFilter `json:"filters"`
}

type ListFilter struct {
	Type    string `json:"type"`
	Keyword string `json:"keyword"`
	Status  string `json:"status"`
}

type View struct {
	ID          uint           `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	Config      map[string]any `json:"config"`
	LastTestOK  bool           `json:"lastTestOk"`
	LastTestMsg string         `json:"lastTestMsg"`
	LastTestAt  *time.Time     `json:"lastTestAt,omitempty"`
	CreatedAt   any            `json:"createdAt"`
	UpdatedAt   any            `json:"updatedAt"`
}

type TestView struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}
