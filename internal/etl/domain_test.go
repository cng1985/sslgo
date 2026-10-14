package etl

import "testing"

func TestNormalizeDomainsWildcard(t *testing.T) {
	domains, err := normalizeDomains([]string{"*.Example.COM", "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 || domains[0] != "*.example.com" {
		t.Fatalf("domains=%v", domains)
	}
	if _, err = normalizeDomains([]string{"foo.*.example.com"}); err == nil {
		t.Fatal("通配符不在最左侧时应拒绝")
	}
	if _, err = normalizeChallenge("http-01", domains); err == nil {
		t.Fatal("HTTP-01 不应接受通配符")
	}
	challenge, err := normalizeChallenge("dns-01", domains)
	if err != nil || challenge != ChallengeDNS01 {
		t.Fatalf("challenge=%s err=%v", challenge, err)
	}
	challenge, err = normalizeChallenge("", []string{"www.example.com"})
	if err != nil || challenge != ChallengeHTTP01 {
		t.Fatalf("default challenge=%s err=%v", challenge, err)
	}
}
