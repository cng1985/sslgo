package etl

import (
	"context"
	"crypto"
	"fmt"
	"net/http"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/http/webroot"
	"github.com/go-acme/lego/v4/registration"

	"sslplatform/internal/common/apperror"
)

// IssueRequest 是 ACME 签发阶段的输入。
type IssueRequest struct {
	Email           string
	DirectoryURL    string
	AccountKeyPEM   string
	RegistrationURI string
	Domains         []string
	Webroot         string
	KeyType         string
	Timeout         time.Duration
	AgreeTOS        bool
}

// IssuedCertificate 是 Transform 阶段交给 Load 的证书材料。
type IssuedCertificate struct {
	CertPEM         string
	KeyPEM          string
	IssuerPEM       string
	AccountKeyPEM   string
	RegistrationURI string
}

// CertificateIssuer 申请 Let's Encrypt 证书。
type CertificateIssuer interface {
	Obtain(ctx context.Context, req IssueRequest) (*IssuedCertificate, error)
}

type acmeUser struct {
	email string
	key   crypto.PrivateKey
	reg   *registration.Resource
}

func (u *acmeUser) GetEmail() string                        { return u.email }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.reg }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// LegoIssuer 使用 HTTP-01 webroot 向 ACME 目录申请证书。
type LegoIssuer struct{}

func NewIssuer() CertificateIssuer { return &LegoIssuer{} }

func (i *LegoIssuer) Obtain(ctx context.Context, req IssueRequest) (*IssuedCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !req.AgreeTOS {
		return nil, apperror.New(400, "必须同意 Let's Encrypt 服务条款")
	}
	if req.Webroot == "" {
		return nil, apperror.New(400, "Nginx 数据源未配置 webroot")
	}
	if len(req.Domains) == 0 {
		return nil, apperror.New(400, "至少填写一个域名")
	}
	keyPEM := req.AccountKeyPEM
	if keyPEM == "" {
		generated, err := newAccountKeyPEM()
		if err != nil {
			return nil, err
		}
		keyPEM = generated
	}
	priv, err := parsePrivateKey([]byte(keyPEM))
	if err != nil {
		return nil, apperror.Wrap(400, "ACME 账户密钥无效", err)
	}
	user := &acmeUser{email: req.Email, key: priv}
	if req.RegistrationURI != "" {
		user.reg = &registration.Resource{URI: req.RegistrationURI}
	}
	cfg := lego.NewConfig(user)
	cfg.CADirURL = req.DirectoryURL
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	cfg.HTTPClient = &http.Client{Timeout: timeout}
	cfg.Certificate.KeyType = certcrypto.EC256
	if req.KeyType == KeyRSA2048 {
		cfg.Certificate.KeyType = certcrypto.RSA2048
	}
	cfg.Certificate.Timeout = timeout
	cfg.UserAgent = "sslplatform/1.0.0"
	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 ACME 客户端失败: %w", err)
	}
	if user.reg == nil {
		resource, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("注册 ACME 账户失败: %w", err)
		}
		user.reg = resource
	}
	provider, err := webroot.NewHTTPProvider(req.Webroot)
	if err != nil {
		return nil, err
	}
	if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
		return nil, err
	}
	obtained, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: req.Domains,
		Bundle:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("申请证书失败: %w", err)
	}
	uri := ""
	if user.reg != nil {
		uri = user.reg.URI
	}
	return &IssuedCertificate{
		CertPEM:         string(obtained.Certificate),
		KeyPEM:          string(obtained.PrivateKey),
		IssuerPEM:       string(obtained.IssuerCertificate),
		AccountKeyPEM:   keyPEM,
		RegistrationURI: uri,
	}, nil
}

// MemoryIssuer 在测试中代替 ACME，生成自签证书。
type MemoryIssuer struct{}

func NewMemoryIssuer() CertificateIssuer { return &MemoryIssuer{} }

func (m *MemoryIssuer) Obtain(ctx context.Context, req IssueRequest) (*IssuedCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !req.AgreeTOS {
		return nil, apperror.New(400, "必须同意 Let's Encrypt 服务条款")
	}
	certPEM, keyPEM, err := MintSelfSigned(req.Domains)
	if err != nil {
		return nil, err
	}
	account := req.AccountKeyPEM
	if account == "" {
		account, err = newAccountKeyPEM()
		if err != nil {
			return nil, err
		}
	}
	return &IssuedCertificate{
		CertPEM:         certPEM,
		KeyPEM:          keyPEM,
		IssuerPEM:       certPEM,
		AccountKeyPEM:   account,
		RegistrationURI: "memory://acme-account",
	}, nil
}
