package acme

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/azuredns"
	"github.com/go-acme/lego/v4/providers/dns/baiducloud"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/digitalocean"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/gcloud"
	"github.com/go-acme/lego/v4/providers/dns/huaweicloud"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/go-acme/lego/v4/providers/dns/volcengine"
)

// NewDNSProvider 按云厂商和凭证构造 DNS-01 挑战客户端。
func NewDNSProvider(name string, values map[string]string) (challenge.Provider, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "none" {
		return nil, fmt.Errorf("未绑定 DNS 服务商")
	}
	clean := map[string]string{}
	for key, value := range values {
		clean[key] = strings.TrimSpace(value)
	}
	if err := ValidateDNS(name, clean); err != nil {
		return nil, err
	}
	builder, ok := dnsBuilders[name]
	if !ok {
		return nil, fmt.Errorf("不支持的 DNS 服务商")
	}
	provider, err := builder(clean)
	if err != nil {
		return nil, fmt.Errorf("%s", clipErr(err))
	}
	return provider, nil
}

var dnsBuilders = map[string]func(map[string]string) (challenge.Provider, error){
	DNSAliyun:       aliyunProvider,
	DNSTencent:      tencentProvider,
	DNSPod:          dnspodProvider,
	DNSHuawei:       huaweiProvider,
	DNSBaidu:        baiduProvider,
	DNSVolcengine:   volcengineProvider,
	DNSRoute53:      route53Provider,
	DNSAzure:        azureProvider,
	DNSGoogle:       googleProvider,
	DNSCloudflare:   cloudflareProvider,
	DNSDigitalOcean: digitalOceanProvider,
}

func aliyunProvider(v map[string]string) (challenge.Provider, error) {
	cfg := alidns.NewDefaultConfig()
	cfg.APIKey = v["accessKeyId"]
	cfg.SecretKey = v["accessKeySecret"]
	cfg.SecurityToken = v["securityToken"]
	if v["region"] != "" {
		cfg.RegionID = v["region"]
	}
	return alidns.NewDNSProviderConfig(cfg)
}

func tencentProvider(v map[string]string) (challenge.Provider, error) {
	cfg := tencentcloud.NewDefaultConfig()
	cfg.SecretID = v["secretId"]
	cfg.SecretKey = v["secretKey"]
	cfg.Region = v["region"]
	cfg.SessionToken = v["sessionToken"]
	return tencentcloud.NewDNSProviderConfig(cfg)
}

func dnspodProvider(v map[string]string) (challenge.Provider, error) {
	cfg := dnspod.NewDefaultConfig()
	cfg.LoginToken = v["loginToken"]
	return dnspod.NewDNSProviderConfig(cfg)
}

func huaweiProvider(v map[string]string) (challenge.Provider, error) {
	cfg := huaweicloud.NewDefaultConfig()
	cfg.AccessKeyID = v["accessKeyId"]
	cfg.SecretAccessKey = v["secretAccessKey"]
	cfg.Region = v["region"]
	return huaweicloud.NewDNSProviderConfig(cfg)
}

func baiduProvider(v map[string]string) (challenge.Provider, error) {
	cfg := baiducloud.NewDefaultConfig()
	cfg.AccessKeyID = v["accessKeyId"]
	cfg.SecretAccessKey = v["secretAccessKey"]
	return baiducloud.NewDNSProviderConfig(cfg)
}

func volcengineProvider(v map[string]string) (challenge.Provider, error) {
	cfg := volcengine.NewDefaultConfig()
	cfg.AccessKey = v["accessKey"]
	cfg.SecretKey = v["secretKey"]
	if v["region"] != "" {
		cfg.Region = v["region"]
	}
	return volcengine.NewDNSProviderConfig(cfg)
}

func route53Provider(v map[string]string) (challenge.Provider, error) {
	cfg := route53.NewDefaultConfig()
	cfg.AccessKeyID = v["accessKeyId"]
	cfg.SecretAccessKey = v["secretAccessKey"]
	cfg.SessionToken = v["sessionToken"]
	cfg.HostedZoneID = v["hostedZoneId"]
	cfg.Region = v["region"]
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	return route53.NewDNSProviderConfig(cfg)
}

func azureProvider(v map[string]string) (challenge.Provider, error) {
	cfg := azuredns.NewDefaultConfig()
	cfg.SubscriptionID = v["subscriptionId"]
	cfg.ResourceGroup = v["resourceGroup"]
	cfg.TenantID = v["tenantId"]
	cfg.ClientID = v["clientId"]
	cfg.ClientSecret = v["clientSecret"]
	cfg.ZoneName = v["zoneName"]
	cfg.AuthMethod = "env"
	switch strings.ToLower(v["environment"]) {
	case "", "public":
		cfg.Environment = cloud.AzurePublic
	case "china":
		cfg.Environment = cloud.AzureChina
	case "usgovernment":
		cfg.Environment = cloud.AzureGovernment
	default:
		return nil, fmt.Errorf("Azure 环境只能是 public、china 或 usgovernment")
	}
	return azuredns.NewDNSProviderConfig(cfg)
}

func googleProvider(v map[string]string) (challenge.Provider, error) {
	return gcloud.NewDNSProviderServiceAccountKey([]byte(v["serviceAccountJSON"]))
}

func cloudflareProvider(v map[string]string) (challenge.Provider, error) {
	cfg := cloudflare.NewDefaultConfig()
	cfg.AuthToken = v["apiToken"]
	cfg.AuthEmail = v["email"]
	cfg.AuthKey = v["apiKey"]
	return cloudflare.NewDNSProviderConfig(cfg)
}

func digitalOceanProvider(v map[string]string) (challenge.Provider, error) {
	cfg := digitalocean.NewDefaultConfig()
	cfg.AuthToken = v["token"]
	return digitalocean.NewDNSProviderConfig(cfg)
}
