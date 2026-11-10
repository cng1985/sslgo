<template>
  <div class="login">
    <div class="wash" aria-hidden="true"></div>
    <svg class="watermark" viewBox="0 0 400 400" aria-hidden="true">
      <g fill="none" stroke="#d7b56a">
        <circle cx="200" cy="200" r="190" stroke-opacity="0.35" />
        <circle cx="200" cy="200" r="166" stroke-opacity="0.55" />
        <circle cx="200" cy="200" r="142" stroke-dasharray="2 8" stroke-opacity="0.45" />
        <circle cx="200" cy="200" r="108" stroke="#f3d7a1" stroke-opacity="0.75" />
        <circle cx="200" cy="200" r="82" stroke="#f3d7a1" stroke-opacity="0.9" />
      </g>
      <g transform="translate(200 200)" stroke="#f3d7a1" stroke-width="1.6" stroke-linecap="round">
        <line y1="-166" y2="-190" />
        <line y1="-166" y2="-190" transform="rotate(30)" />
        <line y1="-166" y2="-190" transform="rotate(60)" />
        <line y1="-166" y2="-190" transform="rotate(90)" />
        <line y1="-166" y2="-190" transform="rotate(120)" />
        <line y1="-166" y2="-190" transform="rotate(150)" />
        <line y1="-166" y2="-190" transform="rotate(180)" />
        <line y1="-166" y2="-190" transform="rotate(210)" />
        <line y1="-166" y2="-190" transform="rotate(240)" />
        <line y1="-166" y2="-190" transform="rotate(270)" />
        <line y1="-166" y2="-190" transform="rotate(300)" />
        <line y1="-166" y2="-190" transform="rotate(330)" />
      </g>
      <text x="200" y="224" text-anchor="middle" fill="#f3d7a1" font-size="92" font-family="Palatino, 'Songti SC', serif">证</text>
    </svg>

    <div class="document">
      <section class="face">
        <p class="spine">证枢 · SSL</p>
        <header class="brand">
          <span class="seal">证</span>
          <div>
            <strong class="brand-mark">证枢</strong>
            <small>SSL 控制台</small>
          </div>
        </header>

        <div class="hero">
          <p class="eyebrow">LET'S ENCRYPT · NGINX</p>
          <span class="rule"></span>
          <h1 class="brand-mark">申请，替换，<br>再重启。</h1>
          <p class="lede">向证书机构申请免费证书，按域名覆盖服务器上的旧文件，检测通过后再重启 Nginx。</p>
        </div>

        <div class="foot">
          <ol class="steps">
            <li>
              <b>01</b>
              <div>
                <strong>登记</strong>
                <span>ACME 账户与 Nginx 目录</span>
              </div>
            </li>
            <li>
              <b>02</b>
              <div>
                <strong>验证</strong>
                <span>HTTP-01，或用 DNS-01 签发通配符</span>
              </div>
            </li>
            <li>
              <b>03</b>
              <div>
                <strong>部署</strong>
                <span>写入新证书，检测失败会恢复旧文件</span>
              </div>
            </li>
          </ol>
          <p class="quiet">私钥留在本机 · 租户随账号隔离</p>
        </div>
      </section>

      <section class="sheet">
        <form @submit.prevent="submit">
          <p class="kicker">账号</p>
          <h2 class="brand-mark">登录控制台</h2>
          <p class="note">使用已分配的账号进入。无需填写租户编号。</p>

          <label class="field" :class="{ bad: showUserError }" for="login-username">
            <span>用户名</span>
            <el-input
              id="login-username"
              ref="usernameInput"
              v-model="form.username"
              name="username"
              autocomplete="username"
              placeholder="请输入用户名"
              size="large"
            >
              <template #prefix>
                <svg class="ico" viewBox="0 0 24 24" aria-hidden="true">
                  <circle cx="12" cy="8" r="3.25" fill="none" stroke="currentColor" stroke-width="1.6" />
                  <path d="M5.2 19.2c1.35-3.05 3.7-4.5 6.8-4.5s5.45 1.45 6.8 4.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
                </svg>
              </template>
            </el-input>
            <em v-if="showUserError">请输入用户名</em>
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
              placeholder="请输入密码"
              size="large"
            >
              <template #prefix>
                <svg class="ico" viewBox="0 0 24 24" aria-hidden="true">
                  <rect x="5" y="10.5" width="14" height="9" rx="2" fill="none" stroke="currentColor" stroke-width="1.6" />
                  <path d="M8 10.5V8a4 4 0 0 1 8 0v2.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
                </svg>
              </template>
            </el-input>
            <em v-if="showPasswordError">请输入密码</em>
          </label>

          <el-button native-type="submit" type="primary" class="go" :loading="loading">进入控制台</el-button>
          <p class="aside">登录后可在右上角修改密码</p>
        </form>
      </section>
    </div>
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
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: safe center;
  padding: 28px 24px;
  position: relative;
  overflow: hidden;
  background-color: #0c1612;
  background-image: radial-gradient(rgba(243, 215, 161, 0.09) 1px, transparent 1px);
  background-size: 22px 22px;
  color: var(--ink);
}

