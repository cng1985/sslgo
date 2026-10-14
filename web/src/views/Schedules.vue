<template>
  <div>
    <div class="page-head">
      <div>
        <h1>调度</h1>
        <p>到期窗口外的自动执行会跳过，不会重复访问 Let's Encrypt。</p>
      </div>
      <el-button v-if="auth.can('schedule:create')" type="primary" @click="open = true">新建调度</el-button>
    </div>
    <div class="panel">
      <el-table :data="rows">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="etlJobId" label="流水线" width="100" />
        <el-table-column prop="cron" label="Cron" width="150" />
        <el-table-column label="下次" width="170">
          <template #default="{ row }">{{ fmt(row.nextRunAt) }}</template>
        </el-table-column>
        <el-table-column label="启用" width="90">
          <template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button v-if="auth.can('schedule:enable') && !row.enabled" link @click="toggle(row, true)">启用</el-button>
            <el-button v-if="auth.can('schedule:disable') && row.enabled" link @click="toggle(row, false)">停用</el-button>
            <el-button v-if="auth.can('schedule:delete')" link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="open" title="新建调度" width="480px">
      <el-form label-position="top">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="流水线 ID"><el-input-number v-model="form.etlJobId" :min="1" /></el-form-item>
        <el-form-item label="Cron">
          <el-input v-model="form.cron" placeholder="15 3 * * *" />
        </el-form-item>
        <el-checkbox v-model="form.enabled">创建后启用</el-checkbox>
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
import { fmt } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const open = ref(false)
const form = reactive({ name: '每日续期检查', etlJobId: 1, cron: '15 3 * * *', enabled: true })

async function load() {
  const body = await http.post('/schedule/list', { page: 1, size: 100, filters: {} })
  rows.value = body.data
}

async function create() {
  await http.post('/schedule/create', form)
  open.value = false
  ElMessage.success('调度已创建')
  load()
}

async function toggle(row, enabled) {
  await http.post(enabled ? '/schedule/enable' : '/schedule/disable', { id: row.id })
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`删除调度「${row.name}」？`, '删除', { type: 'warning' })
  await http.post('/schedule/delete', { id: row.id })
  load()
}

onMounted(load)
</script>
