package etl

import (
	"encoding/json"
	"strings"

	"sslplatform/internal/common/apperror"
)

func normalizeDomains(in []string) ([]string, error) {
	if len(in) > 20 {
		return nil, apperror.New(400, "单张证书最多 20 个域名")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
		}) {
			d := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(part), "."))
			if d == "" {
				continue
			}
			if err := validateDomain(d); err != nil {
				return nil, err
			}
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return nil, apperror.New(400, "至少填写一个域名")
	}
	return out, nil
}

func validateDomain(d string) error {
	if len(d) > 253 || strings.Contains(d, "*") || strings.Contains(d, "/") || !strings.Contains(d, ".") {
		return apperror.New(400, "域名无效，HTTP-01 不支持通配符、IP 或 localhost："+d)
	}
	labels := strings.Split(d, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return apperror.New(400, "域名无效："+d)
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return apperror.New(400, "域名无效："+d)
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return apperror.New(400, "域名无效："+d)
			}
		}
	}
	return nil
}

func encodeDomains(domains []string) (string, error) {
	b, err := json.Marshal(domains)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeDomains(raw string) []string {
	var domains []string
	if json.Unmarshal([]byte(raw), &domains) != nil || domains == nil {
		return []string{}
	}
	return domains
}

func normalizeKeyType(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", KeyEC256:
		return KeyEC256, nil
	case KeyRSA2048:
		return KeyRSA2048, nil
	default:
		return "", apperror.New(400, "keyType 只能是 ec256 或 rsa2048")
	}
}

func normalizeRenewDays(days int) (int, error) {
	if days == 0 {
		return 30, nil
	}
	if days < 1 || days > 60 {
		return 0, apperror.New(400, "续期窗口需在 1 到 60 天之间")
	}
	return days, nil
}

func validEmail(email string) bool {
	return strings.Contains(email, "@") && !strings.ContainsAny(email, " \t\r\n") && len(email) <= 254
}
