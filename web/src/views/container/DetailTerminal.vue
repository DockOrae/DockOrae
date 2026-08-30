<template>
  <div class="flex flex-col h-[520px]">
    <div class="flex items-center gap-2 mb-2 flex-wrap">
      <Select v-model="shell" :disabled="connected" class="!w-32">
        <SelectTrigger class="!w-32 !h-8 !text-xs !py-1"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="/bin/sh">/bin/sh</SelectItem>
          <SelectItem value="/bin/bash">/bin/bash</SelectItem>
          <SelectItem value="/bin/ash">/bin/ash</SelectItem>
          <SelectItem value="/bin/zsh">/bin/zsh</SelectItem>
          <SelectItem value="custom">{{ t('terminal.custom') }}</SelectItem>
        </SelectContent>
      </Select>
      <Input v-if="shell === 'custom'" v-model="customShell" class="!w-40 !h-8 !text-xs !py-1" placeholder="/bin/busybox sh" />
      <Button size="sm" :variant="connected ? 'destructive' : 'brand'" @click="toggle">
        {{ connected ? t('terminal.disconnect') : t('terminal.connect') }}
      </Button>
      <Button variant="ghost" size="sm" @click="clearScreen" :disabled="!term">
        <Icon name="x" size="13" /> {{ t('terminal.clear') }}
      </Button>
      <div class="flex items-center gap-1 rounded-md border border-line px-1 py-0.5">
        <Button variant="icon" :disabled="!hasSelection" :title="t('terminal.copy')" @click="copySel"><Icon name="copy" size="12" /></Button>
        <Button variant="icon" :disabled="!connected" :title="t('terminal.paste')" @click="pasteText"><Icon name="clipboard" size="12" /></Button>
        <span class="w-px h-3 bg-line mx-0.5" />
        <Button variant="icon" :title="t('terminal.fontDec')" @click="fontSize--"><Icon name="minus" size="12" /></Button>
        <span class="text-[11px] text-muted w-6 text-center">{{ fontSize }}</span>
        <Button variant="icon" :title="t('terminal.fontInc')" @click="fontSize++"><Icon name="plus" size="12" /></Button>
      </div>
      <Button variant="ghost" size="sm" :title="t('terminal.theme')" @click="dark = !dark">
        <Icon :name="dark ? 'sun' : 'moon'" size="13" />
      </Button>
      <span v-if="error" class="text-xs text-danger">{{ error }}</span>
    </div>
    <div ref="termEl" class="flex-1 bg-[#0a0d13] border border-line rounded-lg overflow-hidden p-2" />
    <p class="text-[11px] text-muted mt-2">
      {{ t('terminal.hint') }}
    </p>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import Icon from '../../components/Icon.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { wsUrl } from '../../api'
import { toastErr } from '../../toast'

const { t } = useI18n()
const props = defineProps({ id: { type: String, required: true } })

const termEl = ref(null)
const shell = ref('/bin/sh')
const customShell = ref('/bin/sh')
const connected = ref(false)
const error = ref('')
const hasSelection = ref(false)
const fontSize = ref(13)
const dark = ref(true)
let term = null
let fit = null
let ws = null

function currentShell() {
  return shell.value === 'custom' ? customShell.value || '/bin/sh' : shell.value
}

// 复制终端选中内容
function copySel() {
  const sel = term?.getSelection()
  if (!sel) return
  navigator.clipboard?.writeText(sel).catch(() => {})
}

// 粘贴剪贴板文本到终端
async function pasteText() {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  try {
    const text = await navigator.clipboard.readText()
    if (text) ws.send(text)
  } catch { /* 剪贴板权限拒绝时静默 */ }
}

function connect() {
  disconnect()
  error.value = ''
  try {
    ws = new WebSocket(wsUrl(`/containers/${props.id}/terminal?shell=${encodeURIComponent(currentShell())}`))
  } catch (e) {
    error.value = e.message
    return
  }
  ws.onopen = () => {
    connected.value = true
    sendResize()
  }
  ws.onmessage = (ev) => {
    if (term && typeof ev.data === 'object') {
      term.write(new Uint8Array(ev.data))
    } else if (term && typeof ev.data === 'string') {
      term.write(ev.data)
    }
  }
  ws.onclose = () => {
    connected.value = false
    ws = null
    if (term) term.write(`\r\n\x1b[31m[${t('terminal.disconnected')}]\x1b[0m\r\n`)
  }
  ws.onerror = () => {
    error.value = t('terminal.cantConnect')
    ws?.close()
  }
}

function disconnect() {
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  connected.value = false
}

function sendResize() {
  if (ws && ws.readyState === WebSocket.OPEN && fit) {
    const { cols, rows } = fit.proposeDimensions() || { cols: 80, rows: 24 }
    ws.send(`resize:${cols},${rows}`)
  }
}

function toggle() {
  if (connected.value) disconnect()
  else connect()
}

function clearScreen() {
  term?.clear()
}

onMounted(() => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: fontSize.value,
    fontFamily: 'Consolas, "Cascadia Code", monospace',
    theme: { background: '#0a0d13', foreground: '#d4d4d4', cursor: '#ec4899' },
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(termEl.value)
  fit.fit()
  term.onData((d) => ws?.send(d))
  term.onSelectionChange(() => {
    hasSelection.value = !!term?.getSelection()
  })
  window.addEventListener('resize', onResize)
})

// 字号调节
watch(fontSize, (n) => {
  if (n < 9) fontSize.value = 9
  if (n > 24) fontSize.value = 24
  if (term) term.options.fontSize = fontSize.value
})

// 深浅主题切换
watch(dark, (d) => {
  if (!term) return
  term.options.theme = d
    ? { background: '#0a0d13', foreground: '#d4d4d4', cursor: '#ec4899' }
    : { background: '#ffffff', foreground: '#1f2937', cursor: '#ec4899' }
})

function onResize() {
  if (connected.value) {
    fit?.fit()
    sendResize()
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  disconnect()
  term?.dispose()
})
</script>
