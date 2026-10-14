<template>
  <div>
    <div class="page-head">
      <div>
        <h1>角色</h1>
        <p>管理员拥有全部权限。运维可以签发和部署，只读账号只能查看。</p>
      </div>
    </div>
    <div class="layout">
      <div class="panel">
        <el-table :data="rows" highlight-current-row @row-click="select">
          <el-table-column prop="name" label="角色" />
          <el-table-column prop="code" label="编码" />
          <el-table-column label="权限数" width="90">
            <template #default="{ row }">{{ row.permissions?.length || 0 }}</template>
          </el-table-column>
        </el-table>
      </div>
      <div class="panel" v-if="current">
        <h3>{{ current.name }}</h3>
        <p>{{ current.description || '自定义授权' }}</p>
        <div v-for="(items, resource) in grouped" :key="resource" class="block">
          <strong>{{ resource }}</strong>
          <el-checkbox-group v-model="checked">
            <el-checkbox v-for="item in items" :key="item.id" :value="item.id" :disabled="current.code === 'admin' || !auth.can('role:grant')">{{ item.name }}</el-checkbox>
          </el-checkbox-group>
        </div>
        <el-button v-if="auth.can('role:grant') && current.code !== 'admin'" type="primary" @click="grant">保存授权</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const rows = ref([])
const permissions = ref([])
const current = ref(null)
const checked = ref([])
const grouped = computed(() => {
  const groups = {}
  for (const item of permissions.value) {
    groups[item.resource] = groups[item.resource] || []
    groups[item.resource].push(item)
  }
  return groups
})

async function load() {
  const body = await http.post('/role/list', { page: 1, size: 100 })
  rows.value = body.data
  if (auth.can('role:permissionList')) {
    const catalog = await http.post('/role/permissionList', {})
    permissions.value = catalog.data
  }
}

function select(row) {
  current.value = row
  checked.value = (row.permissions || []).map((item) => item.id)
}

async function grant() {
  const body = await http.post('/role/grant', { roleId: current.value.id, permissionIds: checked.value })
  current.value = body.data
  ElMessage.success('授权已更新')
  load()
}

onMounted(load)
</script>

<style scoped>
.layout { display: grid; grid-template-columns: 360px 1fr; gap: 16px; }
.block { margin: 14px 0; }
.block strong { display: block; margin-bottom: 6px; }
@media (max-width: 900px) { .layout { grid-template-columns: 1fr; } }
</style>
