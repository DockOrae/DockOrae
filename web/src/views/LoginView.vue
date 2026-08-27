<template>
  <div class="login-page relative min-h-screen flex items-center justify-center overflow-hidden">
    <!-- 背景照片 -->
    <img src="/bg.jpg" alt="" class="login-bg" />
    <div class="login-bg-overlay" />

    <div class="relative w-full max-w-lg px-4 fade-up">
      <div class="text-center mb-8">
        <img src="/logo.svg" alt="logo" class="login-logo" />
        <h1 class="text-2xl font-bold tracking-tight text-white drop-shadow">{{ t('app.name') }}</h1>
        <p class="text-sm text-white/70 mt-1 drop-shadow">{{ t('login.subtitle') }}</p>
      </div>

      <div class="login-card p-8">
        <div
          v-if="error"
          class="mb-4 flex items-center gap-2 px-3 py-2.5 rounded-lg bg-danger/10 border border-danger/30 text-danger text-[13px]"
        >
          <Icon name="alert" size="14" /> {{ error }}
        </div>

        <!-- 第一步:用户名 + 密码 -->
        <form v-if="!totpStep" @submit.prevent="doLogin">
          <div class="mb-4">
            <label class="label">{{ t('login.username') }}</label>
            <div class="relative">
              <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted">
                <Icon name="box" size="15" />
              </span>
              <input v-model="form.username" class="input !pl-9" :placeholder="t('login.usernamePh')" autocomplete="username" />
            </div>
          </div>
          <div class="mb-5">
            <label class="label">{{ t('login.password') }}</label>
            <div class="relative">
              <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted">
                <Icon :name="showPw ? 'eyeOff' : 'eye'" size="15" />
              </span>
              <input
                v-model="form.password"
                :type="showPw ? 'text' : 'password'"
                class="input !pl-9 !pr-10"
                :placeholder="t('login.passwordPh')"
                autocomplete="current-password"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-text"
                @click="showPw = !showPw"
              >
                <Icon :name="showPw ? 'eye' : 'eyeOff'" size="15" />
              </button>
            </div>
          </div>
          <button type="submit" class="btn btn-brand w-full justify-center !py-2.5 text-sm font-semibold" :disabled="loading">
            <span v-if="loading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
            {{ loading ? t('login.loggingIn') : t('login.login') }}
          </button>
        </form>

        <!-- 第二步:2FA 动态码 -->
        <form v-else @submit.prevent="doTotp">
          <div class="mb-4">
            <div class="flex items-center gap-2 mb-1">
              <Icon name="key" size="15" class="text-brand" />
              <label class="label !mb-0">{{ t('login.totpTitle') }}</label>
            </div>
            <p class="text-[12px] text-muted">{{ t('login.totpDesc') }}</p>
          </div>
          <div class="mb-5">
            <input
              v-model="totpCode"
              class="input text-center !text-lg tracking-[0.5em] font-mono"
              :placeholder="t('login.totpPh')"
              maxlength="6"
              inputmode="numeric"
              autocomplete="one-time-code"
              autofocus
            />
          </div>
          <button type="submit" class="btn btn-brand w-full justify-center !py-2.5 text-sm font-semibold" :disabled="loading">
            <span v-if="loading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
            {{ loading ? t('login.verifying') : t('login.verify') }}
          </button>
          <button type="button" class="w-full text-center text-[12px] text-muted hover:text-text mt-3" @click="totpStep = false">
            ← {{ t('common.back') }}
          </button>
        </form>

        <p v-if="showDefaultHint" class="text-center text-[11px] text-muted mt-4">
          {{ t('login.defaultAccount', { user: 'admin', pass: '123456' }) }}
        </p>

        <!-- 语言 / 主题切换(卡片底部) -->
        <div class="flex items-center justify-between mt-4 pt-4 border-t" style="border-color: var(--dm-login-input-border)">
          <SwitchAppearance />
          <ToggleLocale />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import ToggleLocale from '../components/ToggleLocale.vue'
import SwitchAppearance from '../components/SwitchAppearance.vue'
import { api, setToken } from '../api'
import { toastErr, toastOk } from '../toast'
import { applyUser } from '../store'

const { t } = useI18n()
const router = useRouter()
const form = reactive({ username: 'admin', password: '' })
const loading = ref(false)
const error = ref('')
const showPw = ref(false)
const totpStep = ref(false)
const totpCode = ref('')
// 仅当 admin 仍是默认密码(未改密)时显示"默认账号"提示
const showDefaultHint = ref(false)

onMounted(async () => {
  try {
    const r = await api('/system/default-account')
    showDefaultHint.value = !!r.show
  } catch {
    showDefaultHint.value = false
  }
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
.login-bg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.login-bg-overlay {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(5, 8, 14, 0.55) 0%, rgba(5, 8, 14, 0.75) 100%),
    radial-gradient(ellipse at center, transparent 0%, rgba(5, 8, 14, 0.35) 100%);
}
.login-logo {
  width: 72px;
  height: 72px;
  object-fit: contain;
  filter: drop-shadow(0 8px 24px rgba(0, 0, 0, 0.45));
  margin: 0 auto 14px;
  display: block;
}
.login-card {
  background: var(--dm-login-card);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
  transition: background-color 0.25s ease;
}
.login-card :deep(.label) {
  color: var(--dm-login-text);
}
.login-card :deep(.input) {
  background: var(--dm-login-input);
  border-color: var(--dm-login-input-border);
  color: var(--dm-text);
}
.login-card :deep(.input::placeholder) {
  color: var(--dm-login-placeholder);
}
</style>
