const CRC_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n++) {
    let c = n
    for (let k = 0; k < 8; k++) c = (c & 1) ? (0xedb88320 ^ (c >>> 1)) : (c >>> 1)
    table[n] = c >>> 0
  }
  return table
})()

function crc32(data) {
  let c = 0xffffffff
  for (let i = 0; i < data.length; i++) c = CRC_TABLE[(c ^ data[i]) & 0xff] ^ (c >>> 8)
  return (c ^ 0xffffffff) >>> 0
}

function pemBody(text) {
  return String(text || '').trim()
}

function withNewline(text) {
  return text ? `${text}\n` : ''
}

// certificateFileBase 用主域名做文件名。通配符 *.nbssas.com 去掉 *. 后是 nbssas.com。
export function certificateFileBase(domains) {
  const list = (Array.isArray(domains) ? domains : [])
    .map((item) => String(item).trim().toLowerCase())
    .filter(Boolean)
  const wildcard = list.find((item) => item.startsWith('*.'))
  const raw = (wildcard || list[0] || 'certificate').replace(/^\*\./, '')
  const safe = raw.replace(/[^a-z0-9._-]+/g, '').replace(/^\.+/, '')
  return safe || 'certificate'
}

// nginxCertificatePem 是 Nginx ssl_certificate 用的证书链，不包含私钥。
export function nginxCertificatePem(certPem, issuerPem) {
  const cert = pemBody(certPem)
  const issuer = pemBody(issuerPem)
  if (!cert) return withNewline(issuer)
  if (!issuer || issuer === cert || cert.includes(issuer)) return withNewline(cert)
  return `${cert}\n${issuer}\n`
}

export function nginxPrivateKey(keyPem) {
  return withNewline(pemBody(keyPem))
}

function dosDateTime(date) {
  const time = (date.getHours() << 11) | (date.getMinutes() << 5) | (date.getSeconds() >> 1)
  const day = ((date.getFullYear() - 1980) << 9) | ((date.getMonth() + 1) << 5) | date.getDate()
  return { time, day }
}

function writeUint16(view, offset, value) {
  view.setUint16(offset, value, true)
}

function writeUint32(view, offset, value) {
  view.setUint32(offset, value, true)
}

function buildStoredZip(files, date = new Date()) {
  const encoder = new TextEncoder()
  const stamp = dosDateTime(date)
  const encoded = files.map((file) => {
    const name = encoder.encode(file.name)
    const data = encoder.encode(file.data)
    return { name, data, crc: crc32(data), mode: file.mode }
  })
  const localSize = encoded.reduce((sum, file) => sum + 30 + file.name.length + file.data.length, 0)
  const centralSize = encoded.reduce((sum, file) => sum + 46 + file.name.length, 0)
  const out = new Uint8Array(localSize + centralSize + 22)
  const view = new DataView(out.buffer)
  let offset = 0
  const locals = []
  for (const file of encoded) {
    locals.push(offset)
    writeUint32(view, offset, 0x04034b50)
    writeUint16(view, offset + 4, 20)
    writeUint16(view, offset + 6, 0x0800)
    writeUint16(view, offset + 8, 0)
    writeUint16(view, offset + 10, stamp.time)
    writeUint16(view, offset + 12, stamp.day)
    writeUint32(view, offset + 14, file.crc)
    writeUint32(view, offset + 18, file.data.length)
    writeUint32(view, offset + 22, file.data.length)
    writeUint16(view, offset + 26, file.name.length)
    writeUint16(view, offset + 28, 0)
    out.set(file.name, offset + 30)
    out.set(file.data, offset + 30 + file.name.length)
    offset += 30 + file.name.length + file.data.length
  }
  const centralOffset = offset
  encoded.forEach((file, index) => {
    writeUint32(view, offset, 0x02014b50)
    writeUint16(view, offset + 4, 0x0314)
    writeUint16(view, offset + 6, 20)
    writeUint16(view, offset + 8, 0x0800)
    writeUint16(view, offset + 10, 0)
    writeUint16(view, offset + 12, stamp.time)
    writeUint16(view, offset + 14, stamp.day)
    writeUint32(view, offset + 16, file.crc)
    writeUint32(view, offset + 20, file.data.length)
    writeUint32(view, offset + 24, file.data.length)
    writeUint16(view, offset + 28, file.name.length)
    writeUint16(view, offset + 30, 0)
    writeUint16(view, offset + 32, 0)
    writeUint16(view, offset + 34, 0)
    writeUint16(view, offset + 36, 0)
    writeUint32(view, offset + 38, (0o100000 | file.mode) << 16)
    writeUint32(view, offset + 42, locals[index])
    out.set(file.name, offset + 46)
    offset += 46 + file.name.length
  })
  writeUint32(view, offset, 0x06054b50)
  writeUint16(view, offset + 4, 0)
  writeUint16(view, offset + 6, 0)
  writeUint16(view, offset + 8, encoded.length)
  writeUint16(view, offset + 10, encoded.length)
  writeUint32(view, offset + 12, offset - centralOffset)
  writeUint32(view, offset + 16, centralOffset)
  writeUint16(view, offset + 20, 0)
  return out
}

// buildCertificateZip 打包 Nginx 可用的域名.pem 和域名.key。
export function buildCertificateZip({ domains, certPem, keyPem, issuerPem, date }) {
  const base = certificateFileBase(domains)
  const bytes = buildStoredZip([
    { name: `${base}.pem`, data: nginxCertificatePem(certPem, issuerPem), mode: 0o644 },
    { name: `${base}.key`, data: nginxPrivateKey(keyPem), mode: 0o600 }
  ], date)
  return { filename: `${base}.zip`, bytes }
}
