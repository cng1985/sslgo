package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// GenerateAccountKeyPEM 生成 ACME 账户使用的 P-256 私钥。
func GenerateAccountKeyPEM() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	raw, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw})), nil
}

// ParsePrivateKey 解析 PKCS8、SEC1 或 PKCS1 私钥。
func ParsePrivateKey(raw []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("私钥 PEM 无效")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		priv, ok := key.(crypto.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("无法解析私钥")
		}
		return priv, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("无法解析私钥")
}