.wash {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(820px 520px at 12% 88%, rgba(14, 107, 82, 0.42), transparent 62%),
    radial-gradient(760px 480px at 92% 8%, rgba(215, 181, 106, 0.18), transparent 58%),
    linear-gradient(180deg, rgba(8, 14, 11, 0.15), rgba(8, 14, 11, 0.55));
}

.watermark {
  position: absolute;
  width: min(680px, 62vw);
  right: -8vw;
  bottom: -16vh;
  opacity: 0.28;
  pointer-events: none;
}

.document {
  width: min(1120px, 100%);
  min-height: 640px;
  display: grid;
  grid-template-columns: 1.12fr 0.88fr;
  border-radius: 28px;
  overflow: hidden;
  position: relative;
  z-index: 1;
  border: 1px solid rgba(215, 181, 106, 0.4);
  box-shadow:
    0 1px 0 rgba(255, 248, 230, 0.16) inset,
    0 30px 70px rgba(0, 0, 0, 0.38),
    0 80px 140px rgba(0, 0, 0, 0.28);
  animation: rise 0.55s ease both;
}

.face {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 28px;
  padding: 42px 46px 32px 72px;
  color: #f6f1e7;
  background:
    radial-gradient(420px 280px at 100% 100%, rgba(215, 181, 106, 0.14), transparent 70%),
    linear-gradient(165deg, #1c3329 0%, #14241c 46%, #101c17 100%);
}

.spine {
  position: absolute;
  left: 22px;
  top: 46px;
  bottom: 36px;
  margin: 0;
  writing-mode: vertical-rl;
  letter-spacing: 0.42em;
  font-family: Palatino, "Iowan Old Style", "Songti SC", "Noto Serif CJK SC", "Source Han Serif SC", serif;
  font-size: 13px;
  color: rgba(243, 215, 161, 0.72);
}

.brand { display: flex; align-items: center; gap: 12px; }
.brand strong { display: block; font-size: 22px; letter-spacing: 0.14em; }
.brand small { color: #b7c7be; letter-spacing: 0.12em; }
.seal {
  width: 46px;
  height: 46px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  border: 1px solid #d7b56a;
  color: #f3d7a1;
  box-shadow: 0 0 0 6px rgba(215, 181, 106, 0.08);
  font-family: Palatino, "Songti SC", "Noto Serif CJK SC", serif;
  font-size: 20px;
}

.hero { flex: 1; display: flex; flex-direction: column; justify-content: center; }
.eyebrow, .kicker {
  margin: 0;
  font-size: 12px;
  letter-spacing: 0.26em;
}
.eyebrow { color: #f3d7a1; }
.rule {
  display: block;
  width: 48px;
  height: 1px;
  margin-top: 16px;
  background: linear-gradient(90deg, #f3d7a1, rgba(215, 181, 106, 0));
}
.face h1 {
  margin: 16px 0 0;
  font-size: 54px;
  line-height: 1.12;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.lede {
  margin: 18px 0 0;
  max-width: 420px;
  color: #d5e2da;
  font-size: 15px;
  line-height: 1.75;
}

.steps {
  list-style: none;
  margin: 0;
  padding: 20px 0 0;
  border-top: 1px solid rgba(243, 215, 161, 0.22);
  display: grid;
  gap: 14px;
}
.steps li { display: grid; grid-template-columns: 36px minmax(0, 1fr); gap: 8px; align-items: start; }
.steps b {
  font-family: Palatino, "Songti SC", "Noto Serif CJK SC", serif;
  font-weight: 500;
  color: #f3d7a1;
  letter-spacing: 0.06em;
}
.steps strong { display: block; font-size: 15px; font-weight: 600; }
.steps span { display: block; margin-top: 2px; color: #c9d6ce; font-size: 13px; line-height: 1.5; }
.quiet { margin: 16px 0 0; color: #b7c7be; font-size: 12px; letter-spacing: 0.08em; }

.sheet {
  display: flex;
  align-items: center;
  padding: 48px 52px;
  background: linear-gradient(180deg, #fffdf8 0%, #f6f0e4 100%);
  box-shadow: inset 1px 0 0 rgba(215, 181, 106, 0.28);
}
.sheet form { width: 100%; }
.kicker { color: var(--brass); }
.sheet h2 {
  margin: 10px 0 8px;
  font-size: 36px;
  font-weight: 600;
  letter-spacing: 0.06em;
}
.note {
  margin: 0 0 28px;
  color: #66756d;
  font-size: 14px;
  line-height: 1.7;
}

.field { display: flex; flex-direction: column; gap: 8px; margin: 0 0 18px; }
.field > span { font-size: 13px; font-weight: 600; letter-spacing: 0.04em; }
.field em { font-style: normal; color: var(--danger); font-size: 12px; line-height: 1.4; }
.ico { width: 18px; height: 18px; display: block; }

.field :deep(.el-input__wrapper) {
  background: #fffdf8;
  border-radius: 12px;
  box-shadow: 0 0 0 1px #e4d8c6 inset;
  min-height: 48px;
  padding: 4px 12px;
  transition: box-shadow 0.15s ease;
}
.field :deep(.el-input__wrapper:hover) { box-shadow: 0 0 0 1px #c9b89a inset; }
.field :deep(.el-input__wrapper:focus-within),
.field :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #0e6b52 inset, 0 0 0 4px rgba(14, 107, 82, 0.16);
}
.field :deep(.el-input__prefix) { color: #8d7b66; }
.field :deep(.el-input__inner) { color: var(--ink); }
.field.bad :deep(.el-input__wrapper),
.field.bad :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 1px var(--danger) inset, 0 0 0 4px rgba(141, 58, 50, 0.12);
}

.go {
  width: 100%;
  height: 48px;
  margin-top: 6px;
  border-radius: 12px;
  letter-spacing: 0.18em;
  font-size: 15px;
  --el-button-bg-color: #0e6b52;
  --el-button-border-color: #0e6b52;
  --el-button-hover-bg-color: #127a5e;
  --el-button-hover-border-color: #127a5e;
  --el-button-active-bg-color: #0b5642;
  --el-button-active-border-color: #0b5642;
}
.aside { margin: 16px 0 0; text-align: center; color: #8a7968; font-size: 12px; }

:global(body.is-login) { background: #0c1612; }

@keyframes rise {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: none; }
}

@media (prefers-reduced-motion: reduce) {
  .document { animation: none; }
}

@media (max-width: 980px) {
  .document { grid-template-columns: 1fr; min-height: 0; }
  .spine { display: none; }
  .face { padding: 28px 24px 8px; gap: 18px; }
  .hero { flex: none; }
  .face h1 { font-size: 40px; }
  .steps, .quiet { display: none; }
  .sheet { box-shadow: none; padding: 28px 24px 32px; }
}

@media (max-width: 640px) {
  .login { padding: 16px; background-size: 18px 18px; }
  .face h1 { font-size: 34px; }
  .lede { font-size: 14px; }
  .sheet h2 { font-size: 30px; }
  .sheet { padding: 24px 18px 28px; }
}
</style>
