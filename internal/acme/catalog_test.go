package acme

import "testing"

func TestResolveCA(t *testing.T) {
	ca, directory, eab, err := ResolveCA("zerossl", "", "")
	if err != nil || ca != "zerossl" || !eab || directory == "" {
		t.Fatalf("zerossl ca=%s dir=%s eab=%v err=%v", ca, directory, eab, err)
	}
	ca, directory, eab, err = ResolveCA("", "https://acme-v02.api.letsencrypt.org/directory", "")
	if err != nil || ca != "letsencrypt" || eab {
		t.Fatalf("infer ca=%s eab=%v err=%v", ca, eab, err)
	}
	ca, _, _, err = ResolveCA("custom", "https://acme.example.com/directory", "")
	if err != nil || ca != "custom" {
		t.Fatalf("custom ca=%s err=%v", ca, err)
	}
	if _, _, _, err = ResolveCA("missing", "", ""); err == nil {
		t.Fatal("未知机构应拒绝")
	}
}

func TestValidateDNS(t *testing.T) {
	if err := ValidateDNS("", nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDNS(DNSAliyun, map[string]string{"accessKeyId": "id"}); err == nil {
		t.Fatal("缺少 Secret 应拒绝")
	}
	if err := ValidateDNS(DNSAliyun, map[string]string{"accessKeyId": "id", "accessKeySecret": "sec"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDNS(DNSCloudflare, map[string]string{"email": "a@b.c"}); err == nil {
		t.Fatal("Cloudflare 只有邮箱应拒绝")
	}
	if err := ValidateDNS(DNSCloudflare, map[string]string{"apiToken": "token"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDNS(DNSAzure, map[string]string{
		"subscriptionId": "sub", "resourceGroup": "rg", "tenantId": "tid",
		"clientId": "cid", "clientSecret": "sec", "environment": "china",
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDNS(DNSGoogle, map[string]string{"serviceAccountJSON": `{"project_id":"p","private_key":"k","client_email":"a@p.iam"}`}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDNS("other", map[string]string{}); err == nil {
		t.Fatal("未知服务商应拒绝")
	}
}

func TestSecretFieldsHideKeys(t *testing.T) {
	fields := SecretFields(DNSAliyun)
	found := false
	for _, field := range fields {
		if field == "accessKeySecret" {
			found = true
		}
		if field == "region" {
			t.Fatal("地域不是密钥")
		}
	}
	if !found {
		t.Fatalf("fields=%v", fields)
	}
}
