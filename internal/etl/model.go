package etl

import (
	"time"

	"sslplatform/internal/entity"
)

const (
	StatusDraft    = "draft"
	StatusIssuing  = "issuing"
	StatusIssued   = "issued"
	StatusDeployed = "deployed"
	StatusFailed   = "failed"

	ChallengeHTTP01 = "http-01"
	ChallengeDNS01  = "dns-01"
	KeyEC256        = "ec256"
	KeyRSA2048      = "rsa2048"
	PipelineIssue   = "ssl_issue"
	JobIdle         = "idle"
	JobRunning      = "running"
)

// Certificate 记录一张 Let's Encrypt 证书的申请和部署状态。
// 私钥不出现在列表和详情里。单独申请时保存在 KeyPEM，只能通过下载接口取出；部署后按域名写入 Nginx 的 certDir。
type Certificate struct {
	entity.Base
	TenantID          uint       `gorm:"not null;uniqueIndex:idx_cert_name" json:"tenantId"`
	Name              string     `gorm:"size:128;not null;uniqueIndex:idx_cert_name" json:"name"`
	DomainsJSON       string     `gorm:"type:text;not null" json:"-"`
	Email             string     `gorm:"size:255" json:"email"`
	ChallengeType     string     `gorm:"size:32;not null" json:"challengeType"`
	KeyType           string     `gorm:"size:32;not null" json:"keyType"`
	ACMEDataSourceID  uint       `gorm:"not null;index" json:"acmeDataSourceId"`
	NginxDataSourceID uint       `gorm:"index" json:"nginxDataSourceId"`
	RenewBeforeDays   int        `json:"renewBeforeDays"`
	AutoRenew         bool       `json:"autoRenew"`
	AgreeTOS          bool       `json:"agreeTos"`
	Status            string     `gorm:"size:32;not null;index" json:"status"`
	NotBefore         *time.Time `json:"notBefore,omitempty"`
	NotAfter          *time.Time `json:"notAfter,omitempty"`
	Fingerprint       string     `gorm:"size:255" json:"fingerprint"`
	CertPEM           string     `gorm:"type:text" json:"-"`
	KeyPEM            string     `gorm:"type:text" json:"-"`
	IssuerPEM         string     `gorm:"type:text" json:"-"`
	LastError         string     `gorm:"type:text" json:"lastError"`
	LastIssuedAt      *time.Time `json:"lastIssuedAt,omitempty"`
	LastDeployedAt    *time.Time `json:"lastDeployedAt,omitempty"`
}

func (Certificate) TableName() string { return "certificates" }

// ETLJob 是绑定到证书的签发流水线。
type ETLJob struct {
	entity.Base
	TenantID      uint   `gorm:"not null;index;uniqueIndex:idx_job_name" json:"tenantId"`
	Name          string `gorm:"size:128;not null;uniqueIndex:idx_job_name" json:"name"`
	CertificateID uint   `gorm:"not null;index" json:"certificateId"`
	Pipeline      string `gorm:"size:64;not null" json:"pipeline"`
	Status        string `gorm:"size:32;not null" json:"status"`
	ConfigJSON    string `gorm:"type:text" json:"-"`
}

func (ETLJob) TableName() string { return "etl_jobs" }

type CertificateCreateRequest struct {
	Name              string   `json:"name"`
	Domains           []string `json:"domains"`
	Email             string   `json:"email"`
	ChallengeType     string   `json:"challengeType"`
	KeyType           string   `json:"keyType"`
	ACMEDataSourceID  uint     `json:"acmeDataSourceId"`
	NginxDataSourceID uint     `json:"nginxDataSourceId"`
	RenewBeforeDays   int      `json:"renewBeforeDays"`
	AutoRenew         bool     `json:"autoRenew"`
	AgreeTOS          bool     `json:"agreeTos"`
}

