package app_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"software.sslmate.com/src/go-pkcs12"

	"sslplatform/internal/app"
	"sslplatform/internal/auth"
	"sslplatform/internal/etl"
	"sslplatform/internal/server"
)

func TestPlatformFlow(t *testing.T) {
	dir := t.TempDir()
	webroot := filepath.Join(dir, "www")
	certDir := filepath.Join(dir, "certs")
	snippetDir := filepath.Join(dir, "conf.d")
	for _, path := range []string{webroot, certDir, snippetDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	nginxBin := filepath.Join(dir, "nginx")
	if err := os.WriteFile(nginxBin, []byte("#!/bin/sh\necho \"$*\" >> \""+filepath.Join(dir, "nginx.log")+"\"\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSL_DATABASE_PATH", filepath.Join(dir, "app.db"))
	t.Setenv("SSL_JWT_SECRET", "test-secret-please-change")
	t.Setenv("SSL_SERVER_ADDR", "127.0.0.1:0")
	t.Setenv("SSL_ADMIN_PASSWORD", "Admin@123456")
	t.Setenv("SSL_SERVER_MODE", "test")

	var endpoint *server.Endpoint
	var db *gorm.DB
	application := fx.New(
		app.Module,
		fx.Populate(&endpoint, &db),
		fx.Decorate(func(etl.CertificateIssuer) etl.CertificateIssuer { return etl.NewMemoryIssuer() }),
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
	)
	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := application.Start(startCtx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = application.Stop(stopCtx)
	})
	base := "http://" + endpoint.Get()

	login := decodeData(t, mustPost(t, base, "/api/v1/auth/login", "", map[string]any{
		"username": "admin", "password": "Admin@123456",
	}, 200))
	token, _ := login["token"].(string)
	if token == "" {
		t.Fatal("登录未返回 token")
	}
	profile := login["profile"].(map[string]any)
	if profile["tenant"].(map[string]any)["code"] != "default" {
		t.Fatalf("tenant=%v", profile["tenant"])
	}

	status, _ := raw(t, http.MethodGet, base, "/api/v1/system/info", token, map[string]any{})
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应被拒绝，status=%d", status)
	}
	mustPost(t, base, "/api/v1/auth/profile", token, map[string]any{"tenantId": 1}, 400)

	acme := decodeData(t, mustPost(t, base, "/api/v1/datasource/create", token, map[string]any{
		"name": "acme", "type": "acme",
		"config": map[string]any{"email": "admin@example.com", "directoryURL": "https://acme-staging-v02.api.letsencrypt.org/directory"},
	}, 200))
	nginx := decodeData(t, mustPost(t, base, "/api/v1/datasource/create", token, map[string]any{
		"name": "nginx", "type": "nginx",
		"config": map[string]any{
			"nginxBin": nginxBin, "serviceManager": "signal", "serviceName": "nginx", "action": "restart",
			"webroot": webroot, "certDir": certDir, "snippetDir": snippetDir, "testBefore": true, "dryRun": false,
		},
	}, 200))
	if strings.Contains(mustJSON(acme["config"]), "accountKeyPEM") {
		t.Fatal("响应不应包含账户私钥")
	}

	mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "bad", "domains": []string{"*.example.com"}, "agreeTos": true,
		"acmeDataSourceId": acme["id"], "nginxDataSourceId": nginx["id"],
	}, 400)

	cert := decodeData(t, mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "示例站点", "domains": []string{"example.com", "www.example.com"},
		"agreeTos": true, "autoRenew": true, "renewBeforeDays": 30, "keyType": "ec256",
		"acmeDataSourceId": acme["id"], "nginxDataSourceId": nginx["id"],
	}, 200))
	issued := decodeData(t, mustPost(t, base, "/api/v1/certificate/issue", token, map[string]any{
		"id": cert["id"], "force": true,
	}, 200))
	taskBody := waitTask(t, base, token, issued["id"])
	if taskBody["status"] != "success" {
		t.Fatalf("签发失败: %+v", taskBody)
	}
	result := taskBody["result"].(map[string]any)
	if result["deployed"] != true || result["nginxAction"] != "restart" {
		t.Fatalf("result=%v", result)
	}
	if !strings.Contains(result["message"].(string), "重启") {
		t.Fatalf("message=%v", result["message"])
	}
	leaf := readCert(t, result["certPath"].(string))
	if len(leaf.DNSNames) != 2 {
		t.Fatalf("dns=%v", leaf.DNSNames)
	}
	listed := mustPost(t, base, "/api/v1/certificate/list", token, map[string]any{"page": 1, "size": 20}, 200)
	if strings.Contains(mustJSON(listed), "PRIVATE KEY") || strings.Contains(mustJSON(listed), "keyPem") || strings.Contains(mustJSON(listed), "issuerPem") {
		t.Fatal("证书列表不应返回私钥")
	}
	foundExpiry := false
	for _, item := range listed["data"].([]any) {
		row := item.(map[string]any)
		if row["id"] == cert["id"] {
			foundExpiry = row["notAfter"] != nil && row["daysRemaining"] != nil
		}
	}
	if !foundExpiry {
		t.Fatalf("列表缺少到期时间: %s", mustJSON(listed))
	}
	detail := decodeData(t, mustPost(t, base, "/api/v1/certificate/detail", token, map[string]any{"id": cert["id"]}, 200))
	if strings.Contains(mustJSON(detail), "PRIVATE KEY") || strings.Contains(mustJSON(detail), "keyPem") {
		t.Fatal("证书详情不应返回私钥")
	}
	if !strings.Contains(detail["certPem"].(string), "BEGIN CERTIFICATE") {
		t.Fatal("详情应返回公钥 PEM")
	}
	nginxFiles := unzipDownload(t, decodeData(t, mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{
		"id": cert["id"], "format": "nginx",
	}, 200)))
	if !bytes.Contains(nginxFiles["fullchain.pem"], []byte("BEGIN CERTIFICATE")) || !bytes.Contains(nginxFiles["privkey.pem"], []byte("PRIVATE KEY")) {
		t.Fatal("Nginx 下载包不完整")
	}
	httpdFiles := unzipDownload(t, decodeData(t, mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{
		"id": cert["id"], "format": "httpd",
	}, 200)))
	if !bytes.Contains(httpdFiles["server.crt"], []byte("BEGIN CERTIFICATE")) || !bytes.Contains(httpdFiles["server.key"], []byte("PRIVATE KEY")) {
		t.Fatal("httpd 下载包不完整")
	}
	if _, ok := httpdFiles["chain.crt"]; ok {
		t.Fatal("自签测试证书不应包含重复证书链")
	}
	mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{
		"id": cert["id"], "format": "iis", "password": "short",
	}, 400)
	iisFiles := unzipDownload(t, decodeData(t, mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{
		"id": cert["id"], "format": "iis", "password": "Iis@123456",
	}, 200)))
	if _, _, err := pkcs12.Decode(iisFiles["certificate.pfx"], "Iis@123456"); err != nil {
		t.Fatal(err)
	}
	pending := decodeData(t, mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "未签发", "domains": []string{"pending.example.com"}, "agreeTos": true,
		"acmeDataSourceId": acme["id"], "nginxDataSourceId": nginx["id"],
	}, 200))
	mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{
		"id": pending["id"], "format": "nginx",
	}, 400)
	logText := readFile(t, filepath.Join(dir, "nginx.log"))
	if !strings.Contains(logText, "-t") || !strings.Contains(logText, "-s quit") {
		t.Fatalf("nginx 日志=%s", logText)
	}

	rerun := decodeData(t, mustPost(t, base, "/api/v1/etlJob/execute", token, map[string]any{
		"id": cert["jobId"], "force": false,
	}, 200))
	skipped := waitTask(t, base, token, rerun["id"])
	if skipped["status"] != "skipped" {
		t.Fatalf("未到期证书应跳过: %+v", skipped)
	}
	if readFile(t, filepath.Join(dir, "nginx.log")) != logText {
		t.Fatal("跳过续期时不应重启 Nginx")
	}

	providerCatalog := decodeData(t, mustPost(t, base, "/api/v1/datasource/catalog", token, map[string]any{}, 200))
	foundAliyun := false
	for _, item := range providerCatalog["dnsProviders"].([]any) {
		if item.(map[string]any)["id"] == "aliyun" {
			foundAliyun = true
		}
	}
	if !foundAliyun || len(providerCatalog["cas"].([]any)) < 2 {
		t.Fatalf("catalog=%v", providerCatalog)
	}
	mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "缺少 DNS", "domains": []string{"*.example.com"}, "challengeType": "dns-01", "agreeTos": true,
		"acmeDataSourceId": acme["id"], "nginxDataSourceId": nginx["id"],
	}, 400)
	cloud := decodeData(t, mustPost(t, base, "/api/v1/datasource/create", token, map[string]any{
		"name": "aliyun", "type": "acme",
		"config": map[string]any{
			"email": "admin@example.com", "ca": "letsencrypt-staging", "dnsProvider": "aliyun",
			"dns": map[string]any{"accessKeyId": "LTAItest", "accessKeySecret": "secret-value"},
		},
	}, 200))
	if strings.Contains(mustJSON(cloud["config"]), "secret-value") || strings.Contains(mustJSON(cloud["config"]), "accountKeyPEM") {
		t.Fatal("云厂商密钥或账户私钥不应出现在响应中")
	}
	wild := decodeData(t, mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "通配符", "domains": []string{"*.example.com", "example.com"}, "challengeType": "dns-01",
		"agreeTos": true, "acmeDataSourceId": cloud["id"], "nginxDataSourceId": nginx["id"],
	}, 200))
	if wild["challengeType"] != "dns-01" {
		t.Fatalf("challenge=%v", wild["challengeType"])
	}
	wildTask := decodeData(t, mustPost(t, base, "/api/v1/certificate/issue", token, map[string]any{
		"id": wild["id"], "force": true,
	}, 200))
	wildResult := waitTask(t, base, token, wildTask["id"])
	if wildResult["status"] != "success" {
		t.Fatalf("通配符签发失败: %+v", wildResult)
	}

	mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "缺少验证目录", "domains": []string{"http.example.com"}, "challengeType": "http-01",
		"agreeTos": true, "acmeDataSourceId": acme["id"],
	}, 400)
	only := decodeData(t, mustPost(t, base, "/api/v1/certificate/create", token, map[string]any{
		"name": "仅申请", "domains": []string{"only.example.com"}, "challengeType": "dns-01",
		"agreeTos": true, "acmeDataSourceId": cloud["id"],
	}, 200))
	if only["nginxDataSourceId"].(float64) != 0 || only["status"] != "draft" {
		t.Fatalf("单独申请证书=%v", only)
	}
	mustPost(t, base, "/api/v1/certificate/issue", token, map[string]any{"id": only["id"], "force": true}, 400)
	mustPost(t, base, "/api/v1/etlJob/execute", token, map[string]any{"id": only["jobId"], "force": true, "deploy": true}, 400)
	applied := decodeData(t, mustPost(t, base, "/api/v1/certificate/apply", token, map[string]any{
		"id": only["id"], "force": true,
	}, 200))
	if applied["deploy"] != false {
		t.Fatalf("申请任务应标记为不部署: %+v", applied)
	}
	appliedTask := waitTask(t, base, token, applied["id"])
	if appliedTask["status"] != "success" {
		t.Fatalf("单独申请失败: %+v", appliedTask)
	}
	appliedResult := appliedTask["result"].(map[string]any)
	if appliedResult["issued"] != true || appliedResult["deployed"] != false || !strings.Contains(appliedResult["message"].(string), "尚未部署") {
		t.Fatalf("单独申请结果=%v", appliedResult)
	}
	onlyFile := filepath.Join(certDir, fmt.Sprintf("cert-%.0f.crt", only["id"].(float64)))
	if _, err := os.Stat(onlyFile); !os.IsNotExist(err) {
		t.Fatalf("单独申请不应写入证书文件: %v", err)
	}
	onlyDetail := decodeData(t, mustPost(t, base, "/api/v1/certificate/detail", token, map[string]any{"id": only["id"]}, 200))
	if onlyDetail["status"] != "issued" {
		t.Fatalf("状态=%v", onlyDetail["status"])
	}
	if _, ok := onlyDetail["keyPem"]; ok || strings.Contains(mustJSON(onlyDetail), "PRIVATE KEY") {
		t.Fatal("详情不应返回私钥")
	}
	onlyList := mustPost(t, base, "/api/v1/certificate/list", token, map[string]any{
		"page": 1, "size": 20, "filters": map[string]any{"keyword": "仅申请"},
	}, 200)
	if strings.Contains(mustJSON(onlyList), "PRIVATE KEY") {
		t.Fatal("列表不应返回私钥")
	}
	material := decodeData(t, mustPost(t, base, "/api/v1/certificate/download", token, map[string]any{"id": only["id"]}, 200))
	if !strings.Contains(material["keyPem"].(string), "PRIVATE KEY") || !strings.Contains(material["certPem"].(string), "CERTIFICATE") {
		t.Fatal("下载结果缺少证书或私钥")
	}
	if leaf := readCertPEM(t, material["certPem"].(string)); len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "only.example.com" {
		t.Fatalf("下载的证书域名=%v", leaf.DNSNames)
	}
	bound := decodeData(t, mustPost(t, base, "/api/v1/certificate/update", token, map[string]any{
		"id": only["id"], "name": "仅申请", "domains": []string{"only.example.com"}, "challengeType": "dns-01",
		"agreeTos": true, "acmeDataSourceId": cloud["id"], "nginxDataSourceId": nginx["id"], "keyType": "ec256",
	}, 200))
	if bound["nginxDataSourceId"] != nginx["id"] {
		t.Fatalf("绑定 Nginx 失败: %v", bound["nginxDataSourceId"])
	}
	logBeforeDeploy := readFile(t, filepath.Join(dir, "nginx.log"))
	deployedOnly := decodeData(t, mustPost(t, base, "/api/v1/certificate/deploy", token, map[string]any{"id": only["id"]}, 200))
	if deployedOnly["status"] != "success" || deployedOnly["result"].(map[string]any)["deployed"] != true {
		t.Fatalf("部署已申请证书失败: %+v", deployedOnly)
	}
	if readFile(t, filepath.Join(dir, "nginx.log")) == logBeforeDeploy {
		t.Fatal("部署时应重启 Nginx")
	}
	if got := readCert(t, onlyFile); len(got.DNSNames) != 1 {
		t.Fatalf("部署后的证书=%v", got.DNSNames)
	}

	page := mustPost(t, base, "/api/v1/query/search", token, map[string]any{
		"model": "pageCertificate", "page": 1, "size": 20, "keyword": "example.com", "filters": map[string]any{},
	}, 200)
	if page["responseType"] != "page" || page["total"].(float64) < 1 {
		t.Fatalf("query=%v", page)
	}
	if strings.Contains(mustJSON(page), "PRIVATE KEY") || strings.Contains(mustJSON(page), "keyPem") {
		t.Fatal("通用查询不应返回私钥")
	}
	one := decodeData(t, mustPost(t, base, "/api/v1/query/one", token, map[string]any{
		"model": "user", "filters": map[string]any{"username": "admin"},
	}, 200))
	if _, ok := one["passwordHash"]; ok {
		t.Fatal("通用查询不应返回密码哈希")
	}

	roles := mustPost(t, base, "/api/v1/role/list", token, map[string]any{"page": 1, "size": 20}, 200)
	var viewerID float64
	for _, item := range roles["data"].([]any) {
		role := item.(map[string]any)
		if role["code"] == "viewer" {
			viewerID = role["id"].(float64)
		}
	}
	mustPost(t, base, "/api/v1/user/create", token, map[string]any{
		"username": "viewer1", "password": "Viewer@123456", "displayName": "只读", "roleIds": []any{viewerID},
	}, 200)
	viewerLogin := decodeData(t, mustPost(t, base, "/api/v1/auth/login", "", map[string]any{
		"username": "viewer1", "password": "Viewer@123456",
	}, 200))
	mustPost(t, base, "/api/v1/certificate/create", viewerLogin["token"].(string), map[string]any{
		"name": "denied", "domains": []string{"denied.example.com"}, "agreeTos": true,
		"acmeDataSourceId": acme["id"], "nginxDataSourceId": nginx["id"],
	}, 403)
	mustPost(t, base, "/api/v1/certificate/apply", viewerLogin["token"].(string), map[string]any{"id": only["id"], "force": true}, 403)
	mustPost(t, base, "/api/v1/certificate/download", viewerLogin["token"].(string), map[string]any{"id": only["id"]}, 403)

	otherTenant := auth.Tenant{Name: "其他租户", Code: "other", Status: auth.StatusActive}
	if err := db.Create(&otherTenant).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("Other@123456"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	otherUser := auth.User{TenantID: otherTenant.ID, Username: "other", PasswordHash: string(hash), DisplayName: "其他", Status: auth.StatusActive}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatal(err)
	}
	otherRole := auth.Role{TenantID: otherTenant.ID, Code: auth.RoleAdmin, Name: "管理员", BuiltIn: true}
	if err := db.Create(&otherRole).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&auth.UserRole{UserID: otherUser.ID, RoleID: otherRole.ID}).Error; err != nil {
		t.Fatal(err)
	}
	otherLogin := decodeData(t, mustPost(t, base, "/api/v1/auth/login", "", map[string]any{
		"username": "other", "password": "Other@123456",
	}, 200))
	otherToken := otherLogin["token"].(string)
	otherList := mustPost(t, base, "/api/v1/certificate/list", otherToken, map[string]any{"page": 1, "size": 20}, 200)
	if otherList["total"].(float64) != 0 {
		t.Fatalf("租户隔离失败: %v", otherList)
	}
	mustPost(t, base, "/api/v1/certificate/detail", otherToken, map[string]any{"id": cert["id"]}, 404)
	mustPost(t, base, "/api/v1/certificate/download", otherToken, map[string]any{"id": cert["id"], "format": "nginx"}, 404)

	home, homeBody := raw(t, http.MethodGet, base, "/", "", nil)
	if home != http.StatusOK || !strings.Contains(homeBody, `id="app"`) {
		t.Fatalf("SPA status=%d body=%s", home, homeBody)
	}
	catalog := decodeData(t, mustPost(t, base, "/api/v1/system/catalog", token, map[string]any{}, 200))
	routes, _ := catalog["routes"].([]any)
	if len(routes) < 10 {
		t.Fatalf("catalog routes=%d", len(routes))
	}
}

