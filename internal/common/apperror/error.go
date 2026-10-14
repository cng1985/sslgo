package apperror

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Error 是返回给调用方的业务错误，Code 同时作为 HTTP 状态码。
type Error struct {
	Code int
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

func New(code int, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

func Wrap(code int, msg string, err error) *Error {
	return &Error{Code: code, Msg: msg, Err: err}
}

// FromDB 把记录不存在和唯一约束转换成明确的业务错误。
func FromDB(err error, notFound string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return New(404, notFound)
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "unique") || strings.Contains(lower, "constraint failed") {
		return New(409, "记录已存在")
	}
	return err
}
