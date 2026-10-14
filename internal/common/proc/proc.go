package proc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Allowed 是证书部署可以调用的程序名。参数以数组传递，不经过 shell。
var Allowed = map[string]struct{}{
	"nginx":     {},
	"systemctl": {},
	"service":   {},
}

var serviceNameRe = regexp.MustCompile(`^[A-Za-z0-9_.@:-]+$`)

// Resolve 校验可执行文件，只接受白名单中的程序。
func Resolve(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" || strings.ContainsAny(bin, "\x00|&;$`<>\n\r") || strings.Contains(bin, "..") {
		return "", errors.New("可执行文件路径无效")
	}
	base := filepath.Base(bin)
	if _, ok := Allowed[base]; !ok {
		return "", fmt.Errorf("不允许执行 %s", base)
	}
	if filepath.IsAbs(bin) {
		return filepath.Clean(bin), nil
	}
	resolved, err := exec.LookPath(base)
	if err != nil {
		return "", fmt.Errorf("找不到可执行文件 %s", base)
	}
	return resolved, nil
}

// ValidService 判断 systemd 服务名是否可安全作为参数。
func ValidService(name string) bool {
	return serviceNameRe.MatchString(name)
}

// SafeDir 要求目录是绝对路径，且原始字符串里不包含路径回退。
func SafeDir(p string) (string, error) {
	if strings.TrimSpace(p) == "" || strings.ContainsAny(p, "\x00") || strings.Contains(p, "..") {
		return "", errors.New("目录路径无效")
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("目录必须是绝对路径")
	}
	return filepath.Clean(p), nil
}

// Run 执行白名单程序并返回截断后的输出。
func Run(ctx context.Context, bin string, args ...string) (string, error) {
	resolved, err := Resolve(bin)
	if err != nil {
		return "", err
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, "\x00\n\r") {
			return "", errors.New("命令参数无效")
		}
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	out := buf.String()
	if len(out) > 4096 {
		out = out[:4096]
	}
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", filepath.Base(resolved), strings.Join(args, " "), err, strings.TrimSpace(out))
	}
	return out, nil
}
