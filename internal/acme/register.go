package acme

import (
	"context"
	"crypto"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// RegisterRequest 是向 ACME 目录注册账户的输入。
type RegisterRequest struct {
	Email         string
	DirectoryURL  string
	AccountKeyPEM string
	EABKID        string
	EABHMACKey    string
	Timeout       time.Duration
}

// RegisterResult 保存注册成功后的账户地址和私钥。
type RegisterResult struct {
	AccountKeyPEM   string
	RegistrationURI string
}

type accountUser struct {
	email string
	key   crypto.PrivateKey
	reg   *registration.Resource
}

func (u *accountUser) GetEmail() string                        { return u.email }
func (u *accountUser) GetRegistration() *registration.Resource { return u.reg }
func (u *accountUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// Register 使用账户私钥向 ACME 目录注册。同一把私钥重复注册会返回已有账户。
func Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !strings.Contains(req.Email, "@") {
		return nil, fmt.Errorf("请填写有效的 ACME 邮箱")
	}
	if err := validDirectory(req.DirectoryURL); err != nil {
		return nil, err
	}
	keyPEM := strings.TrimSpace(req.AccountKeyPEM)
	if keyPEM == "" {
		generated, err := GenerateAccountKeyPEM()
		if err != nil {
			return nil, err
		}
		keyPEM = generated
	}
	priv, err := ParsePrivateKey([]byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("ACME 账户密钥无效")
	}
	user := &accountUser{email: strings.TrimSpace(req.Email), key: priv}
	cfg := lego.NewConfig(user)
	cfg.CADirURL = req.DirectoryURL
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cfg.HTTPClient = &http.Client{Timeout: timeout}
	cfg.UserAgent = "sslplatform/1.0.0"
	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 ACME 客户端失败")
	}
	kid := strings.TrimSpace(req.EABKID)
	hmacKey := strings.TrimSpace(req.EABHMACKey)
	var resource *registration.Resource
	if kid != "" || hmacKey != "" {
		if kid == "" || hmacKey == "" {
			return nil, fmt.Errorf("外部账号绑定需要同时填写 EAB Kid 和 HMAC Key")
		}
		resource, err = client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
			TermsOfServiceAgreed: true,
			Kid:                  kid,
			HmacEncoded:          hmacKey,
		})
	} else {
		resource, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	}
	if err != nil {
		return nil, fmt.Errorf("%s", clipErr(err))
	}
	if resource == nil || resource.URI == "" {
		return nil, fmt.Errorf("ACME 目录没有返回账户地址")
	}
	return &RegisterResult{AccountKeyPEM: keyPEM, RegistrationURI: resource.URI}, nil
}

func clipErr(err error) string {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > 280 {
		msg = msg[:280]
	}
	return msg
}
