<template>
  <div class="shell">
    <aside class="rail">
      <div class="logo">
        <span class="seal">证</span>
        <div>
          <strong class="brand-mark">证枢</strong>
          <small>SSL 控制台</small>
        </div>
      </div>
      <el-menu :default-active="route.path" router background-color="transparent" text-color="#d7e4dc" active-text-color="#f3d7a1">
        <el-menu-item v-for="item in visible" :key="item.path" :index="item.path">{{ item.label }}</el-menu-item>
      </el-menu>
    </aside>
    <section class="main">
      <header class="top">
        <div>
          <span class="tenant">{{ auth.profile?.tenant?.name }}</span>
        </div>
        <div class="who">
          <el-button v-if="auth.profile?.user?.mustChangePassword" link type="warning" @click="openPassword">请修改初始密码</el-button>
          <span>{{ auth.name }}</span>
          <el-button link @click="openPassword">修改密码</el-button>
          <el-button link @click="logout">退出</el-button>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </section>
    <el-dialog v-model="passwordOpen" title="修改密码" width="420px">
      <el-form label-position="top">
        <el-form-item label="当前密码"><el-input v-model="password.oldPassword" type="password" show-password /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="password.newPassword" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordOpen = false">取消</el-button>
        <el-button type="primary" @click="changePassword">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menus = [
  { path: '/', label: '总览', perm: 'dashboard:summary' },
  { path: '/certificates', label: '证书', perm: 'certificate:list' },
  { path: '/datasources', label: '数据源', perm: 'datasource:list' },
  { path: '/pipelines', label: '流水线', perm: 'etlJob:list' },
  { path: '/tasks', label: '任务', perm: 'task:list' },
  { path: '/schedules', label: '调度', perm: 'schedule:list' },
  { path: '/users', label: '用户', perm: 'user:list' },
  { path: '/roles', label: '角色', perm: 'role:list' }
]
const visible = computed(() => menus.filter((item) => auth.can(item.perm)))
const passwordOpen = ref(false)
const password = reactive({ oldPassword: '', newPassword: '' })

function openPassword() {
  password.oldPassword = ''
  password.newPassword = ''
  passwordOpen.value = true
}

async function changePassword() {
  await http.post('/auth/changePassword', password)
  passwordOpen.value = false
  if (auth.profile?.user) auth.profile.user.mustChangePassword = false
  ElMessage.success('密码已更新')
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.shell {
  height: 100vh;
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr);
  overflow: hidden;
}
.rail {
  background: var(--ink);
  color: #e7f0ea;
  padding: 22px 14px;
  min-height: 0;
  overflow-y: auto;
}
.logo { display: flex; gap: 12px; align-items: center; padding: 6px 10px 22px; }
.logo strong { display: block; font-size: 22px; }
.logo small { color: #b7c7be; }
.seal {
  width: 42px; height: 42px; border-radius: 50%;
  display: grid; place-items: center;
  border: 1px solid #d7b56a; color: #f3d7a1;
  font-family: Palatino, "Songti SC", serif;
}
.main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.top {
  flex: 0 0 64px;
  display: flex; justify-content: space-between; align-items: center;
  padding: 0 28px; border-bottom: 1px solid var(--line); background: rgba(255,253,248,.92);
}
.tenant { color: #66756d; }
.who { display: flex; gap: 12px; align-items: center; }
.content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 26px 28px 40px;
}
:deep(.el-menu) { border-right: 0; }
:deep(.el-menu-item) { border-radius: 10px; margin-bottom: 4px; }
:deep(.el-menu-item.is-active) { background: rgba(243, 215, 161, .12); }
@media (max-width: 800px) {
  .shell { height: auto; min-height: 100vh; grid-template-columns: 1fr; overflow: visible; }
  .rail { padding-bottom: 8px; overflow: visible; }
  .main { display: block; overflow: visible; }
  .content { overflow: visible; }
}
</style>
