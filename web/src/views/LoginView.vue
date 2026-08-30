<template>
  <div :class="['login-app', { 'is-dark': isDark }]">
    <!-- 背景图 + 遮罩(1Panel 风格;自定义壁纸优先,失败回退内置 bg.jpg) -->
    <img :src="bgFailed ? '/bg.jpg' : '/api/system/wallpaper'" alt="" class="login-bg" @error="onBgError" />
    <div class="login-bg-overlay" />
    <!-- 右上角工具栏:主题切换 + 语言切换(shadcn-vue 组件,与管理页一致) -->
    <div class="login-toolbar">
      <ThemeToggle />
      <ToggleLocale />
    </div>

    <div class="login-wrapper">
      <div class="login-card">
        <div class="brand">
          <img src="/logo.svg" alt="logo" class="brand-logo" />
          <span class="brand-name">Docker Manager</span>
          <span class="brand-accent" aria-hidden="true" />
        </div>
        <h2 class="welcome">
          <b :key="headlineIndex">{{ headlineWords[headlineIndex] }}</b>
        </h2>

        <div v-if="error" class="login-error">
          <Icon name="alert" size="14" /> {{ error }}
        </div>

        <!-- 第一步:用户名 + 密码 -->
        <form v-if="!totpStep" class="login-form" @submit.prevent="doLogin">
          <label class="f-label">{{ t('login.username') }}</label>
          <div class="f-field">
            <Icon name="user" size="15" class="f-icon" />
            <input
              v-model="form.username"
              class="f-input"
              :placeholder="t('login.usernamePh')"
              autocomplete="username"
              autofocus
            />
          </div>
          <label class="f-label">{{ t('login.password') }}</label>
          <div class="f-field">
            <Icon name="lock" size="15" class="f-icon" />
            <input
              v-model="form.password"
              :type="showPw ? 'text' : 'password'"
              class="f-input"
              :placeholder="t('login.passwordPh')"
              autocomplete="current-password"
            />
            <button type="button" class="f-eye" @click="showPw = !showPw">
              <Icon :name="showPw ? 'eyeOff' : 'eye'" size="15" />
            </button>
          </div>
          <button type="submit" class="f-submit" :disabled="loading">
            <span v-if="loading" class="spinner" />
            {{ loading ? t('login.loggingIn') : t('login.login') }}
          </button>
        </form>

        <!-- 第二步:2FA 动态码 -->
        <form v-else class="login-form" @submit.prevent="doTotp">
          <label class="f-label">{{ t('login.totpTitle') }}</label>
          <p class="f-hint">{{ t('login.totpDesc') }}</p>
          <div class="f-field">
            <Icon name="key" size="15" class="f-icon" />
            <input
              v-model="totpCode"
              class="f-input f-totp"
              :placeholder="t('login.totpPh')"
              maxlength="6"
              inputmode="numeric"
              autocomplete="one-time-code"
            />
          </div>
          <button type="submit" class="f-submit" :disabled="loading">
            <span v-if="loading" class="spinner" />
            {{ loading ? t('login.verifying') : t('login.verify') }}
          </button>
          <button type="button" class="f-back" @click="totpStep = false">← {{ t('common.back') }}</button>
        </form>

        <p v-if="showDefaultHint" class="default-hint">
          {{ t('login.defaultAccount', { user: 'admin', pass: '123456' }) }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import ThemeToggle from '../components/ThemeToggle.vue'
import ToggleLocale from '../components/ToggleLocale.vue'
import { api, setToken } from '../api'
import { toastErr, toastOk } from '../toast'
import { applyUser } from '../store'
import { isDark } from '../store'

const { t, locale } = useI18n()
const router = useRouter()
const form = reactive({ username: 'admin', password: '' })
const loading = ref(false)
const error = ref('')
const showPw = ref(false)
const totpStep = ref(false)
const totpCode = ref('')
const headlineIndex = ref(0)
// 仅当 admin 仍是默认密码(未改密)时显示"默认账号"提示
const showDefaultHint = ref(false)
// 自定义壁纸加载失败 → 回退内置背景
const bgFailed = ref(false)
function onBgError() {
  bgFailed.value = true
}

// 欢迎语轮播(仿 3x-ui:2 秒切换)
const headlineWords = computed(() => [t('login.welcomeBack', { name: '' }).replace(/,\s*$/, ''), t('login.subtitle')])
setInterval(() => {
  headlineIndex.value = (headlineIndex.value + 1) % 2
}, 2000)

onMounted(() => {
  api('/system/default-account')
    .then((r) => (showDefaultHint.value = !!r.show))
    .catch(() => (showDefaultHint.value = false))
})
async function doLogin() {
  if (!form.username || !form.password) {
    error.value = t('login.errFill')
    return
  }
  loading.value = true
  error.value = ''
  try {
    const r = await api('/login', { method: 'POST', json: form })
    if (r.totp_required) {
      totpStep.value = true
      totpCode.value = ''
      return
    }
    setToken(r.token)
    applyUser(r)
    toastOk(t('login.welcomeBack', { name: r.nickname || r.username }))
    router.push('/')
  } catch (e) {
    error.value = e.message
    toastErr(e.message)
  } finally {
    loading.value = false
  }
}

async function doTotp() {
  if (!totpCode.value) {
    error.value = t('login.errTotpFill')
    return
  }
  loading.value = true
  error.value = ''
  try {
    const r = await api('/login/totp', { method: 'POST', json: { username: form.username, code: totpCode.value } })
    setToken(r.token)
    applyUser(r)
    toastOk(t('login.welcomeBack', { name: r.nickname || r.username }))
    router.push('/')
  } catch (e) {
    error.value = e.message
    toastErr(e.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* ---------- 仿 3x-ui 登录页:渐变背景 + 光斑动画 + 网格(粉色品牌) ---------- */
.login-app {
  --bg-page: #fdf2f8;
  --bg-card: rgba(255, 255, 255, 0.72);
  --bg-card-solid: #ffffff;
  --color-text: rgba(0, 0, 0, 0.88);
  --color-text-subtle: rgba(0, 0, 0, 0.55);
  --color-accent: #ec4899;
  --color-border: rgba(255, 255, 255, 0.6);
  --shadow-card: 0 1px 3px rgba(0, 0, 0, 0.04), 0 18px 50px rgba(236, 72, 153, 0.18);
  --blob-1: rgba(236, 72, 153, 0.5);
  --blob-2: rgba(168, 85, 247, 0.45);
  --blob-3: rgba(244, 114, 182, 0.4);
  --blob-4: rgba(251, 191, 36, 0.3);
  --blob-5: rgba(56, 189, 248, 0.35);
  --grid-color: rgba(236, 72, 153, 0.06);
  --vignette: radial-gradient(ellipse at center, transparent 30%, rgba(0, 0, 0, 0.05) 100%);

  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background: linear-gradient(135deg, #fdf2f8 0%, #fce7f3 50%, #f5f3ff 100%);
  transition: background 0.3s ease;
}

/* 背景图 + 遮罩(1Panel 风格) */
.login-bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
}
.login-bg-overlay {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(5, 8, 14, 0.55) 0%, rgba(5, 8, 14, 0.75) 100%),
    radial-gradient(ellipse at center, transparent 0%, rgba(5, 8, 14, 0.35) 100%);
  z-index: 1;
}

.login-app.is-dark {
  --bg-card: rgba(28, 26, 34, 0.55);
  --bg-card-solid: #1c1a22;
  --color-text: rgba(255, 255, 255, 0.92);
  --color-text-subtle: rgba(255, 255, 255, 0.55);
  --color-accent: #f472b6;
  --color-border: rgba(255, 255, 255, 0.1);
  --shadow-card: 0 1px 3px rgba(0, 0, 0, 0.4), 0 20px 60px rgba(236, 72, 153, 0.22);
  --blob-1: rgba(244, 114, 182, 0.35);
  --blob-2: rgba(168, 85, 247, 0.3);
  --blob-3: rgba(236, 72, 153, 0.25);
  --blob-4: rgba(251, 146, 60, 0.15);
  --blob-5: rgba(129, 140, 248, 0.25);
  --grid-color: rgba(255, 255, 255, 0.04);
  background: radial-gradient(ellipse at 25% 20%, #1f1220 0%, #0d0b12 60%);
}

.login-app::before,
.login-app::after {
  content: '';
  position: absolute;
  width: 70vw;
  height: 70vw;
  max-width: 900px;
  max-height: 900px;
  border-radius: 50%;
  filter: blur(100px);
  pointer-events: none;
  z-index: 0;
  will-change: transform;
}
.login-app::before {
  top: -25vw;
  left: -20vw;
  background: radial-gradient(circle, var(--blob-1) 0%, transparent 65%);
  animation: blob-drift-a 24s ease-in-out infinite alternate;
}
.login-app::after {
  bottom: -25vw;
  right: -20vw;
  background: radial-gradient(circle, var(--blob-2) 0%, transparent 65%);
  animation: blob-drift-b 30s ease-in-out infinite alternate;
}
.login-wrapper::before {
  content: '';
  position: absolute;
  bottom: 5%;
  left: 10%;
  width: 35vw;
  height: 35vw;
  max-width: 500px;
  max-height: 500px;
  border-radius: 50%;
  background: radial-gradient(circle, var(--blob-5) 0%, transparent 65%);
  filter: blur(90px);
  pointer-events: none;
  z-index: 0;
  will-change: transform;
  animation: blob-drift-e 32s ease-in-out infinite alternate;
}
.login-wrapper::after {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--grid-color) 1px, transparent 1px),
    linear-gradient(90deg, var(--grid-color) 1px, transparent 1px);
  background-size: 48px 48px;
  background-position: center;
  -webkit-mask-image: radial-gradient(ellipse at center, black 30%, transparent 75%);
  mask-image: radial-gradient(ellipse at center, black 30%, transparent 75%);
  pointer-events: none;
  z-index: 0;
}

@keyframes blob-drift-a {
  0% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(18vw, 10vh) scale(1.15); }
  100% { transform: translate(34vw, 22vh) scale(1.25); }
}
@keyframes blob-drift-b {
  0% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(-16vw, -10vh) scale(1.12); }
  100% { transform: translate(-30vw, -22vh) scale(1.2); }
}
@keyframes blob-drift-e {
  0% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(14vw, -8vh) scale(1.1); }
  100% { transform: translate(-6vw, 12vh) scale(1.15); }
}
@media (prefers-reduced-motion: reduce) {
  .login-app::before,
  .login-app::after,
  .login-wrapper::before {
    animation: none;
  }
}

