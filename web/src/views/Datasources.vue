<template>
  <div>
    <div class="page-head">
      <div>
        <h1>数据源</h1>
        <p>ACME 账户负责向证书机构注册，并可绑定云厂商 DNS。Nginx 目标负责存放证书并重启服务。</p>
      </div>
      <el-button v-if="auth.can('datasource:create')" type="primary" @click="openCreate">新建数据源</el-button>
    </div>
    <div class="panel">
      <el-table :data="rows" v-loading="loading">
        <el-table-column prop="name" label="名称" />
        <el-table-column label="类型" min-width="160">
          <template #default="{ row }">{{ typeText(row) }}</template>
        </el-table-column>
        <el-table-column label="账户" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.type === 'acme'" :type="row.config?.accountRegistered ? 'success' : 'info'">
              {{ row.config?.accountRegistered ? '已注册' : '未注册' }}
            </el-tag>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="lastTestMsg" label="最近测试" min-width="220" />
        <el-table-column label="操作" width="240">
          <template #default="{ row }">
            <el-button v-if="row.type === 'acme' && !row.config?.accountRegistered && auth.can('datasource:register')" link type="primary" @click="register(row)">注册</el-button>
            <el-button v-if="auth.can('datasource:test')" link type="primary" @click="test(row)">测试</el-button>
            <el-button v-if="auth.can('datasource:delete')" link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="open" title="新建数据源" width="680px">
      <el-form label-position="top">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio value="acme">ACME 账户</el-radio>
            <el-radio value="nginx">Nginx</el-radio>
          </el-radio-group>
        </el-form-item>
        <template v-if="form.type === 'acme'">
          <el-form-item label="邮箱"><el-input v-model="form.email" placeholder="用于 ACME 账户联系" /></el-form-item>
          <el-form-item label="证书机构">
            <el-select v-model="form.ca" style="width: 100%">
              <el-option v-for="item in catalog.cas" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <p v-if="currentCA?.note" class="hint">{{ currentCA.note }}</p>
          <el-form-item v-if="form.ca === 'custom'" label="ACME 目录">
            <el-input v-model="form.directoryURL" placeholder="https://example.com/acme/directory" />
          </el-form-item>
          <template v-if="currentCA?.eab || form.ca === 'custom'">
            <el-form-item label="EAB Kid"><el-input v-model="form.eabKid" /></el-form-item>
            <el-form-item label="EAB HMAC Key"><el-input v-model="form.eabHmacKey" type="password" show-password /></el-form-item>
          </template>
          <el-form-item label="云厂商 DNS">
            <el-select v-model="form.dnsProvider" clearable placeholder="不绑定，仅使用 HTTP-01" style="width: 100%" @change="onProvider">
              <el-option v-for="item in catalog.dnsProviders" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <p v-if="currentDNS?.note" class="hint">{{ currentDNS.note }}</p>
          <el-form-item v-for="field in currentDNS?.fields || []" :key="field.name" :label="field.label">
            <el-input
              v-if="field.multiline"
              v-model="form.dns[field.name]"
              type="textarea"
              :rows="5"
              :placeholder="field.placeholder"
            />
            <el-input
              v-else
              v-model="form.dns[field.name]"
              :type="field.secret ? 'password' : 'text'"
              :show-password="field.secret"
              :placeholder="field.placeholder"
            />
            <p v-if="field.help" class="hint">{{ field.help }}</p>
          </el-form-item>
          <el-checkbox v-model="form.termsAgreed">同意 ACME 服务条款</el-checkbox>
          <el-checkbox v-model="form.registerNow">创建后立即注册账户</el-checkbox>
        </template>
        <template v-else>
          <el-form-item label="nginx 程序"><el-input v-model="form.nginxBin" /></el-form-item>
          <el-form-item label="服务管理">
            <el-radio-group v-model="form.serviceManager">
              <el-radio value="signal">nginx 信号</el-radio>
              <el-radio value="systemd">systemd</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="动作">
            <el-radio-group v-model="form.action">
              <el-radio value="restart">重启服务</el-radio>
              <el-radio value="reload">重载配置</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.serviceManager === 'systemd'" label="服务名"><el-input v-model="form.serviceName" /></el-form-item>
          <el-form-item label="HTTP-01 网站根目录"><el-input v-model="form.webroot" /></el-form-item>
          <el-form-item label="证书目录"><el-input v-model="form.certDir" /></el-form-item>
          <el-form-item label="可选的 Nginx 片段目录"><el-input v-model="form.snippetDir" placeholder="/etc/nginx/conf.d" /></el-form-item>
          <el-checkbox v-model="form.testBefore">重启前执行 nginx -t</el-checkbox>
          <el-checkbox v-model="form.dryRun">演练模式，只写文件不重启</el-checkbox>
        </template>
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
import { statusText, statusType } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const loading = ref(false)
const open = ref(false)
const catalog = reactive({ cas: [], dnsProviders: [] })
const form = reactive(blank())
const currentCA = computed(() => catalog.cas.find((item) => item.id === form.ca))
const currentDNS = computed(() => catalog.dnsProviders.find((item) => item.id === form.dnsProvider))

