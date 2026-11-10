// Package apperror 分类业务错误。目录是 platform/errors，包名避开标准库 errors。
package apperror

import stderrors "errors"

const (
	KindBadRequest   = "bad_request"
	KindUnauthorized = "unauthorized"
	KindForbidden    = "forbidden"
	KindNotFound     = "not_found"
	KindConflict     = "conflict"
	KindTooLarge     = "too_large"
	KindRateLimit    = "rate_limit"
	KindInternal     = "internal"
	KindUnavailable  = "unavailable"
)

// Error 是可公开的分类错误。Code 是应用码，Cause 只写服务端日志。
type Error struct {
	Kind  string
	Code  int
	Msg   string
	Cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Cause }

func New(code int, msg string) *Error {
	return &Error{Kind: kindOf(code), Code: code, Msg: msg}
}

func Wrap(code int, msg string, err error) *Error {
	return &Error{Kind: kindOf(code), Code: code, Msg: msg, Cause: err}
}

func IsCode(err error, code int) bool {
	var ae *Error
	if stderrors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}

func kindOf(code int) string {
	switch code {
	case 400:
		return KindBadRequest
	case 401:
		return KindUnauthorized
	case 403:
		return KindForbidden
	case 404:
		return KindNotFound
	case 409:
		return KindConflict
	case 413:
		return KindTooLarge
	case 429:
		return KindRateLimit
	case 503:
		return KindUnavailable
	default:
		if code >= 500 {
			return KindInternal
		}
		return KindBadRequest
	}
}