/* ---------- 右上角工具栏(shadcn 组件:主题按钮 + 语言 DropdownMenu) ---------- */
.login-toolbar {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 10;
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

/* 登录页工具栏按钮:放大(44px 圆 + 20px 图标)+ 背景醒目(背景图上清晰可见) */
.login-toolbar :deep(button) {
  width: 44px;
  height: 44px;
  min-width: 44px;
  padding: 0;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.18);
  color: #ffffff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
  transition: all 0.2s;
}
.login-toolbar :deep(button:hover) {
  background: rgba(255, 255, 255, 0.3);
  border-color: var(--color-brand);
  color: var(--color-brand);
  transform: translateY(-1px);
}
.login-toolbar :deep(svg) {
  width: 20px;
  height: 20px;
}

/* ---------- 居中卡片 ---------- */
.login-wrapper {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
}
.login-card {
  position: relative;
  width: 100%;
  max-width: 400px;
  background: var(--bg-card);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  padding: 40px 32px 28px;
  box-shadow: var(--shadow-card);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  backdrop-filter: blur(24px) saturate(180%);
  z-index: 2;
}
.login-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 20px;
  padding: 1px;
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.5),
    rgba(255, 255, 255, 0) 40%,
    rgba(236, 72, 153, 0.35) 80%
  );
  -webkit-mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
}
.is-dark .login-card::before {
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.15),
    rgba(255, 255, 255, 0) 40%,
    rgba(244, 114, 182, 0.4) 80%
  );
}

