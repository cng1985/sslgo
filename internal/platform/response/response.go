package response

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/errors"
)

// Response 是单条业务结果。成功时 Code 为 200。
type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}

// PageResponse 是分页结果。
// 已发布客户端依赖 code、msg、size 和 responseType，因此保留这些字段。
// 详见 docs/decisions.md。
type PageResponse[T any] struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	Data         []T    `json:"data"`
	Page         int    `json:"page"`
	Size         int    `json:"size"`
	Total        int64  `json:"total"`
	ResponseType string `json:"responseType"`
}

func Success[T any](data T) Response[T] {
	return Response[T]{Code: 200, Msg: "success", Data: data}
}

func PageOf[T any](data []T, page, size int, total int64) PageResponse[T] {
	if data == nil {
		data = []T{}
	}
	return PageResponse[T]{
		Code: 200, Msg: "success", Data: data,
		Page: page, Size: size, Total: total, ResponseType: "page",
	}
}

func SuccessResponse[T any](c *gin.Context, data Response[T]) {
	c.JSON(http.StatusOK, data)
}

func SuccessPage[T any](c *gin.Context, data PageResponse[T]) {
	if data.Data == nil {
		data.Data = []T{}
	}
	c.JSON(http.StatusOK, data)
}

// OK 输出单条成功结果。data 为 nil 时使用空对象，避免前端读到 null。
func OK(c *gin.Context, data any) {
	if data == nil {
		data = map[string]any{}
	}
	c.JSON(http.StatusOK, Response[any]{Code: 200, Msg: "success", Data: data})
}

// Page 输出已发布的分页外壳。
func Page(c *gin.Context, data any, page, size int, total int64) {
	if data == nil {
		data = []any{}
	} else {
		v := reflect.ValueOf(data)
		if v.Kind() == reflect.Slice && v.IsNil() {
			data = []any{}
		}
	}
	c.JSON(http.StatusOK, map[string]any{
		"code": 200, "msg": "success", "data": data,
		"page": page, "size": size, "total": total, "responseType": "page",
	})
}

func Fail(c *gin.Context, code int, msg string) {
	if code < 400 {
		code = http.StatusBadRequest
	}
	c.JSON(code, Response[any]{Code: code, Msg: msg, Data: map[string]any{}})
}

// HandleError 把分类错误写成对外响应。未知错误不回传内部原因。
func HandleError(c *gin.Context, err error) {
	if err == nil {
		OK(c, map[string]any{})
		return
	}
	var ae *apperror.Error
	if stderrors.As(err, &ae) {
		Fail(c, ae.Code, ae.Msg)
		return
	}
	slog.Error("请求失败", "err", err, "path", c.FullPath(), "kind", apperror.KindInternal)
	Fail(c, http.StatusInternalServerError, "内部错误")
}

// FromError 与 HandleError 相同，保留原调用名。
func FromError(c *gin.Context, err error) { HandleError(c, err) }
