<template>
  <div class="space-y-4 fade-up">
    <!-- 分段按钮(仿 1Panel:一行占满,选中 2px 主题色边框) -->
    <SegmentedTabs :tabs="tabs" :active="active" @update:active="active = $event" />

    <!-- ============ 面板设置 ============ -->
    <div v-show="active === 'panel'" class="space-y-4">
      <!-- 个人资料 -->
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="box" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('settings.profile') }}</h2>
        </div>
        <div class="flex items-center gap-4 mb-5">
          <img :src="avatarPreview" alt="avatar" class="w-16 h-16 rounded-2xl object-cover ring-2 ring-white/15 shadow-lg" />
          <div class="flex-1">
            <button class="btn btn-ghost btn-sm" @click="avatarInput?.click()">
              <Icon name="image" size="13" /> {{ t('settings.' + (user.avatar ? 'changeAvatar' : 'uploadAvatar')) }}
            </button>
            <input ref="avatarInput" type="file" accept="image/jpeg,image/png,image/gif,image/webp" class="hidden" @change="uploadAvatar" />
            <p class="text-[11px] text-muted mt-1.5">{{ t('settings.avatarNote') }}</p>
          </div>
        </div>
        <form class="space-y-4" @submit.prevent="saveProfile">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="label">{{ t('settings.username') }}</label>
              <input v-model="profile.username" class="input" maxlength="32" />
              <p class="text-[11px] text-muted mt-1">{{ t('settings.usernameNote') }}</p>
            </div>
            <div>
              <label class="label">{{ t('settings.nickname') }}</label>
              <input v-model="profile.nickname" class="input" maxlength="32" :placeholder="t('settings.nicknamePh')" />
            </div>
          </div>
          <div v-if="profileErr" class="text-xs text-danger">{{ profileErr }}</div>
          <button type="submit" class="btn btn-brand" :disabled="profileLoading">
            <span v-if="profileLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
            {{ t('settings.saveProfile') }}
          </button>
        </form>
      </div>

      <!-- 镜像加速 -->
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="image" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('settings.mirrors') }}</h2>
          <span v-if="mirrorPath" class="ml-auto text-[11px] text-muted truncate max-w-[200px]" :title="mirrorPath">{{ mirrorPath }}</span>
        </div>
        <p class="text-[12px] text-muted mb-3 leading-relaxed">{{ t('settings.mirrorHelper') }}</p>
        <textarea v-model="mirrorsText" class="input !h-28 code-panel font-mono text-[12px] leading-relaxed" :placeholder="t('settings.mirrorPlaceholder')" spellcheck="false" />
        <div class="flex items-center gap-3 mt-3">
          <button class="btn btn-primary btn-sm" :disabled="savingMirrors" @click="saveMirrors">
            <Icon name="save" size="13" /> {{ t('settings.saveMirrors') }}
          </button>
          <span v-if="mirrorMsg" class="text-[11px]" :class="mirrorOk ? 'text-ok' : 'text-danger'">{{ mirrorMsg }}</span>
        </div>
        <p class="text-[11px] text-muted mt-3 leading-relaxed">{{ t('settings.mirrorRestartHint') }}</p>
      </div>
    </div>

    <!-- ============ 安全设置 ============ -->
    <div v-show="active === 'safe'" class="space-y-4">
      <!-- 双因素验证 -->
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="key" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('settings.twoFactor') }}</h2>
          <span class="ml-auto badge" :style="user.totpEnabled ? okStyle : mutedStyle">
            {{ t('settings.' + (user.totpEnabled ? 'totpEnabled' : 'totpDisabled')) }}
          </span>
        </div>
        <div v-if="!user.totpEnabled">
          <p class="text-[13px] text-muted mb-3">{{ t('settings.totpSetupDesc') }}</p>
          <div v-if="!totpSetup.uri" class="flex gap-2">
            <input v-model="totpSetup.password" type="password" class="input flex-1" :placeholder="t('settings.oldPwd')" />
            <button class="btn btn-brand" :disabled="totpBusy" @click="totpGetKey">
              <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.totpSetupBtn') }}
            </button>
          </div>
          <div v-else class="space-y-4">
            <div class="flex flex-col sm:flex-row items-center gap-5 bg-surface2/60 border border-line rounded-xl p-4">
              <img :src="totpQr" alt="QR" class="w-36 h-36 rounded-lg bg-white p-1.5" />
              <div class="flex-1 min-w-0 text-[12px] text-muted space-y-1">
                <p>{{ t('settings.totpQrDesc') }}</p>
                <p class="pt-1">{{ t('settings.manualKey') }}:</p>
                <code class="block font-mono text-[11px] text-text break-all select-all">{{ totpSetup.secret }}</code>
              </div>
            </div>
            <div class="flex gap-2">
              <input v-model="totpSetup.code" class="input flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
              <button class="btn btn-brand" :disabled="totpBusy" @click="totpEnable">
                <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                {{ t('settings.totpEnableBtn') }}
              </button>
            </div>
            <button class="text-[12px] text-muted hover:text-text" @click="totpReset">{{ t('common.cancel') }}</button>
          </div>
        </div>
        <div v-else>
          <p class="text-[13px] text-muted mb-3">{{ t('settings.totpDisableDesc') }}</p>
          <div v-if="!disableOpen" class="flex gap-2">
            <button class="btn btn-danger" @click="disableOpen = true">{{ t('settings.totpDisableBtn') }}</button>
          </div>
          <div v-else class="space-y-3">
            <input v-model="disableForm.password" type="password" class="input" :placeholder="t('settings.oldPwd')" />
            <div class="flex gap-2">
              <input v-model="disableForm.code" class="input flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
              <button class="btn btn-danger" :disabled="totpBusy" @click="totpDisable">
                <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                {{ t('settings.totpDisableBtn') }}
              </button>
            </div>
            <button class="text-[12px] text-muted hover:text-text" @click="disableOpen = false">{{ t('common.cancel') }}</button>
          </div>
        </div>
        <div v-if="totpErr" class="text-xs text-danger mt-2">{{ totpErr }}</div>
      </div>

      <!-- 修改密码 -->
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="key" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('settings.changePwd') }}</h2>
        </div>
        <form class="space-y-4" @submit.prevent="changePw">
          <div>
            <label class="label">{{ t('settings.oldPwd') }}</label>
            <input v-model="pw.old" type="password" class="input" autocomplete="current-password" />
          </div>
          <div>
            <label class="label">{{ t('settings.newPwd') }}</label>
            <input v-model="pw.neww" type="password" class="input" autocomplete="new-password" />
          </div>
          <div>
            <label class="label">{{ t('settings.confirmPwd') }}</label>
            <input v-model="pw.confirm" type="password" class="input" autocomplete="new-password" />
          </div>
          <div v-if="pwErr" class="text-xs text-danger">{{ pwErr }}</div>
          <button type="submit" class="btn btn-brand" :disabled="pwLoading">
            <span v-if="pwLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
            {{ t('settings.changePwdBtn') }}
          </button>
        </form>
      </div>
    </div>

    <!-- ============ 许可证 ============ -->
    <div v-show="active === 'license'">
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="key" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('license.title') }}</h2>
          <button class="btn btn-primary btn-sm ml-auto" @click="openLicForm">
            <Icon name="plus" size="13" /> {{ t('license.add') }}
          </button>
        </div>

        <!-- 授权表格(1Panel 风格) -->
        <div class="rounded-xl border border-line overflow-x-auto">
          <table class="table !m-0">
            <thead>
              <tr>
                <th class="th">{{ t('license.id') }}</th>
                <th class="th">{{ t('license.authorizedUser') }}</th>
                <th class="th">{{ t('license.edition') }}</th>
                <th class="th">{{ t('license.status') }}</th>
                <th class="th">{{ t('license.boundTo') }}</th>
                <th class="th">{{ t('license.expires') }}</th>
                <th class="th w-36">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="licActive && licInfo">
                <td class="td font-mono text-[11px]">{{ licKey.slice(0, 14) }}…</td>
                <td class="td">{{ licInfo.user || '-' }}</td>
                <td class="td">
                  <span class="badge" :style="okStyle">{{ t('license.' + licInfo.type) }}</span>
                </td>
                <td class="td">
                  <span class="badge" :style="licInfo.status === 'expired' ? dangerStyle : okStyle">
                    {{ t('license.' + (licInfo.status === 'expired' ? 'expired' : 'active')) }}
                  </span>
                </td>
                <td class="td font-mono text-[11px]">{{ licDeviceId }}</td>
                <td class="td">{{ fmtDate(licInfo.exp) }}</td>
                <td class="td">
                  <div class="flex items-center gap-1">
                    <button class="btn btn-icon btn-sm" :title="t('license.unbind')" :disabled="licBusy" @click="deactivate">
                      <Icon name="link" size="13" />
                    </button>
                    <button class="btn btn-icon btn-sm text-danger" :title="t('license.unbindDelete')" :disabled="licBusy" @click="deactivate">
                      <Icon name="trash" size="13" />
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-else>
                <td colspan="7" class="td text-center text-muted py-8">
                  {{ t('license.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="licErr" class="text-xs text-danger mt-3">{{ licErr }}</div>

        <!-- 添加许可证弹窗(1Panel:点击或拖动许可文件到此处 + 授权) -->
        <div v-if="licFormOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4" @click.self="licFormOpen = false">
          <div class="card p-6 w-full max-w-lg shadow-2xl fade-up" style="border-width: 2px; border-color: var(--dm-line);">
            <h3 class="text-base font-semibold mb-1">{{ t('license.add') }}</h3>
            <p class="text-[12px] text-muted mb-4">{{ t('license.addHint') }}</p>
            <div
              class="rounded-xl border-2 border-dashed border-line hover:border-brand/60 transition-all duration-200 cursor-pointer flex flex-col items-center justify-center gap-2 py-12 px-8 bg-surface2/40"
              :class="{ '!border-brand/80 bg-brand/5 scale-[1.01]': licDragging }"
              @click="licFileInput?.click()"
              @dragover.prevent="licDragging = true"
              @dragleave="licDragging = false"
              @drop.prevent="onLicDrop"
            >
              <Icon name="upload" size="32" class="text-muted" />
              <p class="text-[14px] font-medium">{{ t('license.dropZone') }}</p>
              <p v-if="licFileName" class="text-[12px] text-brand font-mono">{{ licFileName }}</p>
              <p v-else class="text-[11px] text-muted">{{ t('license.dropZoneHint') }}</p>
              <input ref="licFileInput" type="file" class="hidden" accept=".lic,.key,.txt" @change="onLicFile" />
            </div>
            <div class="flex justify-end gap-2 mt-5">
              <button class="btn btn-ghost btn-sm" @click="licFormOpen = false; resetLicForm()">{{ t('common.cancel') }}</button>
              <button
                class="btn btn-primary btn-sm transition-all duration-200"
                :class="licFile ? 'opacity-100 shadow-lg shadow-brand/25 ring-1 ring-brand/50' : 'opacity-35 grayscale'"
                :disabled="licBusy || !licFile"
                @click="authorizeFile"
              >
                <span v-if="licBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                <Icon v-else-if="licFile" name="check" size="13" />
                {{ licFile ? t('license.authorize') : t('license.selectFileFirst') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ============ 关于 ============ -->
    <div v-show="active === 'about'">
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="info" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('settings.about') }}</h2>
        </div>
        <dl class="text-[13px] space-y-2">
          <div class="flex"><dt class="text-muted w-28">{{ t('settings.panelName') }}</dt><dd class="font-medium">{{ t('app.name') }}</dd></div>
          <div class="flex"><dt class="text-muted w-28">{{ t('settings.version') }}</dt><dd>{{ t('app.version') }}</dd></div>
          <div class="flex"><dt class="text-muted w-28">{{ t('settings.stack') }}</dt><dd>Go (gin + Docker SDK) + Vue 3</dd></div>
          <div class="flex"><dt class="text-muted w-28">{{ t('settings.defaultAccount') }}</dt><dd>admin / 123456</dd></div>
          <div class="flex items-center gap-2">
            <dt class="text-muted w-28">{{ t('settings.source') }}</dt>
            <dd>
              <a href="https://github.com/MinimaxFlora/Docker_Manager_Go" target="_blank" rel="noopener" class="link flex items-center gap-1">
                github.com/MinimaxFlora/Docker_Manager_Go <Icon name="external" size="12" />
              </a>
            </dd>
          </div>
        </dl>
        <p class="text-[11px] text-muted mt-4 pt-4 border-t border-line leading-relaxed">
          {{ t('settings.securityNote') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import Icon from '../components/Icon.vue'
import SegmentedTabs from '../components/SegmentedTabs.vue'
import { api, setToken, getRegistryMirrors, saveRegistryMirrors, getLicense, activateLicenseFile, deactivateLicense } from '../api'
import { toastErr, toastOk } from '../toast'
import { applyUser, avatarUrl, loadLicense as refreshLicense, user } from '../store'

const { t } = useI18n()
const route = useRoute()
// 支持 ?tab=panel|safe|license|about 直达对应分组(用户菜单/锁定提示跳转)
const active = ref(['panel', 'safe', 'license', 'about'].includes(String(route.query.tab || '')) ? String(route.query.tab) : 'panel')
watch(
  () => route.query.tab,
  (v) => {
    if (['panel', 'safe', 'license', 'about'].includes(String(v))) active.value = String(v)
  }
)
const tabs = [
  { key: 'panel', labelKey: 'settings.panelTab' },
  { key: 'safe', labelKey: 'settings.safeTab' },
  { key: 'license', labelKey: 'license.title' },
  { key: 'about', labelKey: 'settings.about' },
]
const okStyle = { color: '#34d399', background: 'rgba(52,211,153,.12)', border: '1px solid rgba(52,211,153,.3)' }
const mutedStyle = { color: '#8b93a7', background: 'rgba(139,147,167,.12)', border: '1px solid rgba(139,147,167,.3)' }
const dangerStyle = { color: '#f87171', background: 'rgba(248,113,113,.12)', border: '1px solid rgba(248,113,113,.3)' }

const fmtDate = (ts) => (ts ? new Date(ts * 1000).toLocaleDateString() : '-')

// ---------- 个人资料 ----------
const profile = reactive({ username: user.username || 'admin', nickname: user.nickname || '' })
watch(
  () => [user.username, user.nickname],
  ([u, n]) => {
    if (u && profile.username !== u) profile.username = u
    profile.nickname = n || ''
  }
)
const profileErr = ref('')
const profileLoading = ref(false)
const avatarInput = ref(null)
const avatarPreview = computed(() => avatarUrl() || '/logo.jpg')

async function saveProfile() {
  profileErr.value = ''
  profileLoading.value = true
  try {
    const r = await api('/profile', {
      method: 'POST',
      json: { username: profile.username.trim(), nickname: profile.nickname.trim() || null },
    })
    if (r.token) setToken(r.token)
    applyUser(r)
    profile.username = user.username
    profile.nickname = user.nickname || ''
    toastOk(t('settings.toastProfileSaved'))
  } catch (e) {
    profileErr.value = e.message
    toastErr(e.message)
  } finally {
    profileLoading.value = false
  }
}

async function uploadAvatar(ev) {
  const file = ev.target.files?.[0]
  ev.target.value = ''
  if (!file) return
  if (file.size > 2 * 1024 * 1024) {
    toastErr(t('settings.avatarNote'))
    return
  }
  const reader = new FileReader()
  reader.onload = async () => {
    try {
      const data = String(reader.result).split(',')[1]
      const r = await api('/avatar', { method: 'POST', json: { data } })
      user.avatar = r.avatar
      toastOk(t('settings.toastAvatarSaved'))
    } catch (e) {
      toastErr(e.message)
    }
  }
  reader.readAsDataURL(file)
}

// ---------- 修改密码 ----------
const pw = reactive({ old: '', neww: '', confirm: '' })
const pwErr = ref('')
const pwLoading = ref(false)

async function changePw() {
  pwErr.value = ''
  if (!pw.old || !pw.neww) {
    pwErr.value = t('settings.pwdFillAll')
    return
  }
  if (pw.neww.length < 6) {
    pwErr.value = t('settings.pwdMinLen')
    return
  }
  if (pw.neww !== pw.confirm) {
    pwErr.value = t('settings.pwdNotMatch')
    return
  }
  pwLoading.value = true
  try {
    await api('/password', { method: 'POST', json: { old_password: pw.old, new_password: pw.neww } })
    user.mustChangePassword = false
    toastOk(t('settings.toastPwdChanged'))
    pw.old = pw.neww = pw.confirm = ''
  } catch (e) {
    pwErr.value = e.message
    toastErr(e.message)
  } finally {
    pwLoading.value = false
  }
}

// ---------- 双因素验证 ----------
const totpSetup = reactive({ password: '', uri: '', secret: '', code: '' })
const totpQr = ref('')
const totpBusy = ref(false)
const totpErr = ref('')
const disableOpen = ref(false)
const disableForm = reactive({ password: '', code: '' })

function totpReset() {
  totpSetup.password = ''
  totpSetup.uri = ''
  totpSetup.secret = ''
  totpSetup.code = ''
  totpQr.value = ''
  totpErr.value = ''
}

async function totpGetKey() {
  totpErr.value = ''
  if (!totpSetup.password) {
    totpErr.value = t('settings.pwdFillAll')
    return
  }
  totpBusy.value = true
  try {
    const r = await api('/totp/setup', { method: 'POST', json: { password: totpSetup.password } })
    totpSetup.uri = r.uri
    totpSetup.secret = r.secret
    totpQr.value = await QRCode.toDataURL(r.uri, { width: 280, margin: 1 })
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

async function totpEnable() {
  totpErr.value = ''
  if (!totpSetup.code) {
    totpErr.value = t('login.errTotpFill')
    return
  }
  totpBusy.value = true
  try {
    await api('/totp/enable', { method: 'POST', json: { code: totpSetup.code } })
    user.totpEnabled = true
    totpReset()
    toastOk(t('settings.toastTotpEnabled'))
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

async function totpDisable() {
  totpErr.value = ''
  if (!disableForm.password || !disableForm.code) {
    totpErr.value = t('settings.pwdFillAll')
    return
  }
  totpBusy.value = true
  try {
    await api('/totp/disable', { method: 'POST', json: { password: disableForm.password, code: disableForm.code } })
    user.totpEnabled = false
    disableOpen.value = false
    disableForm.password = ''
    disableForm.code = ''
    toastOk(t('settings.toastTotpDisabled'))
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

// ---------- 镜像加速 ----------
const mirrorsText = ref('')
const mirrorPath = ref('')
const savingMirrors = ref(false)
const mirrorMsg = ref('')
const mirrorOk = ref(false)

async function loadMirrors() {
  try {
    const r = await getRegistryMirrors()
    mirrorsText.value = (r.mirrors || []).join('\n')
    mirrorPath.value = r.path || ''
  } catch { /* 静默 */ }
}

async function saveMirrors() {
  const mirrors = mirrorsText.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  savingMirrors.value = true
  mirrorMsg.value = ''
  try {
    await saveRegistryMirrors(mirrors)
    mirrorOk.value = true
    mirrorMsg.value = t('settings.mirrorSaved')
    // 保存成功后自动重启 Docker 使配置生效(面板运行在 Docker 中,会短暂断连后自动恢复)
    try {
      await api('/system/restart-docker', { method: 'POST' })
      mirrorMsg.value = t('settings.mirrorRestarted')
    } catch (e) {
      mirrorOk.value = false
      mirrorMsg.value = t('settings.mirrorRestartFailed') + ' ' + e.message
    }
  } catch (e) {
    mirrorOk.value = false
    mirrorMsg.value = e.message
  } finally {
    savingMirrors.value = false
  }
}

// ---------- 许可证 ----------
const licKey = ref('')
const licInfo = ref(null)
const licActive = ref(false)
const licDeviceId = ref('')
const licBusy = ref(false)
const licErr = ref('')
// 文件上传激活(1Panel 添加许可证)
const licFormOpen = ref(false)
const licFile = ref(null)
const licFileName = ref('')
const licDragging = ref(false)
const licFileInput = ref(null)

/** 每次打开弹窗都是全新状态(清空上次遗留文件) */
function openLicForm() {
  licErr.value = ''
  licFile.value = null
  licFileName.value = ''
  licFormOpen.value = true
}

function resetLicForm() {
  licFile.value = null
  licFileName.value = ''
  licErr.value = ''
}

async function loadLicense() {
  try {
    const r = await getLicense()
    licActive.value = !!r.active
    licInfo.value = r.info
    licKey.value = r.key || ''
    licDeviceId.value = r.device_id || ''
  } catch { /* 静默 */ }
}

async function activate() {
  licErr.value = ''
  if (!licKey.value.trim()) {
    licErr.value = t('license.keyEmpty')
    return
  }
  licBusy.value = true
  try {
    const r = await activateLicense(licKey.value.trim())
    licActive.value = true
    licInfo.value = r.info
    licDeviceId.value = r.device_id || ''
    refreshLicense() // 同步全局(页脚/菜单徽章)
    toastOk(t('license.activated'))
  } catch (e) {
    licErr.value = e.message
  } finally {
    licBusy.value = false
  }
}

function onLicFile(e) {
  const f = e.target.files?.[0]
  if (f) {
    licFile.value = f
    licFileName.value = f.name
  }
  e.target.value = ''
}

function onLicDrop(e) {
  licDragging.value = false
  const f = e.dataTransfer?.files?.[0]
  if (f) {
    licFile.value = f
    licFileName.value = f.name
  }
}

/** 上传许可文件并授权(绑定到本机) */
async function authorizeFile() {
  if (!licFile.value) return
  licErr.value = ''
  licBusy.value = true
  try {
    const r = await activateLicenseFile(licFile.value)
    licActive.value = true
    licInfo.value = r.info
    licDeviceId.value = r.device_id || ''
    licFormOpen.value = false
    licFile.value = null
    licFileName.value = ''
    refreshLicense() // 同步全局(页脚/菜单徽章)
    toastOk(t('license.activated'))
  } catch (e) {
    licErr.value = e.message
  } finally {
    licBusy.value = false
  }
}

async function deactivate() {
  licBusy.value = true
  licErr.value = ''
  try {
    await deactivateLicense()
    licActive.value = false
    licInfo.value = null
    licKey.value = ''
    refreshLicense() // 同步全局
    toastOk(t('license.deactivated'))
  } catch (e) {
    licErr.value = e.message
  } finally {
    licBusy.value = false
  }
}

onMounted(() => {
  loadMirrors()
  loadLicense()
})
</script>
