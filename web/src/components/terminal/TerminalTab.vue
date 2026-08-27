<template>
  <div class="flex flex-col gap-3">
    <!-- 会话标签栏 -->
    <div class="flex items-center gap-1 flex-wrap">
      <button
        v-for="s in sessions"
        :key="s.id"
        class="group flex items-center gap-2 px-3.5 h-9 rounded-t-lg border border-b-0 text-[12px] font-medium transition-colors"
        :class="s.id === activeId ? 'bg-bg text-brand border-line' : 'bg-surface2/60 text-muted hover:text-text border-transparent'"
        @click="switchTo(s.id)"
      >
        <span class="dot" :style="{ background: s.connected ? '#34d399' : '#8b93a7' }" />
        <span class="max-w-[140px] truncate">{{ s.name }}</span>
        <Icon name="x" size="12" class="opacity-0 group-hover:opacity-100 hover:text-danger transition-opacity shrink-0" @click.stop="closeSession(s.id)" />
      </button>
      <button
        class="flex items-center justify-center w-9 h-9 rounded-lg border border-dashed border-line text-muted hover:text-brand hover:border-brand transition-colors"
        :title="t('terminal.newSession')"
        @click="openNew = true"
      >
        <Icon name="plus" size="14" />
      </button>
    </div>

    <!-- 会话内容(每个会话一个 xterm,v-show 切换) -->
    <div class="relative rounded-lg overflow-hidden border border-line" :style="{ height: termHeight + 'px' }">
      <div
        v-for="s in sessions"
        :key="s.id"
        :ref="(el) => setTermEl(s.id, el)"
        class="absolute inset-0"
        :class="s.id === activeId ? '' : 'hidden'"
      />
      <div v-if="!sessions.length" class="absolute inset-0 flex items-center justify-center text-muted text-[13px]">
        {{ t('terminal.newSessionHint') }}
      </div>
    </div>

    <!-- 底部工具条:快速命令 + 发送命令 -->
    <div class="flex items-center gap-2 flex-wrap">
      <select v-model="activeCmd" class="input !w-auto !py-1.5 !text-[12px] cursor-pointer">
        <option value="">{{ t('terminal.quickCommand') }}</option>
        <option v-for="c in commands" :key="c.id" :value="c.command">{{ c.name }}</option>
      </select>
      <button class="btn btn-sm btn-ghost" :disabled="!activeCmd || !activeSession" @click="sendToActive(activeCmd)">
        {{ t('terminal.run') }}
      </button>
      <input
        v-model="sendText"
        class="input flex-1 min-w-[160px] !py-1.5 !text-[12px]"
        :placeholder="t('terminal.sendHint')"
        @keydown.enter.prevent="sendToActive(sendText)"
      />
      <button class="btn btn-sm btn-ghost" :disabled="!activeSession" @click="sendToActive(sendText)">
        {{ t('terminal.send') }}
      </button>
      <span v-if="activeSession?.connected" class="text-[11px] text-ok flex items-center gap-1.5 ml-auto">
        <span class="dot" style="background: #34d399" /> {{ t('terminal.connected') }}
      </span>
    </div>

    <!-- 新建会话弹窗 -->
    <div v-if="openNew" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm" @click.self="openNew = false">
      <div class="card p-6 w-[380px] fade-up">
        <h3 class="text-sm font-semibold mb-4">{{ t('terminal.newSession') }}</h3>
        <label class="label">{{ t('terminal.sessionType') }}</label>
        <div class="flex gap-2 mb-3">
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-[13px] font-medium border-2 transition-all"
            :class="newForm.type === 'host' ? 'text-brand border-brand bg-surface' : 'text-muted border-transparent bg-surface2/50 hover:text-text'"
            @click="newForm.type = 'host'"
          >
            <Icon name="terminal" size="13" class="mr-1 inline" /> {{ t('terminal.host') }}
          </button>
          <button
            type="button"
            class="flex-1 px-3 py-2 rounded-lg text-[13px] font-medium border-2 transition-all"
            :class="newForm.type === 'container' ? 'text-brand border-brand bg-surface' : 'text-muted border-transparent bg-surface2/50 hover:text-text'"
            @click="newForm.type = 'container'"
          >
            <Icon name="container" size="13" class="mr-1 inline" /> {{ t('terminal.container') }}
          </button>
        </div>
        <template v-if="newForm.type === 'container'">
          <label class="label">{{ t('terminal.selectContainer') }}</label>
          <select v-model="newForm.containerId" class="input mb-3 cursor-pointer">
            <option value="">{{ t('terminal.chooseContainer') }}</option>
            <option v-for="c in running" :key="c.Id" :value="c.Id">{{ name(c) }}</option>
          </select>
        </template>
        <label class="label">{{ t('terminal.shell') }}</label>
        <select v-model="newForm.shell" class="input mb-4 cursor-pointer">
          <option value="/bin/sh">/bin/sh</option>
          <option value="/bin/bash">/bin/bash</option>
        </select>
        <div class="flex justify-end gap-2">
          <button class="btn btn-ghost btn-sm" @click="openNew = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" :disabled="newForm.type === 'container' && !newForm.containerId" @click="createSession">
            {{ t('terminal.connect') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import Icon from '../Icon.vue'
import { api, wsUrl } from '../../api'
import { containerName } from '../../util'
import { termSettings } from '../../store'
import { toastErr } from '../../toast'

const { t } = useI18n()

const containers = ref([])
const commands = ref([])
const sessions = ref([]) // {id, type: 'host'|'container', name, containerId, shell}
const activeId = ref(null)
const openNew = ref(false)
const newForm = ref({ type: 'host', containerId: '', shell: '/bin/sh' })
const activeCmd = ref('')
const sendText = ref('')
const termEls = {} // id -> DOM
const terms = {} // id -> {term, fit, ws, connected}

const running = computed(() => containers.value.filter((c) => c.State === 'running'))
const name = (c) => containerName(c)
const activeSession = computed(() => sessions.value.find((s) => s.id === activeId.value))
const termHeight = 420

async function load() {
  try {
    const [cs, cmds] = await Promise.all([api('/containers'), api('/terminal/quick-commands')])
    containers.value = cs
    commands.value = cmds.commands || []
  } catch (e) {
    toastErr(e.message)
  }
}

function setTermEl(id, el) {
  if (el) termEls[id] = el
}

function switchTo(id) {
  activeId.value = id
  nextTick(() => {
    const t = terms[id]
    if (t && t.fit) {
      t.fit.fit()
      sendResize(id)
    }
  })
}

function createSession() {
  let s
  if (newForm.value.type === 'host') {
    s = {
      id: 's' + Date.now(),
      type: 'host',
      name: t('terminal.host'),
      containerId: null,
      shell: newForm.value.shell,
    }
  } else {
    const c = containers.value.find((x) => x.Id === newForm.value.containerId)
    if (!c) return
    s = {
      id: 's' + Date.now(),
      type: 'container',
      name: name(c),
      containerId: c.Id,
      shell: newForm.value.shell,
    }
  }
  sessions.value.push(s)
  activeId.value = s.id
  openNew.value = false
  newForm.value.containerId = ''
  nextTick(() => initSession(s))
}

/** 默认打开一个宿主机终端会话 */
function defaultHostSession() {
  const s = {
    id: 's' + Date.now(),
    type: 'host',
    name: t('terminal.host'),
    containerId: null,
    shell: termSettings.default_shell,
  }
  sessions.value.push(s)
  activeId.value = s.id
  nextTick(() => initSession(s))
}

function initSession(s) {
  const el = termEls[s.id]
  if (!el) return
  const term = new Terminal({
    fontFamily: termSettings.font_family,
    fontSize: termSettings.font_size,
    cursorBlink: termSettings.cursor_blink,
    scrollback: termSettings.scrollback,
    theme: { background: termSettings.background, foreground: termSettings.foreground },
  })
  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(el)
  fit.fit()
  const wsState = { connected: false }
  const state = { term, fit, ws: null, connected: wsState }
  terms[s.id] = state

  term.onData((data) => {
    const ws = state.ws
    if (ws && ws.readyState === WebSocket.OPEN) ws.send(data)
  })

  connect(s, state)
}

function connect(s, state) {
  disconnect(state)
  const url =
    s.type === 'host'
      ? wsUrl(`/terminal/self/ws?shell=${encodeURIComponent(s.shell)}`)
      : wsUrl(`/containers/${s.containerId}/terminal?shell=${encodeURIComponent(s.shell)}`)
  try {
    const ws = new WebSocket(url)
    state.ws = ws
    ws.onopen = () => {
      state.connected.connected = true
      sendResize(s.id)
    }
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'object') state.term.write(new Uint8Array(ev.data))
      else if (typeof ev.data === 'string') state.term.write(ev.data)
    }
    ws.onclose = () => {
      state.connected.connected = false
      state.ws = null
    }
    ws.onerror = () => {
      state.connected.connected = false
      ws.close()
    }
  } catch (e) {
    toastErr(e.message)
  }
}

