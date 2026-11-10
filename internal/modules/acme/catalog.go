package acme

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	DNSAliyun       = "aliyun"
	DNSTencent      = "tencent"
	DNSPod          = "dnspod"
	DNSHuawei       = "huawei"
	DNSBaidu        = "baidu"
	DNSVolcengine   = "volcengine"
	DNSRoute53      = "route53"
	DNSAzure        = "azure"
	DNSGoogle       = "gcloud"
	DNSCloudflare   = "cloudflare"
	DNSDigitalOcean = "digitalocean"
)

// CA 是可选的 ACME 证书机构。
type CA struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DirectoryURL string `json:"directoryURL"`
	EAB          bool   `json:"eab"`
	Note         string `json:"note,omitempty"`
}

// Field 是云厂商 DNS 凭证里的一个输入项。
type Field struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Secret      bool   `json:"secret"`
	Required    bool   `json:"required"`
	Multiline   bool   `json:"multiline,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

// DNSProviderSpec 描述一家可对接的云厂商 DNS。
type DNSProviderSpec struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Vendor   string     `json:"vendor"`
	Note     string     `json:"note,omitempty"`
	Fields   []Field    `json:"fields"`
	AnyOf    [][]string `json:"anyOf,omitempty"`
	validate func(map[string]string) error
}

// ProviderCatalog 供管理台渲染 ACME 账户表单。
type ProviderCatalog struct {
	CAs          []CA              `json:"cas"`
	DNSProviders []DNSProviderSpec `json:"dnsProviders"`
}

// Catalog 返回支持的证书机构和云厂商 DNS。
func Catalog() ProviderCatalog {
	return ProviderCatalog{CAs: caPresets(), DNSProviders: dnsPresets()}
}

func caPresets() []CA {
	return []CA{
		{ID: "letsencrypt", Name: "Let's Encrypt 生产环境", DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory", Note: "浏览器信任。正式域名再用。"},
		{ID: "letsencrypt-staging", Name: "Let's Encrypt 测试环境", DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory", Note: "不消耗生产额度，证书不受浏览器信任。"},
		{ID: "zerossl", Name: "ZeroSSL", DirectoryURL: "https://acme.zerossl.com/v2/DV90", EAB: true, Note: "需要在 ZeroSSL 控制台创建 External Account Binding。"},
		{ID: "google", Name: "Google Trust Services", DirectoryURL: "https://dv.acme-v02.api.pki.goog/directory", EAB: true, Note: "需要 Google Public CA 的 EAB 密钥。"},
		{ID: "buypass", Name: "Buypass", DirectoryURL: "https://api.buypass.com/acme/directory"},
		{ID: "buypass-test", Name: "Buypass 测试环境", DirectoryURL: "https://api.test4.buypass.no/acme/directory", Note: "测试证书不受浏览器信任。"},
		{ID: "sslcom", Name: "SSL.com", DirectoryURL: "https://acme.ssl.com/sslcom-dv-rsa", EAB: true, Note: "需要 SSL.com 提供的 EAB 密钥。"},
		{ID: "custom", Name: "自定义目录", Note: "填写任意兼容 RFC 8555 的 ACME 目录。"},
	}
}

func dnsPresets() []DNSProviderSpec {
	secret := func(name, label, placeholder, help string) Field {
		return Field{Name: name, Label: label, Secret: true, Required: true, Placeholder: placeholder, Help: help}
	}
	optionalSecret := func(name, label, help string) Field {
		return Field{Name: name, Label: label, Secret: true, Help: help}
	}
	text := func(name, label, placeholder, help string, required bool) Field {
		return Field{Name: name, Label: label, Required: required, Placeholder: placeholder, Help: help}
	}
	return []DNSProviderSpec{
		{
			ID: DNSAliyun, Name: "阿里云 DNS", Vendor: "阿里云",
			Note: "使用访问控制里的 AccessKey。建议只授予 AliyunDNSFullAccess。",
			Fields: []Field{
				secret("accessKeyId", "AccessKey ID", "LTAI...", ""),
				secret("accessKeySecret", "AccessKey Secret", "", ""),
				text("region", "地域", "cn-hangzhou", "留空默认 cn-hangzhou。", false),
				optionalSecret("securityToken", "STS Token", "使用临时凭证时填写。"),
			},
		},
		{
			ID: DNSTencent, Name: "腾讯云 DNS", Vendor: "腾讯云",
			Note: "使用 API 密钥访问 DNSPod。建议只授予 QcloudDNSPodFullAccess。",
			Fields: []Field{
				secret("secretId", "SecretId", "AKID...", ""),
				secret("secretKey", "SecretKey", "", ""),
				text("region", "地域", "ap-guangzhou", "留空使用腾讯云默认地域。", false),
				optionalSecret("sessionToken", "会话令牌", "使用临时密钥时填写。"),
			},
		},
		{
			ID: DNSPod, Name: "DNSPod Token", Vendor: "腾讯云",
			Note: "适合仍使用 DNSPod API Token 的账号，格式为 ID,Token。",
			Fields: []Field{
				secret("loginToken", "Login Token", "12345,abcdefghijklmnopqrst", "在 DNSPod 控制台的密钥管理中创建。"),
			},
		},
		{
			ID: DNSHuawei, Name: "华为云 DNS", Vendor: "华为云",
			Note: "使用 IAM 访问密钥。地域示例：cn-north-4、cn-east-3、cn-south-1。",
			Fields: []Field{
				secret("accessKeyId", "Access Key ID", "", ""),
				secret("secretAccessKey", "Secret Access Key", "", ""),
				text("region", "地域", "cn-north-4", "", true),
			},
		},
		{
			ID: DNSBaidu, Name: "百度智能云 DNS", Vendor: "百度智能云",
			Fields: []Field{
				secret("accessKeyId", "Access Key ID", "", ""),
				secret("secretAccessKey", "Secret Access Key", "", ""),
			},
		},
		{
			ID: DNSVolcengine, Name: "火山引擎 DNS", Vendor: "火山引擎",
			Fields: []Field{
				secret("accessKey", "Access Key", "", ""),
				secret("secretKey", "Secret Key", "", ""),
				text("region", "地域", "cn-north-1", "留空默认 cn-north-1。", false),
			},
		},
		{
			ID: DNSRoute53, Name: "Amazon Route 53", Vendor: "Amazon Web Services",
			Note: "IAM 用户需要 route53:ChangeResourceRecordSets 和 route53:ListHostedZones。",
			Fields: []Field{
				secret("accessKeyId", "Access Key ID", "AKIA...", ""),
				secret("secretAccessKey", "Secret Access Key", "", ""),
				text("region", "地域", "us-east-1", "留空默认 us-east-1。", false),
				text("hostedZoneId", "Hosted Zone ID", "", "留空时按域名查找公有托管区。", false),
				optionalSecret("sessionToken", "Session Token", "使用临时凭证时填写。"),
			},
		},
		{
			ID: DNSAzure, Name: "Azure DNS", Vendor: "Microsoft Azure",
			Note: "使用应用注册的客户端密码。中国区环境填 china。",
			Fields: []Field{
				text("subscriptionId", "订阅 ID", "", "", true),
				text("resourceGroup", "资源组", "", "", true),
				text("tenantId", "租户 ID", "", "", true),
				text("clientId", "客户端 ID", "", "", true),
				secret("clientSecret", "客户端密码", "", ""),
				text("zoneName", "区域名称", "example.com", "留空时列出租户内的托管区域再匹配域名。", false),
				text("environment", "环境", "public", "public、china 或 usgovernment。", false),
			},
			validate: validateAzure,
		},
		{
			ID: DNSGoogle, Name: "Google Cloud DNS", Vendor: "Google Cloud",
			Note: "上传具备 DNS 管理员权限的服务账号 JSON。",
			Fields: []Field{
				{Name: "serviceAccountJSON", Label: "服务账号 JSON", Secret: true, Required: true, Multiline: true, Help: "包含 project_id、client_email 和 private_key。"},
				text("project", "项目 ID", "", "留空则读取 JSON 中的 project_id。", false),
			},
			validate: validateGoogle,
		},
		{
			ID: DNSCloudflare, Name: "Cloudflare", Vendor: "Cloudflare",
			Note: "优先使用只含 Zone:Read 和 DNS:Edit 的 API Token。",
			Fields: []Field{
				optionalSecret("apiToken", "API Token", ""),
				text("email", "账号邮箱", "", "与 Global API Key 一起使用。", false),
				optionalSecret("apiKey", "Global API Key", ""),
			},
			AnyOf: [][]string{{"apiToken"}, {"email", "apiKey"}},
		},
		{
			ID: DNSDigitalOcean, Name: "DigitalOcean", Vendor: "DigitalOcean",
			Fields: []Field{
				secret("token", "API Token", "", "需要域名读写权限。"),
			},
		},
	}
}

func validateAzure(values map[string]string) error {
	switch strings.ToLower(strings.TrimSpace(values["environment"])) {
	case "", "public", "china", "usgovernment":
		return nil
	default:
		return fmt.Errorf("Azure 环境只能是 public、china 或 usgovernment")
	}
}

func validateGoogle(values map[string]string) error {
	var meta struct {
		ProjectID   string `json:"project_id"`
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}
	raw := strings.TrimSpace(values["serviceAccountJSON"])
	if json.Unmarshal([]byte(raw), &meta) != nil || meta.ProjectID == "" || meta.PrivateKey == "" || meta.ClientEmail == "" {
		return fmt.Errorf("Google Cloud 服务账号 JSON 无效")
	}
	return nil
}

// ResolveCA 根据选择的机构和目录得到最终 ACME 目录，并说明是否需要 EAB。
func ResolveCA(ca, directory, fallback string) (string, string, bool, error) {
	ca = strings.TrimSpace(ca)
	directory = strings.TrimRight(strings.TrimSpace(directory), "/")
	fallback = strings.TrimRight(strings.TrimSpace(fallback), "/")
	presets := caPresets()
	byID := map[string]CA{}
	for _, item := range presets {
		byID[item.ID] = item
	}
	if directory == "" && ca != "" && ca != "custom" {
		item, ok := byID[ca]
		if !ok {
			return "", "", false, fmt.Errorf("未知证书机构")
		}
		return item.ID, strings.TrimRight(item.DirectoryURL, "/"), item.EAB, nil
	}
	if directory == "" {
		if fallback == "" {
			return "", "", false, fmt.Errorf("请填写 ACME 目录")
		}
		directory = fallback
	}
	if err := validDirectory(directory); err != nil {
		return "", "", false, err
	}
	for _, item := range presets {
		if item.DirectoryURL != "" && strings.TrimRight(item.DirectoryURL, "/") == directory {
			return item.ID, directory, item.EAB, nil
		}
	}
	if ca == "" {
		ca = "custom"
	}
	if ca != "custom" {
		if _, ok := byID[ca]; !ok {
			return "", "", false, fmt.Errorf("未知证书机构")
		}
	}
	return ca, directory, byID[ca].EAB, nil
}

func validDirectory(directory string) error {
	u, err := url.Parse(directory)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("ACME 目录地址无效")
	}
	return nil
}

// RequiresEAB 判断该机构注册账户时是否必须提供外部账号绑定。
func RequiresEAB(ca, directory string) bool {
	_, _, eab, err := ResolveCA(ca, directory, "")
	return err == nil && eab
}

// ValidateDNS 检查云厂商凭证是否齐全。provider 为空表示不绑定 DNS。
func ValidateDNS(provider string, values map[string]string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" || provider == "none" {
		return nil
	}
	spec, ok := findDNS(provider)
	if !ok {
		return fmt.Errorf("未知 DNS 服务商")
	}
	if len(spec.AnyOf) > 0 {
		matched := false
		for _, group := range spec.AnyOf {
			full := true
			for _, name := range group {
				if strings.TrimSpace(values[name]) == "" {
					full = false
					break
				}
			}
			if full {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s凭证不完整", spec.Name)
		}
	}
	for _, field := range spec.Fields {
		if field.Required && strings.TrimSpace(values[field.Name]) == "" {
			return fmt.Errorf("请填写%s", field.Label)
		}
	}
	if spec.validate != nil {
		return spec.validate(values)
	}
	return nil
}

// SecretFields 返回需要从接口响应中去掉的凭证字段。
func SecretFields(provider string) []string {
	spec, ok := findDNS(provider)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		if field.Secret {
			out = append(out, field.Name)
		}
	}
	return out
}

// DNSName 返回云厂商在界面上的名称。
func DNSName(provider string) string {
	spec, ok := findDNS(provider)
	if !ok {
		return provider
	}
	return spec.Name
}

func findDNS(id string) (DNSProviderSpec, bool) {
	for _, item := range dnsPresets() {
		if item.ID == id {
			return item, true
		}
	}
	return DNSProviderSpec{}, false
}
