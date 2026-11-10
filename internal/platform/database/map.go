package database

import (
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	"sslplatform/internal/platform/errors"
)

// MapError 把记录不存在和唯一约束转换成分类错误。errors 包本身不依赖 GORM。
func MapError(err error, notFound string) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.New(404, notFound)
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "unique") || strings.Contains(lower, "constraint failed") {
		return apperror.New(409, "记录已存在")
	}
	return err
}
