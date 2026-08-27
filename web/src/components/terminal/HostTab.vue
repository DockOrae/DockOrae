<template>
  <div class="space-y-3">
    <!-- 工具栏 -->
    <div class="card p-3 flex items-center gap-2 flex-wrap">
      <button class="btn btn-primary btn-sm" @click="openForm(null)">
        <Icon name="plus" size="13" /> {{ t('host.addHost') }}
      </button>
      <button class="btn btn-ghost btn-sm" @click="load">
        <Icon name="refresh" size="13" /> {{ t('host.refresh') }}
      </button>
      <select v-model="groupFilter" class="input !w-auto !py-1.5 !text-[12px] cursor-pointer ml-auto" @change="load">
        <option value="">{{ t('host.allGroups') }}</option>
        <option v-for="g in groups" :key="g" :value="g">{{ g === 'Default' ? t('host.defaultGroup') : g }}</option>
      </select>
    </div>

    <!-- 主机列表 -->
    <div class="card overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th class="th">{{ t('host.name') }}</th>
            <th class="th">{{ t('host.addr') }}</th>
            <th class="th">{{ t('host.user') }}</th>
            <th class="th">{{ t('host.port') }}</th>
            <th class="th">{{ t('host.group') }}</th>
            <th class="th">{{ t('host.authMode') }}</th>
            <th class="th w-36">{{ t('containers.thActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="h in filteredHosts" :key="h.id">
            <td class="td font-medium">{{ h.name }}</td>
            <td class="td text-muted font-mono text-[12px]">{{ h.addr }}</td>
            <td class="td text-muted">{{ h.user }}</td>
            <td class="td text-muted">{{ h.port }}</td>
            <td class="td">
              <span class="badge" :style="h.group === 'Default' ? mutedStyle : okStyle">{{ h.group === 'Default' ? t('host.defaultGroup') : h.group }}</span>
            </td>
            <td class="td text-muted">{{ h.auth_mode === 'key' ? t('host.keyMode') : t('host.passwordMode') }}</td>
            <td class="td">
              <div class="flex items-center gap-1">
                <button class="btn btn-icon btn-sm btn-primary" :title="t('host.connect')" @click="connectHost(h)">
                  <Icon name="terminal" size="13" />
                </button>
                <button class="btn btn-icon btn-sm" :title="t('host.test')" @click="testHost(h)">
                  <Icon name="refresh" size="13" />
                </button>
                <button class="btn btn-icon btn-sm" :title="t('common.edit')" @click="openForm(h)">
                  <Icon name="settings" size="13" />
                </button>
                <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="delHost(h)">
                  <Icon name="trash" size="13" />
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="!filteredHosts.length">
            <td colspan="7" class="td text-center text-muted py-8">{{ t('host.empty') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 添加/编辑弹窗 -->
    <div v-if="formOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4" @click.self="formOpen = false">
      <div class="card p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto fade-up">
        <h3 class="text-sm font-semibold mb-4">{{ editing ? t('host.editHost') : t('host.addHost') }}</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="label">{{ t('host.name') }}</label>
            <input v-model="form.name" class="input" :placeholder="t('host.namePh')" />
          </div>
          <div>
            <label class="label">{{ t('host.addr') }} *</label>
            <input v-model="form.addr" class="input" placeholder="192.168.1.100" />
          </div>
          <div>
            <label class="label">{{ t('host.port') }}</label>
            <input v-model.number="form.port" type="number" min="1" max="65535" class="input" />
          </div>
          <div>
            <label class="label">{{ t('host.user') }}</label>
            <input v-model="form.user" class="input" placeholder="root" />
          </div>
          <div class="sm:col-span-2">
            <label class="label">{{ t('host.authMode') }}</label>
            <div class="flex gap-2">
              <button
                type="button"
                class="flex-1 px-3 py-2 rounded-lg text-[13px] font-medium border-2 transition-all"
                :class="form.auth_mode === 'password' ? 'text-brand border-brand bg-surface' : 'text-muted border-transparent bg-surface2/50 hover:text-text'"
                @click="form.auth_mode = 'password'"
              >
                {{ t('host.passwordMode') }}
              </button>
              <button
                type="button"
                class="flex-1 px-3 py-2 rounded-lg text-[13px] font-medium border-2 transition-all"
                :class="form.auth_mode === 'key' ? 'text-brand border-brand bg-surface' : 'text-muted border-transparent bg-surface2/50 hover:text-text'"
                @click="form.auth_mode = 'key'"
              >
                {{ t('host.keyMode') }}
              </button>
            </div>
          </div>
          <div v-if="form.auth_mode === 'password'" class="sm:col-span-2">
            <label class="label">{{ t('host.password') }}</label>
            <input v-model="form.password" type="password" class="input" :placeholder="t('host.passwordPh')" autocomplete="new-password" />
          </div>
          <div v-else class="sm:col-span-2">
            <label class="label">{{ t('host.privateKey') }}</label>
            <textarea v-model="form.private_key" class="input !h-24 code-panel font-mono !text-[11px]" :placeholder="t('host.keyPh')" spellcheck="false" />
          </div>
          <div>
            <label class="label">{{ t('host.group') }}</label>
            <input v-model="form.group" class="input" :placeholder="t('host.defaultGroup')" />
          </div>
          <div>
            <label class="label">{{ t('host.description') }}</label>
            <input v-model="form.description" class="input" />
          </div>
        </div>
        <div v-if="formErr" class="text-xs text-danger mt-3">{{ formErr }}</div>
        <div class="flex justify-end gap-2 mt-5">
          <button class="btn btn-ghost btn-sm" @click="formOpen = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" :disabled="saving" @click="saveForm">
            <span v-if="saving" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
            {{ t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 连接终端弹窗 -->
    <div v-if="termOpen" class="fixed inset-0 z-50 flex flex-col bg-bg/95 backdrop-blur p-4">
      <div class="flex items-center gap-2 mb-3 shrink-0">
        <Icon name="terminal" size="15" class="text-brand" />
        <span class="text-[13px] font-semibold">{{ termHost?.name }} ({{ termHost?.user }}@{{ termHost?.addr }})</span>
        <span v-if="termConnected" class="text-[11px] text-ok flex items-center gap-1.5">
          <span class="dot" style="background: #34d399" /> {{ t('terminal.connected') }}
        </span>
        <span v-else-if="termErr" class="text-[11px] text-danger">{{ termErr }}</span>
        <button class="btn btn-icon btn-sm ml-auto" :title="t('terminal.reconnect')" @click="termConnect()">
          <Icon name="refresh" size="14" />
        </button>
        <button class="btn btn-icon btn-sm" :title="t('common.close')" @click="closeTerm">
          <Icon name="x" size="14" />
        </button>
      </div>
      <div class="flex-1 min-h-0 rounded-lg overflow-hidden border border-line bg-[#0a0d13]">
        <div ref="termEl" class="h-full w-full" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import Icon from '../Icon.vue'
import { api, wsUrl } from '../../api'
import { termSettings } from '../../store'
import { toastErr, toastOk } from '../../toast'

const { t } = useI18n()

const hosts = ref([])
const groups = ref([])
const groupFilter = ref('')
const formOpen = ref(false)
const editing = ref(null)
const saving = ref(false)
const formErr = ref('')
const form = reactive({ name: '', addr: '', port: 22, user: 'root', auth_mode: 'password', password: '', private_key: '', group: '', description: '' })

// 终端
const termOpen = ref(false)
const termHost = ref(null)
const termConnected = ref(false)
const termErr = ref('')
const termEl = ref(null)
let term = null
let fit = null
let termWs = null

const okStyle = { color: '#60a5fa', background: 'rgba(96,165,250,.12)', border: '1px solid rgba(96,165,250,.3)' }
const mutedStyle = { color: '#8b93a7', background: 'rgba(139,147,167,.12)', border: '1px solid rgba(139,147,167,.3)' }

const filteredHosts = computed(() =>
  groupFilter.value ? hosts.value.filter((h) => h.group === groupFilter.value) : hosts.value
)

async function load() {
  try {
    const r = await api('/hosts')
    hosts.value = r.hosts || []
    groups.value = r.groups || []
  } catch (e) {
    toastErr(e.message)
  }
}

function openForm(h) {
  editing.value = h || null
  form.name = h?.name || ''
  form.addr = h?.addr || ''
  form.port = h?.port || 22
  form.user = h?.user || 'root'
  form.auth_mode = h?.auth_mode || 'password'
  form.password = ''
  form.private_key = h?.private_key || ''
  form.group = h?.group || ''
  form.description = h?.description || ''
  formErr.value = ''
  formOpen.value = true
}

async function saveForm() {
  formErr.value = ''
  if (!form.addr.trim()) {
    formErr.value = t('host.addrRequired')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await api('/hosts/' + editing.value.id, { method: 'PUT', json: { ...form } })
    } else {
      await api('/hosts', { method: 'POST', json: { ...form } })
    }
    formOpen.value = false
    toastOk(t('host.saved'))
    load()
  } catch (e) {
    formErr.value = e.message
  } finally {
    saving.value = false
  }
}

async function delHost(h) {
  if (!confirm(t('host.confirmDelete') + ' ' + h.name + '?')) return
  try {
    await api('/hosts/' + h.id, { method: 'DELETE' })
    toastOk(t('host.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

async function testHost(h) {
  try {
    await api('/hosts/' + h.id + '/test', { method: 'POST' })
    toastOk(t('host.testOk'))
  } catch (e) {
    toastErr(e.message)
  }
}

function connectHost(h) {
  termHost.value = h
  termErr.value = ''
  termOpen.value = true
  nextTick(() => {
    if (!term) initTerm()
    term?.clear()
    termConnect()
  })
}

function initTerm() {
  term = new Terminal({
    fontFamily: termSettings.font_family,
    fontSize: termSettings.font_size,
    cursorBlink: termSettings.cursor_blink,
    scrollback: termSettings.scrollback,
    theme: { background: '#0a0d13', foreground: termSettings.foreground },
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(termEl.value)
  term.onData((data) => {
    if (termWs && termWs.readyState === WebSocket.OPEN) termWs.send(data)
  })
  requestAnimationFrame(() => fit?.fit())
}

function termConnect() {
  if (!termHost.value) return
  termDisconnect()
  termErr.value = ''
  try {
    termWs = new WebSocket(wsUrl(`/hosts/${termHost.value.id}/terminal`))
  } catch (e) {
    termErr.value = e.message
    return
  }
  termWs.onopen = () => {
    termConnected.value = true
    sendTermResize()
  }
  termWs.onmessage = (ev) => {
    if (typeof ev.data === 'object') term?.write(new Uint8Array(ev.data))
    else if (typeof ev.data === 'string') term?.write(ev.data)
  }
  termWs.onclose = () => {
    termConnected.value = false
    termWs = null
  }
  termWs.onerror = () => {
    termErr.value = t('terminal.cantConnect')
    termWs?.close()
  }
}

function termDisconnect() {
  if (termWs) {
    termWs.onclose = null
    termWs.close()
    termWs = null
  }
  termConnected.value = false
}

function sendTermResize() {
  if (termWs && termWs.readyState === WebSocket.OPEN && fit) {
    const { cols, rows } = fit.proposeDimensions() || { cols: 80, rows: 24 }
    termWs.send(`resize:${cols},${rows}`)
  }
}

function closeTerm() {
  termDisconnect()
  termOpen.value = false
  termHost.value = null
}

onMounted(load)
onBeforeUnmount(() => {
  termDisconnect()
  term?.dispose()
  term = null
})
</script>
