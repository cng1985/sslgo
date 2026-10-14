<template>
  <div>
    <div class="page-head">
      <div>
        <h1>任务</h1>
        <p>每次签发、跳过续期或重启 Nginx 都会留下记录。</p>
      </div>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="rows" v-loading="loading" @row-click="show">
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="certificateId" label="证书" width="90" />
        <el-table-column prop="trigger" label="来源" width="110" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="stage" label="阶段" width="120" />
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ fmt(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column prop="error" label="错误" min-width="180" />
      </el-table>
      <div class="pager">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="onPage" />
      </div>
    </div>
    <el-drawer v-model="drawer" title="任务结果" size="460px">
      <pre class="mono">{{ detail }}</pre>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import http from '../api/http'
import { fmt, statusText, statusType } from '../utils'

const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const drawer = ref(false)
const detail = ref('')

async function load() {
  loading.value = true
  try {
    const body = await http.post('/task/list', { page: page.value, size: size.value, filters: {} })
    rows.value = body.data
    total.value = body.total
  } finally {
    loading.value = false
  }
}

function show(row) {
  detail.value = JSON.stringify(row.result || {}, null, 2)
  drawer.value = true
}

function onPage(value) {
  page.value = value
  load()
}

onMounted(load)
</script>

<style scoped>
pre { white-space: pre-wrap; background: #17261f; color: #f6f1e7; padding: 16px; border-radius: 12px; }
</style>
