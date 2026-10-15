import assert from 'node:assert/strict'
import test from 'node:test'
import { buildCertificateZip, certificateFileBase, nginxCertificatePem } from './certzip.js'

test('通配符域名去掉星号作为文件名', () => {
  assert.equal(certificateFileBase(['*.nbssas.com']), 'nbssas.com')
  assert.equal(certificateFileBase(['nbssas.com', '*.nbssas.com']), 'nbssas.com')
  assert.equal(certificateFileBase(['www.example.com']), 'www.example.com')
})

test('证书链不重复拼接相同颁发者', () => {
  const cert = '-----BEGIN CERTIFICATE-----\nLEAF\n-----END CERTIFICATE-----'
  const issuer = '-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----'
  assert.equal(nginxCertificatePem(cert, issuer), `${cert}\n${issuer}\n`)
  assert.equal(nginxCertificatePem(`${cert}\n${issuer}`, issuer), `${cert}\n${issuer}\n`)
  assert.equal(nginxCertificatePem(cert, cert), `${cert}\n`)
})

test('zip 内文件使用域名', () => {
  const packed = buildCertificateZip({
    domains: ['*.nbssas.com'],
    certPem: '-----BEGIN CERTIFICATE-----\nLEAF\n-----END CERTIFICATE-----\n',
    keyPem: '-----BEGIN PRIVATE KEY-----\nSECRET\n-----END PRIVATE KEY-----\n',
    issuerPem: '-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----\n',
    date: new Date(2026, 9, 15, 8, 0, 0)
  })
  assert.equal(packed.filename, 'nbssas.com.zip')
  const text = new TextDecoder().decode(packed.bytes)
  assert.match(text, /nbssas\.com\.pem/)
  assert.match(text, /nbssas\.com\.key/)
  assert.match(text, /LEAF/)
  assert.match(text, /CA/)
  assert.match(text, /SECRET/)
  assert.equal(packed.bytes[0], 0x50)
  assert.equal(packed.bytes[1], 0x4b)
})