type CertificateUpdateRequest struct {
	ID                uint     `json:"id"`
	Name              string   `json:"name"`
	Domains           []string `json:"domains"`
	Email             string   `json:"email"`
	ChallengeType     string   `json:"challengeType"`
	KeyType           string   `json:"keyType"`
	ACMEDataSourceID  uint     `json:"acmeDataSourceId"`
	NginxDataSourceID uint     `json:"nginxDataSourceId"`
	RenewBeforeDays   int      `json:"renewBeforeDays"`
	AutoRenew         bool     `json:"autoRenew"`
	AgreeTOS          bool     `json:"agreeTos"`
}

type CertificateListRequest struct {
	Page    int                   `json:"page"`
	Size    int                   `json:"size"`
	Filters CertificateListFilter `json:"filters"`
}

type CertificateListFilter struct {
	Status  string `json:"status"`
	Keyword string `json:"keyword"`
}

type CertificateIDRequest struct {
	ID uint `json:"id"`
}

type CertificateIssueRequest struct {
	ID    uint `json:"id"`
	Force bool `json:"force"`
}

type CertificateDeployRequest struct {
	ID uint `json:"id"`
}

type CertificateDownloadRequest struct {
	ID       uint   `json:"id"`
	Format   string `json:"format"`
	Password string `json:"password"`
}

type CertificateDownload struct {
	Filename      string     `json:"filename"`
	ContentType   string     `json:"contentType"`
	ContentBase64 string     `json:"contentBase64"`
	Format        string     `json:"format"`
	NotAfter      *time.Time `json:"notAfter,omitempty"`
}

type JobCreateRequest struct {
	Name          string `json:"name"`
	CertificateID uint   `json:"certificateId"`
}

type JobUpdateRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type JobListRequest struct {
	Page    int           `json:"page"`
	Size    int           `json:"size"`
	Filters JobListFilter `json:"filters"`
}

type JobListFilter struct {
	CertificateID uint   `json:"certificateId"`
	Status        string `json:"status"`
}

type JobIDRequest struct {
	ID uint `json:"id"`
}

type JobExecuteRequest struct {
	ID     uint  `json:"id"`
	Force  bool  `json:"force"`
	Deploy *bool `json:"deploy"`
}

type CertificateView struct {
	ID                uint       `json:"id"`
	Name              string     `json:"name"`
	Domains           []string   `json:"domains"`
	Email             string     `json:"email"`
	ChallengeType     string     `json:"challengeType"`
	KeyType           string     `json:"keyType"`
	ACMEDataSourceID  uint       `json:"acmeDataSourceId"`
	NginxDataSourceID uint       `json:"nginxDataSourceId"`
	RenewBeforeDays   int        `json:"renewBeforeDays"`
	AutoRenew         bool       `json:"autoRenew"`
	AgreeTOS          bool       `json:"agreeTos"`
	Status            string     `json:"status"`
	NotBefore         *time.Time `json:"notBefore,omitempty"`
	NotAfter          *time.Time `json:"notAfter,omitempty"`
	Fingerprint       string     `json:"fingerprint"`
	DaysRemaining     *int       `json:"daysRemaining,omitempty"`
	LastError         string     `json:"lastError"`
	LastIssuedAt      *time.Time `json:"lastIssuedAt,omitempty"`
	LastDeployedAt    *time.Time `json:"lastDeployedAt,omitempty"`
	CertPath          string     `json:"certPath"`
	KeyPath           string     `json:"keyPath"`
	ChainPath         string     `json:"chainPath"`
	JobID             uint       `json:"jobId"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type CertificateDetail struct {
	CertificateView
	CertPEM string `json:"certPem"`
}

type JobView struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	CertificateID uint      `json:"certificateId"`
	Pipeline      string    `json:"pipeline"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Summary struct {
	CertificateTotal int64             `json:"certificateTotal"`
	Deployed         int64             `json:"deployed"`
	Failed           int64             `json:"failed"`
	ExpiringSoon     int64             `json:"expiringSoon"`
	RunningTasks     int64             `json:"runningTasks"`
	DatasourceTotal  int64             `json:"datasourceTotal"`
	Expiring         []CertificateView `json:"expiring"`
}
