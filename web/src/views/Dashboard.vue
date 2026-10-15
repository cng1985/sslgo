<template>
  <div>
    <div class="page-head">
      <div>
        <h1>总览</h1>
        <p>证书到期、部署结果和正在执行的签发任务。</p>
      </div>
    </div>
    <div class="stat-grid">
      <div class="stat"><span>证书</span><strong>{{ summary.certificateTotal }}</strong></div>
      <div class="stat"><span>已部署</span><strong>{{ summary.deployed }}</strong></div>
      <div class="stat"><span>30 天内到期</span><strong>{{ summary.expiringSoon }}</strong><em>含已过期</em></div>
      <div class="stat"><span>失败 / 执行中</span><strong>{{ summary.failed }} / {{ summary.runningTasks }}</strong></div>
    </div>
    <div class="panel">
      <h3>即将到期</h3>
      <el-table :data="summary.expiring" empty-text="没有即将到期的证书">
        <el-table-column prop="name" label="名称" />
        <el-table-column label="域名">
          <template #default="{ row }"><span class="mono">{{ (row.domains || []).join(', ') }}</span></template>
        </el-table-column>
        <el-table-column label="到期" width="220">
          <template #default="{ row }">
            <div>{{ fmt(row.notAfter) }}</div>
            <div v-if="row.notAfter" class="expire warn">{{ remainText(row.daysRemaining, row.notAfter) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive } from 'vue'
import http from '../api/http'
import { fmt, remainText, statusText, statusType } from '../utils'

const summary = reactive({ certificateTotal: 0, deployed: 0, failed: 0, expiringSoon: 0, runningTasks: 0, expiring: [] })

onMounted(async () => {
  const body = await http.post('/dashboard/summary', {})
  Object.assign(summary, body.data)
})
</script>
