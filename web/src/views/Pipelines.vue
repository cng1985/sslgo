<template>
  <div>
    <div class="page-head">
      <div>
        <h1>签发流水线</h1>
        <p>Extract 读取配置，Transform 向 ACME 申请证书。选择部署时，Load 才会替换文件并重启 Nginx。</p>
      </div>
    </div>
    <div class="panel">
      <el-table :data="rows" v-loading="loading">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="certificateId" label="证书" width="100" />
        <el-table-column prop="pipeline" label="管道" width="140" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button v-if="auth.can('etlJob:execute')" link type="primary" @click="execute(row, false)">只申请</el-button>
            <el-button v-if="auth.can('etlJob:execute')" link @click="execute(row, true)">申请并部署</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'
import { statusText, statusType } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const body = await http.post('/etlJob/list', { page: 1, size: 100, filters: {} })
    rows.value = body.data
  } finally {
    loading.value = false
  }
}

async function execute(row, deploy) {
  const created = await http.post('/etlJob/execute', { id: row.id, force: true, deploy })
  ElMessage.success(deploy ? `已提交部署任务 #${created.data.id}` : `已提交申请任务 #${created.data.id}`)
}

onMounted(load)
</script>
