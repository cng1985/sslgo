<template>
  <div class="login">
    <section class="story">
      <p class="eyebrow">LET'S ENCRYPT · NGINX</p>
      <h1 class="brand-mark">证枢</h1>
      <p>为域名申请免费证书，替换服务器上的旧文件，然后重启 Nginx。</p>
      <ol>
        <li>登记 ACME 账户与 Nginx 目录</li>
        <li>用 HTTP-01 完成域名验证</li>
        <li>写入新证书并重启服务</li>
      </ol>
    </section>
    <section class="card">
      <h2>登录控制台</h2>
      <p>租户由账号归属决定，不需要填写租户编号。</p>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-button type="primary" class="go" :loading="loading" @click="submit">进入</el-button>
      </el-form>
    </section>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function submit() {
  loading.value = true
  try {
    await auth.login(form)
    router.push('/')
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login { min-height: 100vh; display: grid; grid-template-columns: 1.1fr .9fr; }
.story {
  background: var(--ink); color: #f6f1e7; padding: 10vh 8vw;
}
.eyebrow { letter-spacing: .22em; color: #d7b56a; font-size: 12px; }
.story h1 { font-size: 76px; margin: 12px 0; }
.story p { max-width: 460px; line-height: 1.7; color: #d5e2da; }
.story ol { padding-left: 18px; line-height: 2; color: #f3d7a1; }
.card { display: flex; flex-direction: column; justify-content: center; padding: 48px; }
.card h2 { font-family: Palatino, "Songti SC", serif; font-size: 32px; margin-bottom: 6px; }
.card p { color: #66756d; }
.go { width: 100%; margin-top: 8px; }
@media (max-width: 860px) {
  .login { grid-template-columns: 1fr; }
  .story { padding: 36px 24px; }
  .story h1 { font-size: 52px; }
}
</style>
