<template>
  <div class="login">
    <section class="face">
      <div class="lockup">
        <div class="chop" aria-hidden="true">
          <span class="chop-char">证</span>
        </div>
        <strong class="name">证枢</strong>
        <small>SSL 控制台</small>
        <p class="inscription">
          <span>申请证书</span>
          <span>替换旧文件</span>
          <span>再重启 <i>Nginx</i></span>
        </p>
      </div>
      <p class="quiet">私钥留在本机</p>
    </section>

    <section class="sheet">
      <p class="running">证枢 · SSL</p>
      <form @submit.prevent="submit">
        <h1>登录</h1>
        <span class="rule"></span>
        <p class="note">使用已分配的账号进入。无需填写租户编号。</p>

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
@font-face {
  font-family: "Noto Serif SC";
  src: url("../assets/fonts/noto-serif-sc-400.woff2") format("woff2");
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}

.login {
  min-height: 100vh;
  min-height: 100dvh;
  height: 100dvh;
  display: grid;
  grid-template-columns: 1fr 1fr;
  overflow: auto;
  background: var(--ink);
  color: var(--ink);
}

.face {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 56px 36px;
  color: #f6f1e7;
  text-align: center;
  background:
    radial-gradient(460px 460px at 50% 44%, rgba(215, 181, 106, 0.08), transparent 70%),
    var(--ink);
  border-right: 1px solid rgba(215, 181, 106, 0.28);
}
.lockup {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.chop {
  width: clamp(188px, 34vh, 272px);
  aspect-ratio: 1;
  border-radius: 50%;
  border: 1px solid rgba(243, 215, 161, 0.8);
  display: grid;
  place-items: center;
  position: relative;
}
.chop::before {
  content: "";
  position: absolute;
  inset: 11px;
  border-radius: 50%;
  border: 1px solid rgba(243, 215, 161, 0.24);
}
.chop-char {
  position: relative;
  font-family: "Songti SC", "Noto Serif SC", Palatino, serif;
  font-weight: 400;
  font-synthesis: none;
  font-size: clamp(96px, 16.5vh, 136px);
  line-height: 1;
  color: #f6f1e7;
  transform: translateY(-0.05em);
}
.name {
  margin-top: 26px;
  font-family: "Songti SC", "Noto Serif SC", Palatino, "Iowan Old Style", serif;
  font-weight: 400;
  font-synthesis: none;
  font-size: 28px;
  letter-spacing: 0.42em;
  text-indent: 0.42em;
}
.face small {
  margin-top: 8px;
  color: #b7c7be;
  letter-spacing: 0.28em;
  font-size: 12px;
}
.inscription {
  margin: 28px 0 0;
  display: grid;
  gap: 7px;
  color: #d7e3db;
  font-size: 14px;
  letter-spacing: 0.16em;
}
.inscription i { font-style: normal; letter-spacing: 0; }
.quiet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 36px;
  margin: 0;
  color: #9fb0a6;
  font-size: 12px;
  letter-spacing: 0.28em;
}

.sheet {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 56px 48px;
  background: var(--paper);
}
.running {
  position: absolute;
  top: 32px;
  right: 36px;
  margin: 0;
  color: var(--brass);
  font-size: 12px;
  letter-spacing: 0.24em;
}
.sheet form { width: min(400px, 100%); }
.sheet h1 {
  margin: 0;
  font-family: "Songti SC", "Noto Serif SC", Palatino, "Iowan Old Style", serif;
  font-weight: 400;
  font-synthesis: none;
  font-size: 64px;
  line-height: 1;
  letter-spacing: 0.18em;
  text-indent: 0.18em;
}
.rule {
  display: block;
  width: 36px;
  height: 1px;
  margin: 22px 0 18px;
  background: #8c6232;
}
.note {
  margin: 0 0 30px;
  max-width: 22em;
  color: #66756d;
  font-size: 14px;
  line-height: 1.75;
}

.field { display: flex; flex-direction: column; gap: 8px; margin: 0 0 6px; }
.field > span { font-size: 13px; font-weight: 600; letter-spacing: 0.06em; }
.field em {
  min-height: 16px;
  font-style: normal;
  font-size: 12px;
  line-height: 16px;
  color: var(--danger);
}
.field :deep(.el-input__wrapper) {
  background: #fffdf8;
  border-radius: 10px;
  box-shadow: 0 0 0 1px #ddd2c0 inset;
  min-height: 50px;
  padding: 2px 14px;
  transition: box-shadow 0.15s ease;
}
.field :deep(.el-input__wrapper:hover) { box-shadow: 0 0 0 1px #c9b89a inset; }
.field :deep(.el-input__wrapper:focus-within),
.field :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #0e6b52 inset, 0 0 0 4px rgba(14, 107, 82, 0.14);
}
.field.bad :deep(.el-input__wrapper),
.field.bad :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 1px var(--danger) inset, 0 0 0 4px rgba(141, 58, 50, 0.1);
}
.field :deep(.el-input__inner) { color: var(--ink); font-size: 15px; }
.field :deep(.el-input__suffix) { color: #8a7968; }

.go {
  width: 100%;
  height: 50px;
  margin-top: 8px;
  border-radius: 10px;
  letter-spacing: 0.16em;
  font-size: 15px;
  --el-button-bg-color: #0e6b52;
  --el-button-border-color: #0e6b52;
  --el-button-text-color: #fffdf8;
  --el-button-hover-bg-color: #127a5e;
  --el-button-hover-border-color: #127a5e;
  --el-button-hover-text-color: #fffdf8;
  --el-button-active-bg-color: #0b5642;
  --el-button-active-border-color: #0b5642;
  --el-button-active-text-color: #fffdf8;
}
.aside {
  margin: 18px 0 0;
  color: #8a7968;
  font-size: 12px;
  letter-spacing: 0.04em;
}

:global(body.is-login) { background: #14241c; }

@media (max-width: 980px) {
  .login { height: auto; grid-template-columns: 1fr; overflow: auto; }
  .face { padding: 40px 24px 32px; border-right: 0; }
  .quiet { position: static; margin-top: 22px; }
  .chop { width: 156px; }
  .chop-char { font-size: 78px; }
  .name { margin-top: 16px; font-size: 24px; }
  .inscription { margin-top: 16px; }
  .sheet { padding: 32px 24px 48px; }
  .running { display: none; }
  .sheet h1 { font-size: 48px; }
}

@media (max-width: 640px) {
  .inscription { letter-spacing: 0.12em; }
  .sheet { padding: 28px 18px 40px; }
  .sheet h1 { font-size: 42px; }
}
</style>