function disconnect(state) {
  if (state.ws) {
    state.ws.onclose = null
    state.ws.close()
    state.ws = null
  }
  state.connected.connected = false
}

function sendResize(id) {
  const state = terms[id]
  if (!state || !state.ws || state.ws.readyState !== WebSocket.OPEN || !state.fit) return
  const { cols, rows } = state.fit.proposeDimensions() || { cols: 80, rows: 24 }
  state.ws.send(`resize:${cols},${rows}`)
}

function closeSession(id) {
  const state = terms[id]
  if (state) {
    disconnect(state)
    state.term?.dispose()
    delete terms[id]
  }
  const idx = sessions.value.findIndex((s) => s.id === id)
  if (idx >= 0) sessions.value.splice(idx, 1)
  delete termEls[id]
  if (activeId.value === id) activeId.value = sessions.value[sessions.value.length - 1]?.id || null
}

function sendToActive(text) {
  if (!text) return
  const s = activeSession.value
  if (!s) return
  const state = terms[s.id]
  if (state && state.ws && state.ws.readyState === WebSocket.OPEN) {
    state.ws.send(text + '\r')
    if (text === sendText.value) sendText.value = ''
    if (text === activeCmd.value) activeCmd.value = ''
  } else {
    toastErr(t('terminal.notConnected'))
  }
}

function fitAll() {
  for (const id of Object.keys(terms)) {
    terms[id].fit?.fit()
    sendResize(id)
  }
}

let timer = null
onMounted(() => {
  load()
  defaultHostSession()
  timer = setInterval(load, 15000)
  window.addEventListener('resize', fitAll)
})
onBeforeUnmount(() => {
  clearInterval(timer)
  window.removeEventListener('resize', fitAll)
  for (const id of Object.keys(terms)) {
    const state = terms[id]
    disconnect(state)
    state.term?.dispose()
  }
})
</script>
