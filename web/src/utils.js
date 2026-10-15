export function fmt(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  const p = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())} ${p(date.getHours())}:${p(date.getMinutes())}`
}

export function remainText(days, notAfter) {
  let value = days
  if (value === undefined || value === null || value === '') {
    if (!notAfter) return ''
    const ms = new Date(notAfter).getTime() - Date.now()
    if (Number.isNaN(ms)) return ''
    value = ms >= 0 ? Math.floor(ms / 86400000) : -Math.ceil(-ms / 86400000)
  }
  value = Number(value)
  if (Number.isNaN(value)) return ''
  if (value < 0) return `已过期 ${-value} 天`
  if (value === 0) return '不足 1 天'
  return `剩余 ${value} 天`
}

export function statusType(status) {
  const map = {
    deployed: 'success',
    success: 'success',
    issued: 'success',
    active: 'success',
    skipped: 'info',
    draft: 'info',
    idle: 'info',
    pending: 'warning',
    running: 'warning',
    issuing: 'warning',
    failed: 'danger',
    disabled: 'info'
  }
  return map[status] || 'info'
}

// 通配符 *.nbssas.com 下载后使用 nbssas.com.pem 与 nbssas.com.key。
export function certificateFileStem(domains) {
  const list = Array.isArray(domains) ? domains : []
  const primary = String(list.find((item) => String(item || '').trim()) || 'certificate').trim().toLowerCase()
  const stem = primary.replace(/^\*\./, '').replace(/[^a-z0-9.-]/g, '')
  return stem && stem !== '.' && stem !== '..' ? stem : 'certificate'
}

function joinPem(...parts) {
  const body = parts.map((part) => String(part || '').trim()).filter(Boolean).join('\n')
  return body ? `${body}\n` : ''
}

const CRC_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let i = 0; i < 256; i++) {
    let crc = i
    for (let bit = 0; bit < 8; bit++) crc = crc & 1 ? 0xedb88320 ^ (crc >>> 1) : crc >>> 1
    table[i] = crc >>> 0
  }
  return table
})()

function crc32(bytes) {
  let crc = 0xffffffff
  for (let i = 0; i < bytes.length; i++) crc = CRC_TABLE[(crc ^ bytes[i]) & 0xff] ^ (crc >>> 8)
  return (crc ^ 0xffffffff) >>> 0
}

function dosDateTime(date) {
  const value = date instanceof Date ? date : new Date()
  const time = (value.getHours() << 11) | (value.getMinutes() << 5) | (value.getSeconds() >> 1)
  const day = ((Math.max(value.getFullYear(), 1980) - 1980) << 9) | ((value.getMonth() + 1) << 5) | value.getDate()
  return { time, day }
}

export function zipStore(entries, when = new Date()) {
  const encoder = new TextEncoder()
  const { time, day } = dosDateTime(when)
  const locals = []
  const centrals = []
  let offset = 0
  for (const entry of entries) {
    const name = encoder.encode(entry.name)
    const data = typeof entry.data === 'string' ? encoder.encode(entry.data) : entry.data
    const crc = crc32(data)
    const local = new Uint8Array(30 + name.length + data.length)
    const view = new DataView(local.buffer)
    view.setUint32(0, 0x04034b50, true)
    view.setUint16(4, 20, true)
    view.setUint16(8, 0, true)
    view.setUint16(10, time, true)
    view.setUint16(12, day, true)
    view.setUint32(14, crc, true)
    view.setUint32(18, data.length, true)
    view.setUint32(22, data.length, true)
    view.setUint16(26, name.length, true)
    local.set(name, 30)
    local.set(data, 30 + name.length)
    locals.push(local)

    const central = new Uint8Array(46 + name.length)
    const centralView = new DataView(central.buffer)
    centralView.setUint32(0, 0x02014b50, true)
    centralView.setUint16(4, 20, true)
    centralView.setUint16(6, 20, true)
    centralView.setUint16(10, 0, true)
    centralView.setUint16(12, time, true)
    centralView.setUint16(14, day, true)
    centralView.setUint32(16, crc, true)
    centralView.setUint32(20, data.length, true)
    centralView.setUint32(24, data.length, true)
    centralView.setUint16(28, name.length, true)
    centralView.setUint32(42, offset, true)
    central.set(name, 46)
    centrals.push(central)
    offset += local.length
  }
  const centralSize = centrals.reduce((sum, part) => sum + part.length, 0)
  const end = new Uint8Array(22)
  const endView = new DataView(end.buffer)
  endView.setUint32(0, 0x06054b50, true)
  endView.setUint16(8, entries.length, true)
  endView.setUint16(10, entries.length, true)
  endView.setUint32(12, centralSize, true)
  endView.setUint32(16, offset, true)
  const out = new Uint8Array(offset + centralSize + end.length)
  let cursor = 0
  for (const part of [...locals, ...centrals, end]) {
    out.set(part, cursor)
    cursor += part.length
  }
  return out
}

export function buildCertificateZip(material) {
  const stem = certificateFileStem(material?.domains)
  const pem = joinPem(material?.certPem, material?.issuerPem)
  const key = joinPem(material?.keyPem)
  return {
    filename: `${stem}.zip`,
    bytes: zipStore([
      { name: `${stem}.pem`, data: pem },
      { name: `${stem}.key`, data: key }
    ])
  }
}

export function statusText(status) {
  const map = {
    deployed: '已部署',
    issued: '已签发',
    draft: '草稿',
    issuing: '签发中',
    failed: '失败',
    success: '成功',
    skipped: '已跳过',
    pending: '等待',
    running: '执行中',
    idle: '空闲',
    active: '启用',
    disabled: '停用',
    triggered: '已触发'
  }
  return map[status] || status || '—'
}
