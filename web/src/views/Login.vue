<template>
  <div class="login">
    <div class="glow glow-a" aria-hidden="true"></div>
    <div class="glow glow-b" aria-hidden="true"></div>

    <section class="hero">
      <div class="center">
        <p class="eyebrow">SSL 控制台</p>
        <h1>证书到期前，<br>它先替你办好。</h1>
        <p class="lede">自动申请，写进 Nginx，完成重启。</p>

        <div class="preview">
          <div class="cert">
            <svg class="shield" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M12 2.75 4.5 5.5v5.2c0 4.6 3 8 7.5 10.55 4.5-2.55 7.5-5.95 7.5-10.55V5.5L12 2.75z" />
              <path d="m8.6 12.05 2.35 2.3 4.45-4.7" />
            </svg>
            <div class="cert-text">
              <strong>*.example.com</strong>
              <span class="days">有效，到期 2027-02-04<em>剩余 86 天</em></span>
            </div>
            <span class="ok" aria-label="已就绪">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="m3 8.5 3.2 3 6.8-7" /></svg>
            </span>
          </div>
          <div class="cert dim">
            <svg class="shield" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M12 2.75 4.5 5.5v5.2c0 4.6 3 8 7.5 10.55 4.5-2.55 7.5-5.95 7.5-10.55V5.5L12 2.75z" />
            </svg>
            <div class="cert-text">
              <strong>api.example.com</strong>
              <span class="days">有效，到期 2027-03-19<em>剩余 129 天</em></span>
            </div>
          </div>
          <p class="auto">已配置自动续期</p>
        </div>
      </div>
    </section>

    <section class="entry">
      <div class="card">
        <form @submit.prevent="submit">
          <div class="card-brand">
            <span class="mark">证</span>
            <span>
              <strong>证枢</strong>
              <small>SSL 控制台</small>
            </span>
          </div>
          <h2>欢迎回来</h2>
          <p class="note">使用已分配的账号进入控制台</p>

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
      </div>
      <p class="foot">租户由账号归属决定</p>
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
  grid-template-columns: 46fr 54fr;
  position: relative;
  overflow: auto;
  background: #0c1210;
  color: #e9f2ed;
}

.glow { position: absolute; border-radius: 50%; filter: blur(90px); pointer-events: none; }
.glow-a {
  width: 520px; height: 520px;
  left: -160px; top: -140px;
  background: rgba(20, 184, 132, 0.24);
}
.glow-b {
  width: 460px; height: 460px;
  right: -120px; bottom: -160px;
  background: rgba(20, 184, 132, 0.12);
}