.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.brand-logo {
  width: 52px;
  height: 52px;
  object-fit: contain;
}
.brand-name {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 1.5px;
  background: linear-gradient(135deg, var(--color-accent), #ec4899);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: transparent;
}
.brand-accent {
  display: block;
  width: 40px;
  height: 3px;
  border-radius: 2px;
  background: linear-gradient(90deg, var(--color-accent), #ec4899);
}
.welcome {
  text-align: center;
  color: var(--color-text);
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  min-height: 30px;
  margin: 10px 0 22px;
  letter-spacing: 0.3px;
}
.welcome b {
  display: inline-block;
  font-weight: inherit;
  animation: headline-in 280ms ease both;
}
@keyframes headline-in {
  0% { opacity: 0; transform: translateY(6px); }
  100% { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) {
  .welcome b { animation: none; }
}

.login-error {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  padding: 9px 12px;
  border-radius: 10px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  font-size: 13px;
}

/* ---------- 表单 ---------- */
.f-label {
  display: block;
  color: var(--color-text);
  font-weight: 500;
  font-size: 14px;
  margin: 14px 0 6px;
}
.f-field {
  position: relative;
  display: flex;
  align-items: center;
}
.f-icon {
  position: absolute;
  left: 12px;
  color: var(--color-text-subtle);
  pointer-events: none;
  z-index: 1;
}
.f-input {
  width: 100%;
  height: 42px;
  padding: 0 40px 0 38px;
  border-radius: 10px;
  border: 1px solid rgba(0, 0, 0, 0.12);
  background: var(--bg-card-solid);
  color: var(--color-text);
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.is-dark .f-input {
  border-color: rgba(255, 255, 255, 0.14);
}
.f-input:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(236, 72, 153, 0.15);
}
.f-input::placeholder {
  color: var(--color-text-subtle);
}
.f-eye {
  position: absolute;
  right: 10px;
  border: none;
  background: transparent;
  color: var(--color-text-subtle);
  cursor: pointer;
  display: flex;
  padding: 4px;
}
.f-eye:hover {
  color: var(--color-accent);
}
.f-totp {
  text-align: center;
  letter-spacing: 0.5em;
  font-family: ui-monospace, monospace;
  font-size: 16px;
}
.f-submit {
  width: 100%;
  height: 42px;
  margin-top: 20px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #ec4899, #d946ef);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: opacity 0.2s, transform 0.1s;
}
.f-submit:hover:not(:disabled) {
  opacity: 0.92;
}
.f-submit:active:not(:disabled) {
  transform: scale(0.99);
}
.f-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.f-back {
  width: 100%;
  margin-top: 10px;
  border: none;
  background: transparent;
  color: var(--color-text-subtle);
  font-size: 12px;
  cursor: pointer;
  text-align: center;
}
.f-back:hover {
  color: var(--color-accent);
}
.f-hint {
  color: var(--color-text-subtle);
  font-size: 12px;
  margin: 0 0 6px;
}

.default-hint {
  text-align: center;
  font-size: 11px;
  color: var(--color-text-subtle);
  margin-top: 16px;
}

/* 语言菜单动画 */
.dm-drop-enter-active,
.dm-drop-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.dm-drop-enter-from,
.dm-drop-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}
</style>
