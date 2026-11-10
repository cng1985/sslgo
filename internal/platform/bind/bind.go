package bind

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/errors"
)

// JSON 解析请求体。空对象允许通过，字段校验由 Resource 完成。
func JSON(c *gin.Context, dst any) error {
	if c.Request.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return apperror.New(400, "读取请求失败")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return apperror.New(400, "请求 JSON 无效")
	}
	return nil
}
