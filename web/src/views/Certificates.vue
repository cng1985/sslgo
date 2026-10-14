<template>
  <div>
    <div class="page-head">
      <div>
        <h1>证书</h1>
        <p>申请 Let's Encrypt 证书，替换旧文件，并按数据源配置重启 Nginx。</p>
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
        <el-table-column label="到期" width="170">
          <template #default="{ row }">{{ fmt(row.notAfter) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="280">
          <template #default="{ row }">
            <el-button v-if="auth.can('certificate:issue')" link type="primary" @click="issue(row, true)">签发并部署</el-button>
            <el-button v-if="auth.can('certificate:deploy')" link @click="deploy(row)">重启 Nginx</el-button>
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
        <el-form-item label="域名，多个用空格或换行分隔"><el-input v-model="form.domainText" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="ACME 数据源">
          <el-select v-model="form.acmeDataSourceId" style="width: 100%">
            <el-option v-for="item in acmeSources" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Nginx 数据源">
          <el-select v-model="form.nginxDataSourceId" style="width: 100%">
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
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'
import { fmt, statusText, statusType } from '../utils'

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
const statuses = ['draft', 'issuing', 'deployed', 'failed']
const form = reactive({
  name: '', domainText: '', acmeDataSourceId: null, nginxDataSourceId: null,
  keyType: 'ec256', renewBeforeDays: 30, autoRenew: true, agreeTos: false
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
  Object.assign(form, { name: '', domainText: '', acmeDataSourceId: null, nginxDataSourceId: null, keyType: 'ec256', renewBeforeDays: 30, autoRenew: true, agreeTos: false })
  open.value = true
}

async function create() {
  await http.post('/certificate/create', {
    ...form,
    domains: form.domainText.split(/[\s,]+/).filter(Boolean)
  })
  open.value = false
  ElMessage.success('证书申请已创建')
  load()
}

async function issue(row, force) {
  try {
    await ElMessageBox.confirm(`将为 ${row.domains.join('、')} 申请证书，成功后替换旧文件并重启 Nginx。`, '签发并部署')
  } catch {
    return
  }
  const created = await http.post('/certificate/issue', { id: row.id, force })
  ElMessage.info('任务已开始')
  const task = await poll(created.data.id)
  if (task.status === 'failed') {
    ElMessage.error(task.error || '签发失败')
    return
  }
  ElMessage.success(task.result?.message || statusText(task.status))
  load()
}

async function deploy(row) {
  await ElMessageBox.confirm('将使用已落地的证书文件重新检测并重启 Nginx。', '重启 Nginx')
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