function blank() {
  return {
    name: '', type: 'acme', email: '', ca: 'letsencrypt-staging', directoryURL: '',
    eabKid: '', eabHmacKey: '', termsAgreed: false, registerNow: false,
    dnsProvider: '', dns: {},
    nginxBin: '/usr/sbin/nginx', serviceManager: 'systemd', serviceName: 'nginx', action: 'restart',
    webroot: '/var/www/acme', certDir: '/etc/nginx/ssl', snippetDir: '/etc/nginx/conf.d',
    testBefore: true, dryRun: false
  }
}

function typeText(row) {
  if (row.type !== 'acme') return 'Nginx'
  const ca = catalog.cas.find((item) => item.id === row.config?.ca)
  const dns = catalog.dnsProviders.find((item) => item.id === row.config?.dnsProvider)
  const caName = ca?.name || 'ACME'
  return dns ? `${caName} · ${dns.name}` : caName
}

async function load() {
  loading.value = true
  try {
    const body = await http.post('/datasource/list', { page: 1, size: 100, filters: {} })
    rows.value = body.data
  } finally {
    loading.value = false
  }
}

async function loadCatalog() {
  if (!auth.can('datasource:catalog')) return
  const body = await http.post('/datasource/catalog', {})
  catalog.cas = body.data.cas || []
  catalog.dnsProviders = body.data.dnsProviders || []
}

function openCreate() {
  Object.assign(form, blank())
  form.dns = {}
  open.value = true
}

function onProvider() {
  form.dns = {}
}

watch(() => form.registerNow, (value) => {
  if (value) form.termsAgreed = true
})

async function create() {
  const config = form.type === 'acme'
    ? {
        email: form.email,
        ca: form.ca,
        directoryURL: form.ca === 'custom' ? form.directoryURL : '',
        eabKid: form.eabKid,
        eabHmacKey: form.eabHmacKey,
        termsAgreed: form.termsAgreed || form.registerNow,
        dnsProvider: form.dnsProvider || '',
        dns: { ...form.dns },
        register: form.registerNow
      }
    : {
        nginxBin: form.nginxBin, serviceManager: form.serviceManager, serviceName: form.serviceName,
        action: form.action, webroot: form.webroot, certDir: form.certDir, snippetDir: form.snippetDir,
        testBefore: form.testBefore, dryRun: form.dryRun
      }
  const registering = form.type === 'acme' && form.registerNow
  await http.post('/datasource/create', { name: form.name, type: form.type, config }, {
    timeout: registering ? 60000 : 30000
  })
  open.value = false
  ElMessage.success(registering ? 'ACME 账户已创建并注册' : '数据源已创建')
  load()
}

async function register(row) {
  await ElMessageBox.confirm(`使用「${row.name}」的邮箱向证书机构注册 ACME 账户？`, '注册账户')
  await http.post('/datasource/register', { id: row.id, agreeTos: true }, { timeout: 60000 })
  ElMessage.success('ACME 账户已注册')
  load()
}

async function test(row) {
  const body = await http.post('/datasource/test', { id: row.id })
  ElMessage[body.data.ok ? 'success' : 'warning'](body.data.message)
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`删除数据源「${row.name}」？`, '删除', { type: 'warning' })
  await http.post('/datasource/delete', { id: row.id })
  load()
}

onMounted(() => {
  load()
  loadCatalog()
})
</script>