func mustPost(t *testing.T, base, path, token string, body any, want int) map[string]any {
	t.Helper()
	status, text := raw(t, http.MethodPost, base, path, token, body)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("%s 响应不是 JSON: %s", path, text)
	}
	if status != want {
		t.Fatalf("%s status=%d body=%s", path, status, text)
	}
	if want == 200 && out["code"].(float64) != 200 {
		t.Fatalf("%s code=%v", path, out["code"])
	}
	return out
}

func decodeData(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data=%v", body["data"])
	}
	return data
}

func raw(t *testing.T, method, base, path, token string, body any) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatal(err)
		}
		reader = buf
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(payload)
}

func waitTask(t *testing.T, base, token string, id any) map[string]any {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		body := decodeData(t, mustPost(t, base, "/api/v1/task/detail", token, map[string]any{"id": id}, 200))
		status, _ := body["status"].(string)
		if status == "success" || status == "failed" || status == "skipped" {
			return body
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("任务超时")
	return nil
}

func unzipDownload(t *testing.T, body map[string]any) map[string][]byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(body["contentBase64"].(string))
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, file := range zr.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[file.Name] = data
	}
	return out
}

func readCert(t *testing.T, path string) *x509.Certificate {
	t.Helper()
	return readCertPEM(t, readFile(t, path))
}

func readCertPEM(t *testing.T, raw string) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		t.Fatalf("证书 PEM 无效: %s", raw)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
