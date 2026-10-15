package etl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/config"
	"sslplatform/internal/datasource"
	"sslplatform/internal/paging"
	"sslplatform/internal/task"
)

// ETLService 管理证书、签发流水线和部署。
type ETLService struct {
	repo     *Repository
	tasks    *task.Repository
	sources  *datasource.Repository
	issuer   CertificateIssuer
	deployer NginxDeployer
	cfg      *config.Config
	pipeline *Pipeline
	locks    *keyLock
}

func NewETLService(repo *Repository, tasks *task.Repository, sources *datasource.Repository, issuer CertificateIssuer, deployer NginxDeployer, cfg *config.Config) *ETLService {
	svc := &ETLService{
		repo: repo, tasks: tasks, sources: sources, issuer: issuer, deployer: deployer, cfg: cfg,
		locks: &keyLock{locks: map[uint]*sync.Mutex{}},
	}
	svc.pipeline = newPipeline(svc)
	return svc
}

func (s *ETLService) CreateCertificate(ctx context.Context, tenantID uint, req CertificateCreateRequest) (*CertificateView, error) {
	item, err := s.prepareCertificate(ctx, tenantID, req.Name, req.Domains, req.Email, req.ChallengeType, req.KeyType, req.ACMEDataSourceID, req.NginxDataSourceID, req.RenewBeforeDays, req.AutoRenew, req.AgreeTOS)
	if err != nil {
		return nil, err
	}
	item.Status = StatusDraft
	if err := s.repo.CreateCertificate(ctx, item); err != nil {
		return nil, err
	}
	job := &ETLJob{
		TenantID: tenantID, Name: "签发-" + item.Name, CertificateID: item.ID,
		Pipeline: PipelineIssue, Status: JobIdle, ConfigJSON: "{}",
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	return s.view(ctx, item, job.ID)
}

func (s *ETLService) UpdateCertificate(ctx context.Context, tenantID uint, req CertificateUpdateRequest) (*CertificateView, error) {
	current, err := s.repo.GetCertificate(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusIssuing {
		return nil, apperror.New(409, "证书正在签发，不能修改")
	}
	item, err := s.prepareCertificate(ctx, tenantID, req.Name, req.Domains, req.Email, req.ChallengeType, req.KeyType, req.ACMEDataSourceID, req.NginxDataSourceID, req.RenewBeforeDays, req.AutoRenew, req.AgreeTOS)
	if err != nil {
		return nil, err
	}
	current.Name = item.Name
	current.DomainsJSON = item.DomainsJSON
	current.Email = item.Email
	current.ChallengeType = item.ChallengeType
	current.KeyType = item.KeyType
	current.ACMEDataSourceID = item.ACMEDataSourceID
	current.NginxDataSourceID = item.NginxDataSourceID
	current.RenewBeforeDays = item.RenewBeforeDays
	current.AutoRenew = item.AutoRenew
	current.AgreeTOS = item.AgreeTOS
	if err := s.repo.SaveCertificate(ctx, current); err != nil {
		return nil, err
	}
	job, _ := s.repo.JobByCertificate(ctx, tenantID, current.ID)
	var jobID uint
	if job != nil {
		jobID = job.ID
	}
	return s.view(ctx, current, jobID)
}

func (s *ETLService) DeleteCertificate(ctx context.Context, tenantID, id uint) error {
	running, err := s.tasks.HasRunning(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if running {
		return apperror.New(409, "证书有正在执行的任务")
	}
	return s.repo.DeleteCertificate(ctx, tenantID, id)
}

func (s *ETLService) DetailCertificate(ctx context.Context, tenantID, id uint) (*CertificateDetail, error) {
	item, err := s.repo.GetCertificate(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	job, _ := s.repo.JobByCertificate(ctx, tenantID, id)
	var jobID uint
	if job != nil {
		jobID = job.ID
	}
	view, err := s.view(ctx, item, jobID)
	if err != nil {
		return nil, err
	}
	return &CertificateDetail{CertificateView: *view, CertPEM: item.CertPEM}, nil
}

func (s *ETLService) Download(ctx context.Context, tenantID uint, req CertificateDownloadRequest) (*CertificateDownload, error) {
	format, err := normalizeBundleFormat(req.Format)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.GetCertificate(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	certPEM, keyPEM, issuerPEM, err := s.certificateMaterial(ctx, item)
	if err != nil {
		return nil, err
	}
	file, err := buildBundle(bundleInput{
		Format: format, Name: item.Name, Domains: decodeDomains(item.DomainsJSON),
		NotAfter: derefTime(item.NotAfter), CertPEM: certPEM, KeyPEM: keyPEM, IssuerPEM: issuerPEM,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return &CertificateDownload{
		Filename: file.Name, ContentType: "application/zip",
		ContentBase64: base64.StdEncoding.EncodeToString(file.Body),
		Format:        format,
		NotAfter:      item.NotAfter,
	}, nil
}

func (s *ETLService) certificateMaterial(ctx context.Context, item *Certificate) (string, string, string, error) {
	certPEM, keyPEM, issuerPEM := item.CertPEM, item.KeyPEM, item.IssuerPEM
	if strings.TrimSpace(certPEM) == "" || strings.TrimSpace(keyPEM) == "" {
		nginx, err := s.sources.Get(ctx, item.TenantID, item.NginxDataSourceID)
		if err == nil {
			if cfg, err := datasource.ParseNginx(nginx.ConfigJSON); err == nil {
				diskCert, diskKey, diskIssuer, err := readMaterialFiles(cfg.CertDir, decodeDomains(item.DomainsJSON))
				if err == nil {
					if strings.TrimSpace(certPEM) == "" {
						certPEM = diskCert
					}
					if strings.TrimSpace(keyPEM) == "" {
						keyPEM = diskKey
					}
					if strings.TrimSpace(issuerPEM) == "" {
						issuerPEM = diskIssuer
					}
				}
			}
		}
	}
	if strings.TrimSpace(certPEM) == "" || strings.TrimSpace(keyPEM) == "" {
		return "", "", "", apperror.New(400, "证书尚未签发，无法下载")
	}
	return certPEM, keyPEM, issuerPEM, nil
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func (s *ETLService) ListCertificates(ctx context.Context, tenantID uint, req CertificateListRequest) ([]CertificateView, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.ListCertificates(ctx, tenantID, page, size, req.Filters)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]CertificateView, 0, len(rows))
	for i := range rows {
		job, _ := s.repo.JobByCertificate(ctx, tenantID, rows[i].ID)
		var jobID uint
		if job != nil {
			jobID = job.ID
		}
		view, err := s.view(ctx, &rows[i], jobID)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		views = append(views, *view)
	}
	return views, page, size, total, nil
}

func (s *ETLService) Issue(ctx context.Context, tenantID, userID uint, req CertificateIssueRequest) (*task.View, error) {
	job, err := s.repo.JobByCertificate(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	deploy := true
	return s.Execute(ctx, tenantID, userID, JobExecuteRequest{ID: job.ID, Force: req.Force, Deploy: &deploy}, task.TriggerManual, false)
}

func (s *ETLService) Apply(ctx context.Context, tenantID, userID uint, req CertificateIssueRequest) (*task.View, error) {
	job, err := s.repo.JobByCertificate(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	deploy := false
	return s.Execute(ctx, tenantID, userID, JobExecuteRequest{ID: job.ID, Force: req.Force, Deploy: &deploy}, task.TriggerManual, false)
}

func (s *ETLService) DeployExisting(ctx context.Context, tenantID, userID, id uint) (*task.View, error) {
	cert, err := s.repo.GetCertificate(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if cert.NginxDataSourceID == 0 {
		return nil, apperror.New(400, "这张证书没有绑定 Nginx，无法部署")
	}
	nginx, err := s.sources.Get(ctx, tenantID, cert.NginxDataSourceID)
	if err != nil {
		return nil, err
	}
	cfg, err := datasource.ParseNginx(nginx.ConfigJSON)
	if err != nil {
		return nil, err
	}
	job, err := s.repo.JobByCertificate(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	running, err := s.tasks.HasRunning(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if running {
		return nil, apperror.New(409, "证书有正在执行的任务")
	}
	hasMaterial := strings.TrimSpace(cert.CertPEM) != "" && strings.TrimSpace(cert.KeyPEM) != ""
	if !hasMaterial {
		pemPath, keyPath := certificatePaths(cfg.CertDir, decodeDomains(cert.DomainsJSON))
		if err := filesExist(pemPath, keyPath); err != nil {
			return nil, apperror.New(400, "还没有可部署的证书，请先申请")
		}
	}
	now := time.Now()
	item := &task.Task{
		TenantID: tenantID, UserID: userID, ETLJobID: job.ID, CertificateID: id,
		Trigger: task.TriggerManual, Status: task.StatusRunning, Stage: "load", Force: true, Deploy: true, StartedAt: &now,
	}
	if err := s.tasks.Create(ctx, item); err != nil {
		return nil, err
	}
	deployReq := DeployRequest{Nginx: cfg, CertificateID: id, Domains: decodeDomains(cert.DomainsJSON), RestartOnly: true}
	if hasMaterial {
		deployReq.RestartOnly = false
		deployReq.CertPEM = cert.CertPEM
		deployReq.KeyPEM = cert.KeyPEM
		deployReq.IssuerPEM = cert.IssuerPEM
	}
	result, err := s.deployer.Deploy(ctx, deployReq)
	finished := time.Now()
	item.FinishedAt = &finished
	if err != nil {
		item.Status = task.StatusFailed
		item.Error = err.Error()
		cert.LastError = err.Error()
		_ = s.repo.SaveCertificate(ctx, cert)
		_ = s.tasks.Save(ctx, item)
		return nil, err
	}
	cert.Status = StatusDeployed
	cert.LastError = ""
	cert.LastDeployedAt = &finished
	if err := s.repo.SaveCertificate(ctx, cert); err != nil {
		return nil, err
	}
	item.Status = task.StatusSuccess
	body := resultBody{Deployed: true, NginxAction: result.Action, Message: deployMessage(result.Action), Stages: []string{"load"}, CertPath: result.CertPath, KeyPath: result.KeyPath, ChainPath: result.ChainPath, SnippetPath: result.SnippetPath}
	raw, _ := json.Marshal(body)
	item.ResultJSON = string(raw)
	if err := s.tasks.Save(ctx, item); err != nil {
		return nil, err
	}
	view := task.ViewFrom(item)
	return &view, nil
}

func (s *ETLService) CreateJob(ctx context.Context, tenantID uint, req JobCreateRequest) (*JobView, error) {
	cert, err := s.repo.GetCertificate(ctx, tenantID, req.CertificateID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.JobByCertificate(ctx, tenantID, cert.ID); err == nil {
		return nil, apperror.New(409, "该证书已有签发流水线")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "签发-" + cert.Name
	}
	job := &ETLJob{TenantID: tenantID, Name: name, CertificateID: cert.ID, Pipeline: PipelineIssue, Status: JobIdle, ConfigJSON: "{}"}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	return toJobView(job), nil
}

func (s *ETLService) UpdateJob(ctx context.Context, tenantID uint, req JobUpdateRequest) (*JobView, error) {
	job, err := s.repo.GetJob(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, apperror.New(400, "请填写流水线名称")
	}
	job.Name = strings.TrimSpace(req.Name)
	if err := s.repo.SaveJob(ctx, job); err != nil {
		return nil, err
	}
	return toJobView(job), nil
}

func (s *ETLService) DeleteJob(ctx context.Context, tenantID, id uint) error {
	return s.repo.DeleteJob(ctx, tenantID, id)
}

func (s *ETLService) ListJobs(ctx context.Context, tenantID uint, req JobListRequest) ([]JobView, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.ListJobs(ctx, tenantID, page, size, req.Filters)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]JobView, 0, len(rows))
	for i := range rows {
		views = append(views, *toJobView(&rows[i]))
	}
	return views, page, size, total, nil
}

func (s *ETLService) DetailJob(ctx context.Context, tenantID, id uint) (*JobView, error) {
	job, err := s.repo.GetJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return toJobView(job), nil
}

func (s *ETLService) Execute(ctx context.Context, tenantID, userID uint, req JobExecuteRequest, trigger string, sync bool) (*task.View, error) {
	job, err := s.repo.GetJob(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	cert, err := s.repo.GetCertificate(ctx, tenantID, job.CertificateID)
	if err != nil {
		return nil, err
	}
	deploy, err := resolveDeploy(cert, req.Deploy)
	if err != nil {
		return nil, err
	}
	running, err := s.tasks.HasRunning(ctx, tenantID, job.CertificateID)
	if err != nil {
		return nil, err
	}
	if running {
		return nil, apperror.New(409, "证书有正在执行的任务")
	}
	if trigger == "" {
		trigger = task.TriggerManual
	}
	item := &task.Task{
		TenantID: tenantID, UserID: userID, ETLJobID: job.ID, CertificateID: job.CertificateID,
		Trigger: trigger, Status: task.StatusPending, Stage: "extract", Force: req.Force, Deploy: deploy,
	}
	if err := s.tasks.Create(ctx, item); err != nil {
		return nil, err
	}
	if sync {
		s.finish(item.ID)
		stored, err := s.tasks.Get(ctx, tenantID, item.ID)
		if err != nil {
			return nil, err
		}
		view := task.ViewFrom(stored)
		if stored.Status == task.StatusFailed {
			return &view, apperror.New(500, stored.Error)
		}
		return &view, nil
	}
	go s.finish(item.ID)
	view := task.ViewFrom(item)
	return &view, nil
}

func (s *ETLService) ExecuteSync(ctx context.Context, tenantID, userID uint, req JobExecuteRequest) (*task.View, error) {
	return s.Execute(ctx, tenantID, userID, req, task.TriggerManual, true)
}

func (s *ETLService) Summary(ctx context.Context, tenantID uint) (*Summary, error) {
	total, err := s.repo.CountCertificates(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}
	deployed, err := s.repo.CountCertificates(ctx, tenantID, StatusDeployed)
	if err != nil {
		return nil, err
	}
	failed, err := s.repo.CountCertificates(ctx, tenantID, StatusFailed)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(30 * 24 * time.Hour)
	expiringCount, err := s.repo.CountExpiring(ctx, tenantID, deadline)
	if err != nil {
		return nil, err
	}
	expiring, err := s.repo.Expiring(ctx, tenantID, deadline)
	if err != nil {
		return nil, err
	}
	running, err := s.tasks.CountSince(ctx, tenantID, task.StatusRunning, time.Time{})
	if err != nil {
		return nil, err
	}
	var sources int64
	if err := s.sourcesCount(ctx, tenantID, &sources); err != nil {
		return nil, err
	}
	views := make([]CertificateView, 0, len(expiring))
	for i := range expiring {
		view, err := s.view(ctx, &expiring[i], 0)
		if err != nil {
			return nil, err
		}
		views = append(views, *view)
	}
	return &Summary{
		CertificateTotal: total, Deployed: deployed, Failed: failed, ExpiringSoon: expiringCount,
		RunningTasks: running, DatasourceTotal: sources, Expiring: views,
	}, nil
}

func (s *ETLService) finish(taskID uint) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("证书流水线异常", "task", taskID, "panic", rec)
			s.markFailed(taskID, fmt.Sprintf("内部错误: %v", rec))
		}
	}()
	item, err := s.tasks.GetByID(context.Background(), taskID)
	if err != nil {
		slog.Error("读取任务失败", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.issueTimeout(item))
	defer cancel()
	unlock := s.locks.lock(item.CertificateID)
	defer unlock()
	if err := s.tasks.MarkRunning(ctx, item.ID, "extract"); err != nil {
		slog.Error("标记任务运行失败", "err", err)
		return
	}
	data := &ETLContext{
		Pipeline:      PipelineContext{Name: PipelineIssue, Force: item.Force, Trigger: item.Trigger, Deploy: item.Deploy},
		Task:          TaskState{TaskID: item.ID, Status: task.StatusRunning},
		Tenant:        TenantContext{TenantID: item.TenantID, UserID: item.UserID},
		CertificateID: item.CertificateID,
		OnStage: func(stage string) {
			_ = s.tasks.MarkRunning(context.Background(), item.ID, stage)
		},
	}
	if err := s.pipeline.Run(ctx, data); err != nil {
		s.markFailed(item.ID, err.Error())
		return
	}
	if err := s.markSuccess(ctx, item.ID, data); err != nil {
		slog.Error("保存签发结果失败", "err", err)
		s.markFailed(item.ID, err.Error())
	}
}

func (s *ETLService) markSuccess(ctx context.Context, taskID uint, data *ETLContext) error {
	item, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	cert := data.Input.Certificate
	if cert == nil {
		cert, err = s.repo.GetCertificate(ctx, item.TenantID, item.CertificateID)
		if err != nil {
			return err
		}
	}
	now := time.Now()
	body := resultBody{
		Skipped: data.Result.Skipped, Issued: data.Result.Issued, Deployed: data.Result.Deployed,
		NginxAction: data.Result.NginxAction, Message: data.Result.Message, Stages: data.Result.Stages,
		CertPath: data.Target.CertPath, KeyPath: data.Target.KeyPath, ChainPath: data.Target.ChainPath,
		SnippetPath: data.Target.SnippetPath, Fingerprint: data.Target.Fingerprint,
	}
	if data.Result.Skipped {
		cert.Status = data.PreviousStatus
		if cert.Status == "" || cert.Status == StatusIssuing {
			cert.Status = StatusDraft
		}
		item.Status = task.StatusSkipped
	} else {
		cert.CertPEM = data.Target.CertPEM
		cert.KeyPEM = data.Target.KeyPEM
		cert.IssuerPEM = data.Target.IssuerPEM
		cert.NotBefore = &data.Target.NotBefore
		cert.NotAfter = &data.Target.NotAfter
		cert.Fingerprint = data.Target.Fingerprint
		cert.LastError = ""
		cert.LastIssuedAt = &now
		if data.Result.Deployed {
			cert.Status = StatusDeployed
			cert.LastDeployedAt = &now
		} else {
			cert.Status = StatusIssued
		}
		item.Status = task.StatusSuccess
		body.NotAfter = cert.NotAfter
	}
	if err := s.repo.SaveCertificate(ctx, cert); err != nil {
		return err
	}
	if job, err := s.repo.GetJob(ctx, item.TenantID, item.ETLJobID); err == nil {
		job.Status = JobIdle
		_ = s.repo.SaveJob(ctx, job)
	}
	raw, _ := json.Marshal(body)
	item.ResultJSON = string(raw)
	item.Stage = data.Task.Stage
	item.Error = ""
	item.FinishedAt = &now
	return s.tasks.Save(ctx, item)
}

func (s *ETLService) markFailed(taskID uint, message string) {
	ctx := context.Background()
	item, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return
	}
	now := time.Now()
	item.Status = task.StatusFailed
	item.Error = message
	item.FinishedAt = &now
	_ = s.tasks.Save(ctx, item)
	if cert, err := s.repo.GetCertificate(ctx, item.TenantID, item.CertificateID); err == nil {
		cert.Status = StatusFailed
		cert.LastError = message
		_ = s.repo.SaveCertificate(ctx, cert)
	}
	if job, err := s.repo.GetJob(ctx, item.TenantID, item.ETLJobID); err == nil {
		job.Status = JobIdle
		_ = s.repo.SaveJob(ctx, job)
	}
}

func (s *ETLService) issueTimeout(item *task.Task) time.Duration {
	timeout := s.cfg.ACME.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	if cert, err := s.repo.GetCertificate(context.Background(), item.TenantID, item.CertificateID); err == nil && cert.ChallengeType == ChallengeDNS01 {
		if timeout < 4*time.Minute {
			timeout = 4 * time.Minute
		}
	}
	return timeout + 30*time.Second
}

func (s *ETLService) prepareCertificate(ctx context.Context, tenantID uint, name string, domains []string, email, challenge, keyType string, acmeID, nginxID uint, renewDays int, autoRenew, agree bool) (*Certificate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperror.New(400, "请填写证书名称")
	}
	if !agree {
		return nil, apperror.New(400, "必须同意 ACME 服务条款")
	}
	normalized, err := normalizeDomains(domains)
	if err != nil {
		return nil, err
	}
	challenge, err = normalizeChallenge(challenge, normalized)
	if err != nil {
		return nil, err
	}
	if email != "" && !validEmail(email) {
		return nil, apperror.New(400, "邮箱格式无效")
	}
	keyType, err = normalizeKeyType(keyType)
	if err != nil {
		return nil, err
	}
	renewDays, err = normalizeRenewDays(renewDays)
	if err != nil {
		return nil, err
	}
	acme, err := s.sources.Get(ctx, tenantID, acmeID)
	if err != nil {
		return nil, err
	}
	if acme.Type != datasource.TypeACME {
		return nil, apperror.New(400, "请选择 ACME 数据源")
	}
	var nginx *datasource.DataSource
	if nginxID != 0 {
		nginx, err = s.sources.Get(ctx, tenantID, nginxID)
		if err != nil {
			return nil, err
		}
		if nginx.Type != datasource.TypeNginx {
			return nil, apperror.New(400, "请选择 Nginx 数据源")
		}
	}
	if challenge == ChallengeHTTP01 && nginx == nil {
		return nil, apperror.New(400, "HTTP-01 需要 Nginx 数据源提供验证目录。只申请证书时请使用 DNS-01，并可以不选 Nginx")
	}
	acmeCfg, err := datasource.ParseACME(acme.ConfigJSON)
	if err != nil {
		return nil, err
	}
	if challenge == ChallengeDNS01 && strings.TrimSpace(acmeCfg.DNSProvider) == "" {
		return nil, apperror.New(400, "DNS-01 需要先在 ACME 账户绑定云厂商 DNS")
	}
	rawDomains, err := encodeDomains(normalized)
	if err != nil {
		return nil, err
	}
	var boundNginx uint
	if nginx != nil {
		boundNginx = nginx.ID
	}
	return &Certificate{
		TenantID: tenantID, Name: name, DomainsJSON: rawDomains, Email: email,
		ChallengeType: challenge, KeyType: keyType, ACMEDataSourceID: acme.ID,
		NginxDataSourceID: boundNginx, RenewBeforeDays: renewDays, AutoRenew: autoRenew, AgreeTOS: true,
	}, nil
}

func resolveDeploy(cert *Certificate, requested *bool) (bool, error) {
	if requested == nil {
		return cert.NginxDataSourceID != 0, nil
	}
	if *requested && cert.NginxDataSourceID == 0 {
		return false, apperror.New(400, "部署需要 Nginx 数据源。可以不部署，单独申请证书")
	}
	return *requested, nil
}

func (s *ETLService) view(ctx context.Context, item *Certificate, jobID uint) (*CertificateView, error) {
	view := &CertificateView{
		ID: item.ID, Name: item.Name, Domains: decodeDomains(item.DomainsJSON), Email: item.Email,
		ChallengeType: item.ChallengeType, KeyType: item.KeyType, ACMEDataSourceID: item.ACMEDataSourceID,
		NginxDataSourceID: item.NginxDataSourceID, RenewBeforeDays: item.RenewBeforeDays, AutoRenew: item.AutoRenew,
		AgreeTOS: item.AgreeTOS, Status: item.Status, NotBefore: item.NotBefore, NotAfter: item.NotAfter,
		Fingerprint: item.Fingerprint, LastError: item.LastError, LastIssuedAt: item.LastIssuedAt,
		LastDeployedAt: item.LastDeployedAt, JobID: jobID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.NotAfter != nil {
		days := remainingDays(*item.NotAfter, time.Now())
		view.DaysRemaining = &days
	}
	if nginx, err := s.sources.Get(ctx, item.TenantID, item.NginxDataSourceID); err == nil {
		if cfg, err := datasource.ParseNginx(nginx.ConfigJSON); err == nil && cfg.CertDir != "" {
			view.CertPath, view.KeyPath = certificatePaths(cfg.CertDir, decodeDomains(item.DomainsJSON))
		}
	}
	return view, nil
}

func (s *ETLService) persistACME(ctx context.Context, item *datasource.DataSource, cfg datasource.ACMEConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	item.ConfigJSON = string(raw)
	return s.sources.Save(ctx, item)
}

func (s *ETLService) sourcesCount(ctx context.Context, tenantID uint, dst *int64) error {
	rows, total, err := s.sources.List(ctx, tenantID, 1, 1, datasource.ListFilter{})
	if err != nil {
		return err
	}
	_ = rows
	*dst = total
	return nil
}

func toJobView(job *ETLJob) *JobView {
	return &JobView{
		ID: job.ID, Name: job.Name, CertificateID: job.CertificateID, Pipeline: job.Pipeline,
		Status: job.Status, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
	}
}

type resultBody struct {
	Skipped     bool       `json:"skipped"`
	Issued      bool       `json:"issued"`
	Deployed    bool       `json:"deployed"`
	NginxAction string     `json:"nginxAction"`
	Message     string     `json:"message"`
	Stages      []string   `json:"stages"`
	CertPath    string     `json:"certPath,omitempty"`
	KeyPath     string     `json:"keyPath,omitempty"`
	ChainPath   string     `json:"chainPath,omitempty"`
	SnippetPath string     `json:"snippetPath,omitempty"`
	Fingerprint string     `json:"fingerprint,omitempty"`
	NotAfter    *time.Time `json:"notAfter,omitempty"`
}

func deployMessage(action string) string {
	switch action {
	case "dry-run":
		return "已跳过 Nginx 重启"
	case datasource.ActionReload:
		return "Nginx 已重载"
	default:
		return "Nginx 服务已重启"
	}
}

type keyLock struct {
	mu    sync.Mutex
	locks map[uint]*sync.Mutex
}

func (l *keyLock) lock(id uint) func() {
	l.mu.Lock()
	m, ok := l.locks[id]
	if !ok {
		m = &sync.Mutex{}
		l.locks[id] = m
	}
	l.mu.Unlock()
	m.Lock()
	return m.Unlock
}
