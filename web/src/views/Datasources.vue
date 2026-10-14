<template>
  <div>
    <div class="page-head">
      <div>
        <h1>数据源</h1>
        <p>ACME 账户负责申请证书，Nginx 目标负责存放文件并重启服务。</p>
      </div>
      <el-button v-if="auth.can('datasource:create')" type="primary" @click="openCreate">新建数据源</el-button>
    </div>
    <div class="panel">
      <el-table :data="rows" v-loading="loading">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="lastTestMsg" label="最近测试" min-width="220" />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button v-if="auth.can('datasource:test')" link type="primary" @click="test(row)">测试</el-button>
            <el-button v-if="auth.can('datasource:delete')" link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="open" title="新建数据源" width="620px">
      <el-form label-position="top">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio value="acme">ACME</el-radio>
            <el-radio value="nginx">Nginx</el-radio>
          </el-radio-group>
        </el-form-item>
        <template v-if="form.type === 'acme'">
          <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
          <el-form-item label="目录">
            <el-select v-model="form.directoryURL" style="width: 100%">
              <el-option label="Let's Encrypt 测试环境" value="https://acme-staging-v02.api.letsencrypt.org/directory" />
              <el-option label="Let's Encrypt 生产环境" value="https://acme-v02.api.letsencrypt.org/directory" />
            </el-select>
          </el-form-item>
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
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'
import { statusText, statusType } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const loading = ref(false)
const open = ref(false)
const form = reactive(blank())

function blank() {
  return {
    name: '', type: 'nginx', email: '',
    directoryURL: 'https://acme-staging-v02.api.letsencrypt.org/directory',
    nginxBin: '/usr/sbin/nginx', serviceManager: 'systemd', serviceName: 'nginx', action: 'restart',
    webroot: '/var/www/acme', certDir: '/etc/nginx/ssl', snippetDir: '/etc/nginx/conf.d',
    testBefore: true, dryRun: false
  }
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

function openCreate() {
  Object.assign(form, blank())
  open.value = true
}

async function create() {
  const config = form.type === 'acme'
    ? { email: form.email, directoryURL: form.directoryURL }
    : {
        nginxBin: form.nginxBin, serviceManager: form.serviceManager, serviceName: form.serviceName,
        action: form.action, webroot: form.webroot, certDir: form.certDir, snippetDir: form.snippetDir,
        testBefore: form.testBefore, dryRun: form.dryRun
      }
  await http.post('/datasource/create', { name: form.name, type: form.type, config })
  open.value = false
  ElMessage.success('数据源已创建')
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

onMounted(load)
</script>
