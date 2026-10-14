package etl

import (
	"context"
	"errors"
	"os"
	"time"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/datasource"
)

// ErrStop 表示流水线在当前阶段结束后正常停止，例如证书尚未进入续期窗口。
var ErrStop = errors.New("etl stop")

// TenantContext 来自登录态，不允许从请求体读取。
type TenantContext struct {
	TenantID uint
	UserID   uint
	Username string
}

// PipelineContext 描述这一次 y = f(x) 的运行参数。
type PipelineContext struct {
	Name    string
	Force   bool
	Trigger string
}

// TaskState 是流水线执行过程中的任务状态。
type TaskState struct {
	TaskID uint
	Status string
	Stage  string
	Error  string
}

// InputData 是 Extract 汇总的数据源、配置和证书。
type InputData struct {
	Certificate *Certificate
	ACME        *datasource.DataSource
	Nginx       *datasource.DataSource
	ACMEConfig  datasource.ACMEConfig
	NginxConfig datasource.NginxConfig
	Domains     []string
}

// TargetData 是 Transform 产出、Load 消费的证书材料。
type TargetData struct {
	CertPEM     string
	KeyPEM      string
	IssuerPEM   string
	NotBefore   time.Time
	NotAfter    time.Time
	Fingerprint string
	CertPath    string
	KeyPath     string
	ChainPath   string
	SnippetPath string
}

// TaskResult 是流水线输出。
type TaskResult struct {
	Skipped     bool
	Issued      bool
	Deployed    bool
	NginxAction string
	Message     string
	Stages      []string
}

// ETLContext 贯穿 Extract、Transform、Load。
type ETLContext struct {
	Pipeline       PipelineContext
	Task           TaskState
	Tenant         TenantContext
	CertificateID  uint
	Input          InputData
	Target         TargetData
	Result         TaskResult
	PreviousStatus string
	OnStage        func(stage string)
}

func (e *ETLContext) enter(stage string) {
	e.Task.Stage = stage
	e.Result.Stages = append(e.Result.Stages, stage)
	if e.OnStage != nil {
		e.OnStage(stage)
	}
}

// ETLStage 是流水线中的一个阶段。
type ETLStage interface {
	Execute(ctx context.Context, data *ETLContext) error
}

// Pipeline 按 Extract -> Transform -> Load 执行。
type Pipeline struct {
	stages []ETLStage
}

func (p *Pipeline) Run(ctx context.Context, data *ETLContext) error {
	for _, stage := range p.stages {
		if err := stage.Execute(ctx, data); err != nil {
			if errors.Is(err, ErrStop) {
				return nil
			}
			return err
		}
	}
	return nil
}

type extractStage struct{ svc *ETLService }
type transformStage struct{ svc *ETLService }
type loadStage struct{ svc *ETLService }

func (s extractStage) Execute(ctx context.Context, data *ETLContext) error {
	return s.svc.extract(ctx, data)
}
func (s transformStage) Execute(ctx context.Context, data *ETLContext) error {
	return s.svc.transform(ctx, data)
}
func (s loadStage) Execute(ctx context.Context, data *ETLContext) error {
	return s.svc.load(ctx, data)
}

func newPipeline(svc *ETLService) *Pipeline {
	return &Pipeline{stages: []ETLStage{extractStage{svc}, transformStage{svc}, loadStage{svc}}}
}

func (s *ETLService) extract(ctx context.Context, data *ETLContext) error {
	data.enter("extract")
	cert, err := s.repo.GetCertificate(ctx, data.Tenant.TenantID, data.CertificateID)
	if err != nil {
		return err
	}
	data.PreviousStatus = cert.Status
	if cert.Status == StatusIssuing {
		return apperror.New(409, "证书正在签发")
	}
	acme, err := s.sources.Get(ctx, data.Tenant.TenantID, cert.ACMEDataSourceID)
	if err != nil {
		return err
	}
	nginx, err := s.sources.Get(ctx, data.Tenant.TenantID, cert.NginxDataSourceID)
	if err != nil {
		return err
	}
	if acme.Type != datasource.TypeACME || nginx.Type != datasource.TypeNginx {
		return apperror.New(400, "证书关联的数据源类型不正确")
	}
	if acme.Status != datasource.StatusActive || nginx.Status != datasource.StatusActive {
		return apperror.New(400, "数据源已停用")
	}
	acmeCfg, err := datasource.ParseACME(acme.ConfigJSON)
	if err != nil {
		return err
	}
	nginxCfg, err := datasource.ParseNginx(nginx.ConfigJSON)
	if err != nil {
		return err
	}
	if cert.ChallengeType != ChallengeDNS01 {
		info, err := os.Stat(nginxCfg.Webroot)
		if err != nil || !info.IsDir() {
			return apperror.New(400, "ACME webroot 不存在，请先准备 Nginx 的 HTTP-01 目录")
		}
	}
	domains := decodeDomains(cert.DomainsJSON)
	if !data.Pipeline.Force && !cert.AutoRenew && data.Pipeline.Trigger == "schedule" {
		data.Result.Skipped = true
		data.Result.Message = "证书未开启自动续期"
		data.Input = InputData{Certificate: cert, ACME: acme, Nginx: nginx, ACMEConfig: acmeCfg, NginxConfig: nginxCfg, Domains: domains}
		return ErrStop
	}
	if !data.Pipeline.Force && cert.NotAfter != nil {
		window := time.Duration(cert.RenewBeforeDays) * 24 * time.Hour
		if window <= 0 {
			window = 30 * 24 * time.Hour
		}
		if time.Until(*cert.NotAfter) > window {
			data.Result.Skipped = true
			data.Result.Message = "证书尚未进入续期窗口"
			data.Input = InputData{Certificate: cert, ACME: acme, Nginx: nginx, ACMEConfig: acmeCfg, NginxConfig: nginxCfg, Domains: domains}
			return ErrStop
		}
	}
	cert.Status = StatusIssuing
	cert.LastError = ""
	if err := s.repo.SaveCertificate(ctx, cert); err != nil {
		return err
	}
	data.Input = InputData{Certificate: cert, ACME: acme, Nginx: nginx, ACMEConfig: acmeCfg, NginxConfig: nginxCfg, Domains: domains}
	return nil
}

