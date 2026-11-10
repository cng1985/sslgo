package certificate

import (
	"context"

	"sslplatform/internal/modules/task"
	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

type Deleted struct {
	ID uint `json:"id"`
}

type SummaryRequest struct{}

func (s *Resource) CreateCertificate(ctx context.Context, req *CertificateCreateRequest) (response.Response[CertificateView], error) {
	if req == nil {
		return response.Response[CertificateView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[CertificateView]{}, err
	}
	view, err := s.createCertificate(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[CertificateView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) UpdateCertificate(ctx context.Context, req *CertificateUpdateRequest) (response.Response[CertificateView], error) {
	if req == nil {
		return response.Response[CertificateView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[CertificateView]{}, err
	}
	view, err := s.updateCertificate(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[CertificateView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) DeleteCertificate(ctx context.Context, req *CertificateIDRequest) (response.Response[Deleted], error) {
	if req == nil {
		return response.Response[Deleted]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[Deleted]{}, err
	}
	if err := s.deleteCertificate(ctx, scope.TenantID, req.ID); err != nil {
		return response.Response[Deleted]{}, err
	}
	return response.Success(Deleted{ID: req.ID}), nil
}

func (s *Resource) ListCertificates(ctx context.Context, req *CertificateListRequest) (response.PageResponse[CertificateView], error) {
	if req == nil {
		req = &CertificateListRequest{}
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[CertificateView]{}, err
	}
	rows, page, size, total, err := s.listCertificates(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[CertificateView]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) DetailCertificate(ctx context.Context, req *CertificateIDRequest) (response.Response[CertificateDetail], error) {
	if req == nil {
		return response.Response[CertificateDetail]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[CertificateDetail]{}, err
	}
	view, err := s.detailCertificate(ctx, scope.TenantID, req.ID)
	if err != nil {
		return response.Response[CertificateDetail]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Download(ctx context.Context, req *CertificateDownloadRequest) (response.Response[CertificateDownload], error) {
	if req == nil {
		return response.Response[CertificateDownload]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[CertificateDownload]{}, err
	}
	view, err := s.download(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[CertificateDownload]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Issue(ctx context.Context, req *CertificateIssueRequest) (response.Response[task.View], error) {
	if req == nil {
		return response.Response[task.View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	view, err := s.issue(ctx, scope.TenantID, scope.UserID, *req)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Apply(ctx context.Context, req *CertificateIssueRequest) (response.Response[task.View], error) {
	if req == nil {
		return response.Response[task.View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	view, err := s.apply(ctx, scope.TenantID, scope.UserID, *req)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) DeployExisting(ctx context.Context, req *CertificateDeployRequest) (response.Response[task.View], error) {
	if req == nil {
		return response.Response[task.View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	view, err := s.deployExisting(ctx, scope.TenantID, scope.UserID, req.ID)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) CreateJob(ctx context.Context, req *JobCreateRequest) (response.Response[JobView], error) {
	if req == nil {
		return response.Response[JobView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	view, err := s.createJob(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) UpdateJob(ctx context.Context, req *JobUpdateRequest) (response.Response[JobView], error) {
	if req == nil {
		return response.Response[JobView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	view, err := s.updateJob(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) DeleteJob(ctx context.Context, req *JobIDRequest) (response.Response[Deleted], error) {
	if req == nil {
		return response.Response[Deleted]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[Deleted]{}, err
	}
	if err := s.deleteJob(ctx, scope.TenantID, req.ID); err != nil {
		return response.Response[Deleted]{}, err
	}
	return response.Success(Deleted{ID: req.ID}), nil
}

func (s *Resource) ListJobs(ctx context.Context, req *JobListRequest) (response.PageResponse[JobView], error) {
	if req == nil {
		req = &JobListRequest{}
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[JobView]{}, err
	}
	rows, page, size, total, err := s.listJobs(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[JobView]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) DetailJob(ctx context.Context, req *JobIDRequest) (response.Response[JobView], error) {
	if req == nil {
		return response.Response[JobView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	view, err := s.detailJob(ctx, scope.TenantID, req.ID)
	if err != nil {
		return response.Response[JobView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Execute(ctx context.Context, req *JobExecuteRequest) (response.Response[task.View], error) {
	if req == nil {
		return response.Response[task.View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	trigger := task.TriggerManual
	if scope.System {
		trigger = task.TriggerSchedule
	}
	view, err := s.execute(ctx, scope.TenantID, scope.UserID, *req, trigger, false)
	if err != nil {
		return response.Response[task.View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Summary(ctx context.Context, _ *SummaryRequest) (response.Response[Summary], error) {
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[Summary]{}, err
	}
	view, err := s.summary(ctx, scope.TenantID)
	if err != nil {
		return response.Response[Summary]{}, err
	}
	return response.Success(*view), nil
}
