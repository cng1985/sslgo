package acme

import "testing"

func TestEveryDNSProviderHasBuilder(t *testing.T) {
	for _, spec := range Catalog().DNSProviders {
		if _, ok := dnsBuilders[spec.ID]; !ok {
			t.Fatalf("缺少 %s 的 DNS 客户端", spec.ID)
		}
	}
}

func TestAliyunAndCloudflareClients(t *testing.T) {
	aliyun, err := NewDNSProvider(DNSAliyun, map[string]string{
		"accessKeyId": "LTAItest", "accessKeySecret": "secret",
	})
	if err != nil || aliyun == nil {
		t.Fatal(err)
	}
	cf, err := NewDNSProvider(DNSCloudflare, map[string]string{"apiToken": "cf-token"})
	if err != nil || cf == nil {
		t.Fatal(err)
	}
	pod, err := NewDNSProvider(DNSPod, map[string]string{"loginToken": "1,token"})
	if err != nil || pod == nil {
		t.Fatal(err)
	}
	if _, err = NewDNSProvider(DNSHuawei, map[string]string{
		"accessKeyId": "ak", "secretAccessKey": "sk", "region": "not-a-region",
	}); err == nil {
		t.Fatal("无效华为云地域应拒绝")
	}
}
