package etl

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sslplatform/internal/common/proc"
	"sslplatform/internal/datasource"
)

// DeployRequest 是 Load 阶段替换证书并重启 Nginx 的输入。
type DeployRequest struct {
	Nginx         datasource.NginxConfig
	CertificateID uint
	Domains       []string
	CertPEM       string
	KeyPEM        string
	IssuerPEM     string
	RestartOnly   bool
}

// DeployResult 记录证书文件位置和实际执行的 Nginx 动作。
type DeployResult struct {
	CertPath    string
	KeyPath     string
	ChainPath   string
	SnippetPath string
	Action      string
}

// NginxDeployer 按域名覆盖证书目录里的 pem 和 key，然后重启或重载 Nginx。
// 不会写入或修改 Nginx 配置文件。
type NginxDeployer interface {
	Deploy(ctx context.Context, req DeployRequest) (*DeployResult, error)
}

type nginxDeployer struct{}

func NewDeployer() NginxDeployer { return nginxDeployer{} }

func (nginxDeployer) Deploy(ctx context.Context, req DeployRequest) (*DeployResult, error) {
	certDir, err := proc.SafeDir(req.Nginx.CertDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(certDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建证书目录失败: %w", err)
	}
	pemPath, keyPath := certificatePaths(certDir, req.Domains)
	result := &DeployResult{CertPath: pemPath, KeyPath: keyPath, Action: req.Nginx.Action}
	if req.RestartOnly {
		if err := filesExist(pemPath, keyPath); err != nil {
			return nil, err
		}
		if err := applyNginx(ctx, req.Nginx); err != nil {
			return nil, err
		}
		if req.Nginx.DryRun {
			result.Action = "dry-run"
		}
		return result, nil
	}
	certBody := nginxCertificateBody(req.CertPEM, req.IssuerPEM)
	keyBody := nginxPrivateKeyBody(req.KeyPEM)
	if certBody == "" || keyBody == "" {
		return nil, errors.New("证书或私钥为空")
	}
	snaps := make([]snapshot, 0, 2)
	for _, path := range []string{pemPath, keyPath} {
		shot, err := takeSnapshot(path)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, shot)
	}
	restore := func() {
		for i := len(snaps) - 1; i >= 0; i-- {
			_ = snaps[i].restore()
		}
	}
	if err := writeAtomic(pemPath, []byte(certBody), 0o644); err != nil {
		restore()
		return nil, err
	}
	if err := writeAtomic(keyPath, []byte(keyBody), 0o600); err != nil {
		restore()
		return nil, err
	}
	if err := applyNginx(ctx, req.Nginx); err != nil {
		restore()
		_ = applyNginx(ctx, req.Nginx)
		return nil, err
	}
	if req.Nginx.DryRun {
		result.Action = "dry-run"
	}
	return result, nil
}

// certificatePaths 用主域名命名 Nginx 已经引用的证书文件。
// *.nbsaas.com 写成 /usr/local/nginx/conf/certs/nbsaas.com.pem 和 nbsaas.com.key。
func certificatePaths(certDir string, domains []string) (string, string) {
	stem := nginxFileStem(domains)
	return filepath.Join(certDir, stem+".pem"), filepath.Join(certDir, stem+".key")
}

func nginxCertificateBody(certPEM, issuerPEM string) string {
	cert := strings.TrimSpace(certPEM)
	issuer := strings.TrimSpace(issuerPEM)
	if cert == "" {
		return withTrailingNewline(issuer)
	}
	if parsed, err := nginxFullChain(cert, issuer); err == nil {
		return parsed
	}
	if issuer == "" || issuer == cert || strings.Contains(cert, issuer) {
		return withTrailingNewline(cert)
	}
	return cert + "\n" + issuer + "\n"
}

func nginxPrivateKeyBody(keyPEM string) string {
	return withTrailingNewline(strings.TrimSpace(keyPEM))
}

func withTrailingNewline(text string) string {
	if text == "" || strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

func nginxFullChain(certPEM, issuerPEM string) (string, error) {
	certs, err := parseCertificates(certPEM, issuerPEM)
	if err != nil {
		return "", err
	}
	seen := map[string]struct{}{}
	ordered := make([]*x509.Certificate, 0, len(certs))
	for _, cert := range certs {
		raw := string(cert.Raw)
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		ordered = append(ordered, cert)
	}
	if len(ordered) == 0 {
		return "", errors.New("证书 PEM 无效")
	}
	return encodeCerts(ordered), nil
}

func applyNginx(ctx context.Context, cfg datasource.NginxConfig) error {
	if cfg.DryRun {
		return nil
	}
	if cfg.TestBefore {
		if _, err := proc.Run(ctx, cfg.NginxBin, "-t"); err != nil {
			return fmt.Errorf("nginx -t 未通过，已保留原证书: %w", err)
		}
	}
	switch cfg.ServiceManager {
	case datasource.ManagerSystemd:
		verb := "restart"
		if cfg.Action == datasource.ActionReload {
			verb = "reload"
		}
		if _, err := proc.Run(ctx, "systemctl", verb, cfg.ServiceName); err != nil {
			return fmt.Errorf("重启 Nginx 服务失败: %w", err)
		}
		return nil
	default:
		if cfg.Action == datasource.ActionReload {
			if _, err := proc.Run(ctx, cfg.NginxBin, "-s", "reload"); err != nil {
				return fmt.Errorf("重载 Nginx 失败: %w", err)
			}
			return nil
		}
		_, quitErr := proc.Run(ctx, cfg.NginxBin, "-s", "quit")
		if _, err := proc.Run(ctx, cfg.NginxBin); err != nil {
			if quitErr != nil {
				return fmt.Errorf("重启 Nginx 失败: %v; %w", quitErr, err)
			}
			return fmt.Errorf("启动 Nginx 失败: %w", err)
		}
		return nil
	}
}

type snapshot struct {
	path    string
	existed bool
	data    []byte
	mode    os.FileMode
}

func takeSnapshot(path string) (snapshot, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{path: path, mode: 0o644}, nil
	}
	if err != nil {
		return snapshot{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot{}, err
	}
	bak := path + ".bak." + time.Now().Format("20060102150405")
	if err := writeAtomic(bak, data, info.Mode().Perm()); err != nil {
		return snapshot{}, err
	}
	return snapshot{path: path, existed: true, data: data, mode: info.Mode().Perm()}, nil
}

func (s snapshot) restore() error {
	if !s.existed {
		err := os.Remove(s.path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return writeAtomic(s.path, s.data, s.mode)
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cert-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func filesExist(paths ...string) error {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return fmt.Errorf("证书文件不存在: %s", path)
		}
	}
	return nil
}
