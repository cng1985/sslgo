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
