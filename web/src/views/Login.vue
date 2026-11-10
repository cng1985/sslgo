<template>
  <div class="login">
    <section class="hero">
      <div class="rings" aria-hidden="true">
        <svg class="shield" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2.75 4.5 5.5v5.2c0 4.6 3 8 7.5 10.55 4.5-2.55 7.5-5.95 7.5-10.55V5.5L12 2.75z" />
          <path d="m8.6 12.05 2.35 2.3 4.45-4.7" />
        </svg>
      </div>

      <header class="brand">证枢</header>

      <div class="copy">
        <h1>证书到期前，<br>它先替你办好。</h1>
        <p class="lede">向证书机构免费申请，自动续期，写进 Nginx 后再重启。</p>
        <ul class="traits">
          <li>
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m3 8.5 3.2 3 6.8-7" /></svg>
            免费证书，支持通配符
          </li>
          <li>
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m3 8.5 3.2 3 6.8-7" /></svg>
            到期自动续期
          </li>
          <li>
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m3 8.5 3.2 3 6.8-7" /></svg>
            检测通过再重启 Nginx
          </li>
        </ul>
      </div>
    </section>

    <section class="entry">
      <form class="card" @submit.prevent="submit">
        <p class="kicker">登录</p>
        <h2>进入控制台</h2>
        <p class="note">使用已分配的账号。无需填写租户编号。</p>

        <label class="field" :class="{ bad: showUserError }" for="login-username">
          <span>用户名</span>
          <el-input
            id="login-username"
            ref="usernameInput"
            v-model="form.username"
            name="username"
            autocomplete="username"
            size="large"
          />
          <em>{{ showUserError ? '请输入用户名' : '' }}</em>
        </label>

        <label class="field" :class="{ bad: showPasswordError }" for="login-password">
          <span>密码</span>
          <el-input
            id="login-password"
            v-model="form.password"
            name="password"
            type="password"
            show-password
            autocomplete="current-password"
            size="large"
          />
          <em>{{ showPasswordError ? '请输入密码' : '' }}</em>
        </label>

        <el-button native-type="submit" type="primary" class="go" :loading="loading">进入控制台</el-button>
        <p class="aside">登录后可在右上角修改密码</p>
      </form>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const submitted = ref(false)
const usernameInput = ref(null)
const form = reactive({ username: '', password: '' })

const showUserError = computed(() => submitted.value && !form.username.trim())
const showPasswordError = computed(() => submitted.value && !form.password)

onMounted(() => {
  document.body.classList.add('is-login')
  usernameInput.value?.focus?.()
})

onBeforeUnmount(() => {
  document.body.classList.remove('is-login')
})

async function submit() {
  if (loading.value) return
  submitted.value = true
  if (!form.username.trim() || !form.password) return
  loading.value = true
  try {
    await auth.login({ username: form.username.trim(), password: form.password })
    router.push('/')
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  min-height: 100vh;
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(0, 0.92fr);
  overflow: auto;
  background: #ffffff;
  color: #0b1310;
}

.hero {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 48px 52px;
  background: #0a1613;
  color: #e7f4ee;
  overflow: hidden;
}
.rings {
  position: absolute;
  right: -140px;
  bottom: -180px;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  border: 1px solid rgba(231, 244, 238, 0.12);
  display: grid;
  place-items: center;
  color: #37d3a0;
  opacity: 0.85;
}
.rings::before {
  content: "";
  width: 360px;
  height: 360px;
  border-radius: 50%;
  border: 1px solid rgba(231, 244, 238, 0.14);
}
.shield {
  position: absolute;
  width: 96px;
  height: 96px;
}

.brand {
  position: relative;
  z-index: 1;
  font-size: 19px;
  font-weight: 600;
  letter-spacing: 0.42em;
  text-indent: 0.42em;
}

.copy {
  position: relative;
  z-index: 1;
  margin-top: 12vh;
  max-width: 460px;
}
.copy h1 {
  margin: 0;
  font-size: 44px;
  font-weight: 600;
  letter-spacing: 0.01em;
  line-height: 1.25;
}
.lede {
  margin: 22px 0 0;
  max-width: 26em;
  color: rgba(231, 244, 238, 0.66);
  font-size: 15px;
  line-height: 1.8;
}
.traits {
  list-style: none;
  margin: 44px 0 0;
  padding: 0;
  display: grid;
  gap: 14px;
}
.traits li {
  display: flex;
  align-items: center;
  gap: 10px;
  color: rgba(231, 244, 238, 0.74);
  font-size: 14px;
  letter-spacing: 0.03em;
}
.traits svg {
  width: 18px;
  height: 18px;
  flex: none;
  color: #37d3a0;
}

.entry {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 32px;
}
.card { width: min(380px, 100%); }
.kicker {
  margin: 0 0 12px;
  color: #0e6b52;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.22em;
}
.card h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: 0.01em;
}
.note {
  margin: 10px 0 30px;
  color: #5b6d66;
  font-size: 13.5px;
  line-height: 1.6;
}

.field { display: flex; flex-direction: column; gap: 8px; margin: 0 0 6px; }
.field > span { font-size: 13px; font-weight: 600; color: #2c3c35; }
.field em {
  min-height: 16px;
  font-style: normal;
  font-size: 12px;
  line-height: 16px;
  color: var(--danger);
}
.field :deep(.el-input__wrapper) {
  border-radius: 10px;
  background: #f4f7f5;
  box-shadow: none;
  min-height: 50px;
  padding: 2px 14px;
  transition: box-shadow 0.15s ease;
}
.field :deep(.el-input__wrapper:hover) { background: #eef3f1; }
.field :deep(.el-input__wrapper:focus-within),
.field :deep(.el-input__wrapper.is-focus) {
  background: #ffffff;
  box-shadow: 0 0 0 2px #0e6b52;
}
.field.bad :deep(.el-input__wrapper),
.field.bad :deep(.el-input__wrapper:focus-within) {
  background: #ffffff;
  box-shadow: 0 0 0 2px var(--danger);
}
.field :deep(.el-input__inner) { color: #0b1310; font-size: 15px; }
.field :deep(.el-input__suffix) { color: #8d9c95; }

.go {
  width: 100%;
  height: 52px;
  margin-top: 8px;
  border: 0;
  border-radius: 10px;
  letter-spacing: 0.12em;
  font-size: 15px;
  font-weight: 600;
  background: #0a1613;
  --el-button-bg-color: #0a1613;
  --el-button-border-color: #0a1613;
  --el-button-text-color: #ffffff;
  --el-button-hover-bg-color: #1c322b;
  --el-button-hover-border-color: #1c322b;
  --el-button-hover-text-color: #ffffff;
  --el-button-active-bg-color: #0a1613;
  --el-button-active-border-color: #0a1613;
  --el-button-active-text-color: #ffffff;
}
.aside {
  margin: 18px 0 0;
  color: #8d9c95;
  font-size: 12.5px;
}

:global(body.is-login) { background: #ffffff; }

@media (max-width: 860px) {
  .login { grid-template-columns: 1fr; }
  .hero { padding: 24px 20px 30px; }
  .rings { width: 300px; height: 300px; right: -130px; bottom: -120px; opacity: 0.5; }
  .rings::before { width: 200px; height: 200px; }
  .shield { width: 56px; height: 56px; }
  .copy { margin-top: 26px; }
  .copy h1 { font-size: 28px; line-height: 1.35; }
  .lede { margin-top: 12px; font-size: 14px; }
  .traits { display: none; }
  .entry { padding: 28px 20px 44px; }
}
</style>