func (s *ETLService) transform(ctx context.Context, data *ETLContext) error {
	data.enter("transform")
	if data.Input.Certificate == nil {
		return apperror.New(500, "流水线缺少证书输入")
	}
	email := data.Input.Certificate.Email
	if email == "" {
		email = data.Input.ACMEConfig.Email
	}
	issued, err := s.issuer.Obtain(ctx, IssueRequest{
		Email:           email,
		DirectoryURL:    data.Input.ACMEConfig.DirectoryURL,
		AccountKeyPEM:   data.Input.ACMEConfig.AccountKeyPEM,
		RegistrationURI: data.Input.ACMEConfig.RegistrationURI,
		EABKID:          data.Input.ACMEConfig.EABKID,
		EABHMACKey:      data.Input.ACMEConfig.EABHMACKey,
		Domains:         data.Input.Domains,
		Webroot:         data.Input.NginxConfig.Webroot,
		Challenge:       data.Input.Certificate.ChallengeType,
		DNSProvider:     data.Input.ACMEConfig.DNSProvider,
		DNS:             data.Input.ACMEConfig.DNS,
		KeyType:         data.Input.Certificate.KeyType,
		Timeout:         s.cfg.ACME.Timeout,
		AgreeTOS:        data.Input.Certificate.AgreeTOS,
	})
	if err != nil {
		return err
	}
	notBefore, notAfter, fingerprint, err := inspectCertificate(issued.CertPEM)
	if err != nil {
		return err
	}
	if issued.AccountKeyPEM != data.Input.ACMEConfig.AccountKeyPEM || issued.RegistrationURI != data.Input.ACMEConfig.RegistrationURI {
		data.Input.ACMEConfig.AccountKeyPEM = issued.AccountKeyPEM
		data.Input.ACMEConfig.RegistrationURI = issued.RegistrationURI
		if err := s.persistACME(ctx, data.Input.ACME, data.Input.ACMEConfig); err != nil {
			return err
		}
	}
	data.Target = TargetData{
		CertPEM: issued.CertPEM, KeyPEM: issued.KeyPEM, IssuerPEM: issued.IssuerPEM,
		NotBefore: notBefore, NotAfter: notAfter, Fingerprint: fingerprint,
	}
	data.Result.Issued = true
	return nil
}

func (s *ETLService) load(ctx context.Context, data *ETLContext) error {
	data.enter("load")
	deployed, err := s.deployer.Deploy(ctx, DeployRequest{
		Nginx:         data.Input.NginxConfig,
		CertificateID: data.Input.Certificate.ID,
		Domains:       data.Input.Domains,
		CertPEM:       data.Target.CertPEM,
		KeyPEM:        data.Target.KeyPEM,
		IssuerPEM:     data.Target.IssuerPEM,
	})
	if err != nil {
		return err
	}
	data.Target.CertPath = deployed.CertPath
	data.Target.KeyPath = deployed.KeyPath
	data.Target.ChainPath = deployed.ChainPath
	data.Target.SnippetPath = deployed.SnippetPath
	data.Result.Deployed = true
	data.Result.NginxAction = deployed.Action
	switch deployed.Action {
	case "dry-run":
		data.Result.Message = "证书已写入，已跳过 Nginx 重启"
	case datasource.ActionReload:
		data.Result.Message = "证书已替换，Nginx 已重载"
	default:
		data.Result.Message = "证书已替换，Nginx 服务已重启"
	}
	return nil
}
