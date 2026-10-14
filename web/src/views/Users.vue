<template>
  <div>
    <div class="page-head">
      <div>
        <h1>用户</h1>
        <p>用户属于当前登录租户。角色决定能否签发证书和重启 Nginx。</p>
      </div>
      <el-button v-if="auth.can('user:create')" type="primary" @click="open = true">新建用户</el-button>
    </div>
    <div class="panel">
      <el-table :data="rows">
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="displayName" label="显示名" />
        <el-table-column label="角色">
          <template #default="{ row }">{{ row.roles.map((item) => item.name).join('、') || '—' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button v-if="auth.can('user:assignRole')" link @click="editRoles(row)">分配角色</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog v-model="open" title="新建用户" width="460px">
      <el-form label-position="top">
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="显示名"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.roleIds" multiple style="width: 100%">
            <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="open = false">取消</el-button>
        <el-button type="primary" @click="create">创建</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="roleOpen" title="分配角色" width="420px">
      <el-select v-model="roleIds" multiple style="width: 100%">
        <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.id" />
      </el-select>
      <template #footer>
        <el-button @click="roleOpen = false">取消</el-button>
        <el-button type="primary" @click="saveRoles">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'
import { statusText, statusType } from '../utils'

const auth = useAuthStore()
const rows = ref([])
const roles = ref([])
const open = ref(false)
const roleOpen = ref(false)
const roleIds = ref([])
const currentUser = ref(0)
const form = reactive({ username: '', displayName: '', password: '', roleIds: [] })

async function load() {
  const body = await http.post('/user/list', { page: 1, size: 100, filters: {} })
  rows.value = body.data
  if (auth.can('role:list')) {
    const roleBody = await http.post('/role/list', { page: 1, size: 100 })
    roles.value = roleBody.data
  }
}

async function create() {
  await http.post('/user/create', form)
  open.value = false
  ElMessage.success('用户已创建')
  load()
}

function editRoles(row) {
  currentUser.value = row.id
  roleIds.value = row.roles.map((item) => item.id)
  roleOpen.value = true
}

async function saveRoles() {
  await http.post('/user/assignRole', { userId: currentUser.value, roleIds: roleIds.value })
  roleOpen.value = false
  load()
}

onMounted(load)
</script>
