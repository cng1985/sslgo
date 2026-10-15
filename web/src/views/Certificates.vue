<template>
  <div>
    <div class="page-head">
      <div>
        <h1>证书</h1>
        <p>可以只向 ACME 申请，也可以申请后立刻部署到 Nginx。DNS-01 不选 Nginx 也能单独申请，私钥稍后下载。</p>
      </div>
      <el-button v-if="auth.can('certificate:create')" type="primary" @click="openCreate">新建证书</el-button>
    </div>
    <div class="panel">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="名称或域名" clearable style="width: 220px" @keyup.enter="load" />
        <el-select v-model="status" clearable placeholder="状态" style="width: 140px">
          <el-option v-for="item in statuses" :key="item" :label="statusText(item)" :value="item" />
        </el-select>
        <el-button @click="load">查询</el-button>
      </div>
      <el-table :data="rows" v-loading="loading">
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column label="域名" min-width="200">
          <template #default="{ row }"><span class="mono">{{ row.domains.join(' ') }}</span></template>
        </el-table-column>
        <el-table-column label="到期" min-width="210">
          <template #default="{ row }">
            <div>{{ fmt(row.notAfter) }}</div>
            <div v-if="row.notAfter" class="expire" :class="expireClass(row)">{{ remainText(row.daysRemaining, row.notAfter) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="460">
          <template #default="{ row }">
            <el-button v-if="auth.can('certificate:apply')" link type="primary" @click="apply(row)">申请</el-button>
            <el-button v-if="auth.can('certificate:issue') && row.nginxDataSourceId" link type="primary" @click="issue(row)">签发并部署</el-button>
            <el-button v-if="canDeploy(row)" link @click="deploy(row)">{{ row.status === 'deployed' ? '重启 Nginx' : '部署' }}</el-button>
            <el-dropdown v-if="canDownload(row)" trigger="click" @command="(format) => download(row, format)">
              <el-button link type="primary">下载</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="nginx">Nginx（fullchain + key）</el-dropdown-item>
                  <el-dropdown-item command="httpd">httpd（crt + key + chain）</el-dropdown-item>
                  <el-dropdown-item command="iis">IIS（PFX）</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button v-if="auth.can('certificate:delete')" link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="onPage" />
      </div>
    </div>

    <el-dialog v-model="open" title="新建证书" width="560px">
      <el-form label-position="top">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="域名，多个用空格或换行分隔"><el-input v-model="form.domainText" type="textarea" :rows="3" placeholder="example.com &#10;*.example.com" /></el-form-item>
        <el-form-item label="验证方式">
          <el-radio-group v-model="form.challengeType">
            <el-radio value="http-01">HTTP-01</el-radio>
            <el-radio value="dns-01">DNS-01</el-radio>
          </el-radio-group>
        </el-form-item>
        <p v-if="form.challengeType === 'dns-01'" class="hint">DNS-01 使用 ACME 账户里绑定的云厂商 DNS。通配符只能走这条路径。不选 Nginx 时只申请证书，不会部署。</p>
        <p v-else class="hint">HTTP-01 必须选择 Nginx，验证文件会写到它的 webroot。仍可以只申请，不立刻重启。</p>
        <el-form-item label="ACME 数据源">
          <el-select v-model="form.acmeDataSourceId" style="width: 100%">
            <el-option v-for="item in acmeSources" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Nginx 数据源">
          <el-select v-model="form.nginxDataSourceId" clearable placeholder="不选则只申请，不部署" style="width: 100%">
            <el-option v-for="item in nginxSources" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="密钥类型">
          <el-radio-group v-model="form.keyType">
            <el-radio value="ec256">EC P-256</el-radio>
            <el-radio value="rsa2048">RSA 2048</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="到期前续期天数"><el-input-number v-model="form.renewBeforeDays" :min="1" :max="60" /></el-form-item>
        <el-checkbox v-model="form.autoRenew">允许调度自动续期</el-checkbox>
        <el-checkbox v-model="form.agreeTos">同意 ACME 服务条款</el-checkbox>
      </el-form>
      <template #footer>
        <el-button @click="open = false">取消</el-button>
        <el-button type="primary" @click="create">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'
import { fmt, remainText, statusText, statusType } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const keyword = ref('')
const status = ref('')
const loading = ref(false)
const open = ref(false)
const sources = ref([])
const statuses = ['draft', 'issuing', 'issued', 'deployed', 'failed']
const form = reactive({
  name: '', domainText: '', acmeDataSourceId: null, nginxDataSourceId: null,
  challengeType: 'http-01', keyType: 'ec256', renewBeforeDays: 30, autoRenew: true, agreeTos: false
})
const acmeSources = computed(() => sources.value.filter((item) => item.type === 'acme'))
const nginxSources = computed(() => sources.value.filter((item) => item.type === 'nginx'))

async function load() {
  loading.value = true
  try {
    const body = await http.post('/certificate/list', {
      page: page.value, size: size.value,
      filters: { keyword: keyword.value, status: status.value }
    })
    rows.value = body.data
    total.value = body.total
  } finally {
    loading.value = false
  }
}

async function loadSources() {
  const body = await http.post('/datasource/list', { page: 1, size: 100, filters: { status: 'active' } })
  sources.value = body.data
}

function openCreate() {
  Object.assign(form, {
    name: '', domainText: '', acmeDataSourceId: null, nginxDataSourceId: null,
    challengeType: 'http-01', keyType: 'ec256', renewBeforeDays: 30, autoRenew: true, agreeTos: false
  })
  open.value = true
}

watch(() => form.domainText, (value) => {
  if (value.includes('*.')) form.challengeType = 'dns-01'
})

async function create() {
  await http.post('/certificate/create', {
    ...form,
    domains: form.domainText.split(/[\s,]+/).filter(Boolean)
  })
  open.value = false
  ElMessage.success('证书申请已创建')
  load()
}

function canDeploy(row) {
  return auth.can('certificate:deploy') && row.nginxDataSourceId && row.status !== 'issuing' && (row.status === 'issued' || row.status === 'deployed' || row.lastIssuedAt)
}

function canDownload(row) {
  return auth.can('certificate:download') && row.lastIssuedAt && row.status !== 'issuing'
}

async function apply(row) {
  try {
    await ElMessageBox.confirm(`将为 ${row.domains.join('、')} 向 ACME 申请证书，不会替换 Nginx 上的文件，也不会重启服务。`, '单独申请')
  } catch {
    return
  }
  const created = await http.post('/certificate/apply', { id: row.id, force: true })
  await finishTask(created.data.id)
}

async function issue(row) {
  try {
    await ElMessageBox.confirm(`将为 ${row.domains.join('、')} 申请证书，成功后替换旧文件并重启 Nginx。`, '签发并部署')
  } catch {
    return
  }
  const created = await http.post('/certificate/issue', { id: row.id, force: true })
  await finishTask(created.data.id)
}

async function finishTask(id) {
  ElMessage.info('任务已开始')
  const task = await poll(id)
  if (task.status === 'failed') {
    ElMessage.error(task.error || '签发失败')
    load()
    return
  }
  ElMessage.success(task.result?.message || statusText(task.status))
  load()
}

const bundleNames = { nginx: 'Nginx', httpd: 'httpd', iis: 'IIS' }

function expireClass(row) {
  const text = remainText(row.daysRemaining, row.notAfter)
  if (text.startsWith('已过期')) return 'bad'
  const days = Number(row.daysRemaining)
  if (!Number.isNaN(days) && days <= 30) return 'warn'
  return 'ok'
}

async function download(row, format) {
  let password = ''
  if (format === 'iis') {
    try {
      const res = await ElMessageBox.prompt('IIS 导入 PFX 需要密码，8 到 64 位。', '下载 IIS 证书', {
        inputType: 'password',
        inputPlaceholder: '设置导入密码',
        confirmButtonText: '下载',
        cancelButtonText: '取消',
        inputValidator: (value) => {
          const text = value || ''
          if (text.length < 8 || text.length > 64) return '密码需要 8 到 64 位'
          return true
        }
      })
      password = res.value
    } catch {
      return
    }
  }
  const body = await http.post('/certificate/download', { id: row.id, format, password })
  const data = body.data
  const bin = atob(data.contentBase64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  const blob = new Blob([bytes], { type: data.contentType || 'application/zip' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = data.filename || `${row.name}-${format}.zip`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
  ElMessage.success(`${bundleNames[format] || ''} 证书已下载`)
}

async function deploy(row) {
  const title = row.status === 'deployed' ? '重启 Nginx' : '部署到 Nginx'
  const message = row.status === 'deployed'
    ? '将使用已申请的证书重新写入文件，检测配置后重启 Nginx。'
    : '将把已申请的证书写入 Nginx 目录，检测配置后重启服务。'
  await ElMessageBox.confirm(message, title)
  const body = await http.post('/certificate/deploy', { id: row.id })
  ElMessage.success(body.data.result?.message || '已执行')
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`删除证书「${row.name}」及其调度？磁盘备份会保留。`, '删除证书', { type: 'warning' })
  await http.post('/certificate/delete', { id: row.id })
  load()
}

async function poll(id) {
  for (let i = 0; i < 40; i++) {
    const body = await http.post('/task/detail', { id })
    if (['success', 'failed', 'skipped'].includes(body.data.status)) return body.data
    await new Promise((resolve) => setTimeout(resolve, 1500))
  }
  return { status: 'failed', error: '任务仍在执行，请到任务记录查看' }
}

function onPage(value) {
  page.value = value
  load()
}

onMounted(() => {
  load()
  if (auth.can('datasource:list')) loadSources()
})
</script>