.hero {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  padding: 44px 52px;
  background:
    radial-gradient(560px 340px at 30% 18%, rgba(20, 184, 132, 0.14), transparent 70%),
    linear-gradient(180deg, #101916 0%, #0c1210 100%);
}
.mark {
  width: 44px; height: 44px;
  border-radius: 14px;
  display: grid; place-items: center;
  background: linear-gradient(135deg, #19c48d, #0e6b52);
  color: #ffffff;
  font-size: 20px;
  box-shadow: 0 10px 24px rgba(14, 107, 82, 0.26);
}

.center {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 32px 0;
}
.eyebrow {
  margin: 0 0 16px;
  color: #37d3a0;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.22em;
}
.center h1 {
  margin: 0;
  font-size: 40px;
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1.3;
}
.lede {
  margin: 18px 0 0;
  color: rgba(233, 242, 237, 0.62);
  font-size: 15px;
  letter-spacing: 0.06em;
}

.preview {
  margin-top: 34px;
  width: min(400px, 100%);
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
}
.cert {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) 24px;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.09);
}
.cert.dim { opacity: 0.72; grid-template-columns: 38px minmax(0, 1fr); }
.shield {
  width: 38px; height: 38px;
  padding: 8px;
  border-radius: 12px;
  color: #37d3a0;
  background: rgba(55, 211, 160, 0.12);
  box-sizing: border-box;
}
.cert-text { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.cert-text strong { font-size: 15px; font-weight: 600; letter-spacing: 0.02em; }
.days { color: rgba(233, 242, 237, 0.52); font-size: 12.5px; }
.days em { font-style: normal; margin-left: 8px; color: rgba(55, 211, 160, 0.9); }
.ok {
  width: 24px; height: 24px;
  border-radius: 50%;
  display: grid; place-items: center;
  color: #0c1210;
  background: #37d3a0;
}
.ok svg { width: 14px; height: 14px; }
.auto { margin: 2px 4px 0; color: rgba(233, 242, 237, 0.5); font-size: 12.5px; }

.entry {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 48px 32px;
}
.card {
  width: min(440px, 100%);
  background: #ffffff;
  color: #0c1210;
  border-radius: 28px;
  padding: 40px 40px 32px;
  border: 1px solid #e6ede9;
  box-shadow:
    0 1px 2px rgba(12, 18, 16, 0.06),
    0 16px 40px rgba(0, 0, 0, 0.22);
}
.card-brand { display: flex; align-items: center; gap: 13px; }
.card-brand > span:last-child { display: flex; flex-direction: column; gap: 2px; }
.card-brand strong {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.16em;
}
.card-brand small { color: #8a9a93; font-size: 11px; letter-spacing: 0.16em; }
.card h2 {
  margin: 30px 0 0;
  font-family: "Songti SC", "Noto Serif CJK SC", "Source Han Serif SC", Georgia, serif;
  font-size: 30px;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.note {
  margin: 8px 0 28px;
  color: #6b7b74;
  font-size: 13.5px;
  line-height: 1.6;
}

.field { display: flex; flex-direction: column; gap: 8px; margin: 0 0 6px; }
.field > span { font-size: 13px; font-weight: 600; color: #31413a; }
.field em {
  min-height: 16px;
  font-style: normal;
  font-size: 12px;
  line-height: 16px;
  color: var(--danger);
}
.field :deep(.el-input__wrapper) {
  border-radius: 12px;
  background: #f4f7f5;
  box-shadow: 0 0 0 1px #e2eae6 inset;
  min-height: 50px;
  padding: 2px 14px;
  transition: box-shadow 0.15s ease, background 0.15s ease;
}
.field :deep(.el-input__wrapper:hover) { box-shadow: 0 0 0 1px #cfddd6 inset; }
.field :deep(.el-input__wrapper:focus-within),
.field :deep(.el-input__wrapper.is-focus) {
  background: #ffffff;
  box-shadow: 0 0 0 1px #14b884 inset, 0 0 0 4px rgba(20, 184, 132, 0.14);
}
.field.bad :deep(.el-input__wrapper),
.field.bad :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 1px var(--danger) inset, 0 0 0 4px rgba(141, 58, 50, 0.1);
}
.field :deep(.el-input__inner) { color: #0c1210; font-size: 15px; }
.field :deep(.el-input__suffix) { color: #94a3a0; }

.go {
  width: 100%;
  height: 52px;
  margin-top: 8px;
  border: 0;
  border-radius: 14px;
  letter-spacing: 0.1em;
  font-size: 15px;
  font-weight: 600;
  background: linear-gradient(90deg, #0c1412 0%, #14543f 58%, #0e6b52 100%);
  --el-button-bg-color: transparent;
  --el-button-border-color: transparent;
  --el-button-text-color: #ffffff;
  --el-button-hover-bg-color: transparent;
  --el-button-hover-border-color: transparent;
  --el-button-hover-text-color: #ffffff;
  --el-button-active-bg-color: transparent;
  --el-button-active-border-color: transparent;
  --el-button-active-text-color: #ffffff;
  box-shadow: 0 14px 28px rgba(14, 107, 82, 0.24);
}
.go:hover { filter: brightness(1.06); box-shadow: 0 16px 32px rgba(14, 107, 82, 0.28); }
.aside {
  margin: 18px 0 0;
  color: #93a3a0;
  font-size: 12.5px;
}
.foot {
  margin: 0;
  color: rgba(233, 242, 237, 0.48);
  font-size: 12px;
  letter-spacing: 0.08em;
}

:global(body.is-login) { background: #0c1210; }

@media (max-width: 860px) {
  .login { grid-template-columns: 1fr; }
  .glow-a { left: -220px; top: -220px; }
  .glow-b { display: none; }
  .hero { padding: 24px 20px 0; }
  .center { flex: none; padding: 22px 0 0; }
  .center h1 { font-size: 30px; line-height: 1.35; }
  .lede { margin-top: 12px; font-size: 14px; }
  .preview { display: none; }
  .entry { padding: 24px 16px 40px; align-items: stretch; }
  .card { width: 100%; padding: 30px 22px 28px; border-radius: 24px; }
  .card h2 { margin-top: 26px; font-size: 26px; }
  .foot { text-align: center; }
}
</style>
