package response

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/common/apperror"
)

type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type PageBody struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	Data         any    `json:"data"`
	Page         int    `json:"page"`
	Size         int    `json:"size"`
	Total        int64  `json:"total"`
	ResponseType string `json:"responseType"`
}

func OK(c *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	c.JSON(http.StatusOK, Body{Code: 200, Msg: "success", Data: data})
}

func Page(c *gin.Context, data any, page, size int, total int64) {
	if data == nil {
		data = []any{}
	} else {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			data = []any{}
		}
	}
	c.JSON(http.StatusOK, PageBody{
		Code:         200,
		Msg:          "success",
		Data:         data,
		Page:         page,
		Size:         size,
		Total:        total,
		ResponseType: "page",
	})
}

func Fail(c *gin.Context, code int, msg string) {
	if code < 400 {
		code = http.StatusBadRequest
	}
	c.JSON(code, Body{Code: code, Msg: msg, Data: map[string]any{}})
}

func FromError(c *gin.Context, err error) {
	if err == nil {
		OK(c, map[string]any{})
		return
	}
	var ae *apperror.Error
	if errors.As(err, &ae) {
		Fail(c, ae.Code, ae.Msg)
		return
	}
	slog.Error("request failed", "err", err, "path", c.FullPath())
	Fail(c, http.StatusInternalServerError, "内部错误")
}
