package certificate

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/bind"
	"sslplatform/internal/platform/response"
)

func done[T any](ctx *gin.Context, result response.Response[T], err error) {
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessResponse(ctx, result)
}

func donePage[T any](ctx *gin.Context, result response.PageResponse[T], err error) {
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessPage(ctx, result)
}

func (c *Controller) CreateCertificate(ctx *gin.Context) {
	var req CertificateCreateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.CreateCertificate(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) UpdateCertificate(ctx *gin.Context) {
	var req CertificateUpdateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.UpdateCertificate(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) DeleteCertificate(ctx *gin.Context) {
	var req CertificateIDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.DeleteCertificate(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) ListCertificates(ctx *gin.Context) {
	var req CertificateListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.ListCertificates(ctx.Request.Context(), &req)
	donePage(ctx, result, err)
}

func (c *Controller) DetailCertificate(ctx *gin.Context) {
	var req CertificateIDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.DetailCertificate(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Download(ctx *gin.Context) {
	var req CertificateDownloadRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Download(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Issue(ctx *gin.Context) {
	var req CertificateIssueRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Issue(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Apply(ctx *gin.Context) {
	var req CertificateIssueRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Apply(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Deploy(ctx *gin.Context) {
	var req CertificateDeployRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.DeployExisting(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) CreateJob(ctx *gin.Context) {
	var req JobCreateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.CreateJob(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) UpdateJob(ctx *gin.Context) {
	var req JobUpdateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.UpdateJob(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) DeleteJob(ctx *gin.Context) {
	var req JobIDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.DeleteJob(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) ListJobs(ctx *gin.Context) {
	var req JobListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.ListJobs(ctx.Request.Context(), &req)
	donePage(ctx, result, err)
}

func (c *Controller) DetailJob(ctx *gin.Context) {
	var req JobIDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.DetailJob(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Execute(ctx *gin.Context) {
	var req JobExecuteRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Execute(ctx.Request.Context(), &req)
	done(ctx, result, err)
}

func (c *Controller) Summary(ctx *gin.Context) {
	var req SummaryRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Summary(ctx.Request.Context(), &req)
	done(ctx, result, err)
}
