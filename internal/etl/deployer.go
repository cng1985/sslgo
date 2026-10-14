package etl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// NginxDeployer 把证书写入目标目录，检测配置后重启或重载 Nginx。
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
	crt, key, chain := artifactPaths(certDir, req.CertificateID)
	result := &DeployResult{CertPath: crt, KeyPath: key, ChainPath: chain, Action: req.Nginx.Action}
	var snippet string
	if req.Nginx.SnippetDir != "" {
		dir, err := proc.SafeDir(req.Nginx.SnippetDir)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		snippet = filepath.Join(dir, fmt.Sprintf("sslplatform-cert-%d.conf", req.CertificateID))
		result.SnippetPath = snippet
	}
	if req.RestartOnly {
		if err := filesExist(crt, key); err != nil {
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
	if req.CertPEM == "" || req.KeyPEM == "" {
		return nil, errors.New("证书或私钥为空")
	}
	snaps := make([]snapshot, 0, 4)
	for _, path := range []string{crt, key, chain, snippet} {
		if path == "" {
			continue
		}
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
	writes := []struct {
		path string
		body string
		mode os.FileMode
	}{
		{crt, req.CertPEM, 0o644},
		{key, req.KeyPEM, 0o600},
		{chain, req.IssuerPEM, 0o644},
	}
	for _, file := range writes {
		if file.body == "" && file.path == chain {
			continue
		}
		if err := writeAtomic(file.path, []byte(file.body), file.mode); err != nil {
			restore()
			return nil, err
		}
	}
	if snippet != "" {
		body := nginxSnippet(req.Domains, crt, key, req.Nginx.Webroot)
		if err := writeAtomic(snippet, []byte(body), 0o644); err != nil {
			restore()
			return nil, err
		}
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

func artifactPaths(certDir string, id uint) (string, string, string) {
	stem := fmt.Sprintf("cert-%d", id)
	return filepath.Join(certDir, stem+".crt"),
		filepath.Join(certDir, stem+".key"),
		filepath.Join(certDir, stem+".issuer.crt")
}

func nginxSnippet(domains []string, crt, key, webroot string) string {
	names := ""
	for i, d := range domains {
		if i > 0 {
			names += " "
		}
		names += d
	}
	return fmt.Sprintf(`server {
    listen 443 ssl;
    server_name %s;
    ssl_certificate %s;
    ssl_certificate_key %s;
    location ^~ /.well-known/acme-challenge/ {
        root %s;
    }
    location / {
        default_type text/plain;
        return 200 'sslplatform certificate is active\n';
    }
}
`, names, crt, key, webroot)
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
