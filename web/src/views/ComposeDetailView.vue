<template>
  <div class="space-y-4 fade-up" v-if="data">
    <!-- 头部 -->
    <Card class="px-5 py-4">
      <div class="flex items-center gap-3 flex-wrap">
        <Button variant="ghost" size="sm" @click="$router.push('/compose')"><Icon name="x" size="13" /> {{ t('common.back') }}</Button>
        <h2 class="text-base font-semibold font-mono">{{ project }}</h2>
        <StatusBadge :state="status" />
        <Badge v-if="data.managed" variant="success">
          {{ t('composeDetail.managedBadge') }}
        </Badge>
        <Badge v-else variant="warning">
          {{ t('composeDetail.notManagedBadge') }}
        </Badge>
        <div class="ml-auto flex items-center gap-1.5 flex-wrap">
          <Button v-if="status !== 'running'" variant="ok" size="sm" @click="act('start')"><Icon name="play" size="13" /> {{ t('common.start') }}</Button>
          <Button v-if="status === 'running'" variant="ghost" size="sm" @click="act('stop')"><Icon name="stop" size="13" /> {{ t('common.stop') }}</Button>
          <Button variant="ghost" size="sm" @click="act('restart')"><Icon name="restart" size="13" /> {{ t('common.restart') }}</Button>
          <Button variant="ghost" size="sm" @click="down(false)"><Icon name="x" size="13" /> {{ t('composeDetail.down') }}</Button>
          <Button variant="destructive" size="sm" @click="remove"><Icon name="trash" size="13" /> {{ t('common.delete') }}</Button>
        </div>
      </div>
    </Card>

    <!-- 标签页 -->
    <div class="flex gap-1 border-b border-line">
      <button
        v-for="tabItem in tabs"
        :key="tabItem.key"
        class="px-4 py-2.5 text-[13px] font-medium rounded-t-lg transition-colors -mb-px border-b-2 cursor-pointer"
        :class="tab === tabItem.key ? 'text-brand border-brand' : 'text-muted hover:text-text border-transparent'"
        @click="tab = tabItem.key"
      >
        {{ t(tabItem.labelKey) }}
      </button>
    </div>

    <!-- 编排文件 -->
    <Card v-if="tab === 'file'" class="p-5">
      <div class="flex items-center justify-between mb-3">
        <span class="text-sm font-semibold">docker-compose.yml</span>
        <div class="flex items-center gap-2">
          <Button variant="ghost" size="sm" @click="formatYaml"><Icon name="refresh" size="12" /> {{ t('composeDetail.reload') }}</Button>
          <Button variant="brand" size="sm" :disabled="!editable || saving" @click="save">
            <Icon name="check" size="13" /> {{ t('composeDetail.saveDeploy') }}
          </Button>
        </div>
      </div>
      <Textarea v-model="yamlText" rows="20" spellcheck="false" :disabled="!editable" />
      <div v-if="!editable" class="mt-3 rounded-lg border border-line bg-surface2/50 p-3">
        <p class="text-xs text-muted mb-2">{{ t('composeDetail.adoptDesc') }}</p>
        <Button variant="brand" size="sm" @click="adoptOpen = true"><Icon name="download" size="12" /> {{ t('composeDetail.adopt') }}</Button>
      </div>
      <p v-if="!editable" class="text-xs text-muted mt-2">{{ t('composeDetail.notEditable') }}</p>
      <!-- 保存部署过程实时输出 -->
      <div v-if="saving || output" class="mt-3 code-panel border border-line rounded-lg p-3 max-h-52 overflow-y-auto font-mono text-[11px] whitespace-pre-wrap" :class="saveFailed ? 'text-danger' : 'text-muted'">
        <template v-if="saving">
          <div class="flex items-center gap-2 mb-1.5 text-brand">
            <span class="inline-block w-3 h-3 border-2 border-brand/30 border-t-brand rounded-full animate-spin" />
            {{ t('compose.deploying') }}
          </div>
        </template>
        <div v-for="(l, i) in outputLines" :key="i" class="leading-relaxed break-all">{{ l }}</div>
        <div v-if="saveFailed" class="text-danger font-semibold pt-1">{{ t('compose.deployFailed') }}</div>
      </div>
    </Card>

    <!-- 容器 -->
    <Card v-else-if="tab === 'containers'" class="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('composeDetail.thService') }}</TableHead>
            <TableHead>{{ t('composeDetail.thContainer') }}</TableHead>
            <TableHead>{{ t('composeDetail.thImage') }}</TableHead>
            <TableHead>{{ t('composeDetail.thStatus') }}</TableHead>
            <TableHead>{{ t('composeDetail.thPorts') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="c in data.containers" :key="c.Id" class="cursor-pointer" @click="$router.push('/containers/' + c.Id)">
            <TableCell class="font-medium">{{ c.Labels?.['com.docker.compose.service'] || '-' }}</TableCell>
            <TableCell>{{ name(c) }}</TableCell>
            <TableCell class="text-muted">{{ c.Image }}</TableCell>
            <TableCell><StatusBadge :state="c.State" /></TableCell>
            <TableCell class="text-muted text-[12px]">{{ ports(c) }}</TableCell>
          </TableRow>
          <TableRow v-if="!data.containers.length">
            <TableCell colspan="5" class="text-center text-muted py-8">{{ t('composeDetail.noContainers') }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>

    <!-- 日志 -->
    <Card v-else-if="tab === 'logs'" class="p-4">
      <LogViewer :stream="`/compose/${project}/logs`" follow />
    </Card>

    <!-- 接管外部栈 -->
    <Modal :model-value="adoptOpen" :title="t('composeDetail.adoptTitle')" @close="adoptOpen = false">
      <div class="space-y-3">
        <p class="text-xs text-muted">{{ t('composeDetail.adoptDesc') }}</p>
        <div class="rounded-lg border border-warn/40 bg-warn/10 p-2.5 text-xs text-warn leading-relaxed">
          ⚠️ {{ t('composeDetail.adoptWarn') }}
        </div>
        <Textarea v-model="adoptText" rows="14" class="text-[12px]" spellcheck="false" :placeholder="t('composeDetail.adoptYamlPh')" />
        <p v-if="adoptErr" class="text-xs text-danger">{{ adoptErr }}</p>
      </div>
      <template #footer>
        <Button variant="ghost" size="sm" @click="adoptOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="brand" size="sm" :disabled="adopting || !adoptText.trim()" @click="adopt">
          <span v-if="adopting" class="inline-block w-3 h-3 border-2 border-white/40 border-t-white rounded-full animate-spin mr-1.5" />
          {{ t('composeDetail.adopt') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import StatusBadge from '../components/StatusBadge.vue'
import LogViewer from '../components/LogViewer.vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Card } from '@/components/ui/card'
import { api, composeStream } from '../api'
import { containerName, humanPorts } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const route = useRoute()
const project = computed(() => route.params.project)
const confirm = useConfirm()

const data = ref(null)
const yamlText = ref('')
const tab = ref('file')
const saving = ref(false)
const saveFailed = ref(false)
const outputLines = ref([])
const adoptOpen = ref(false)
const adoptText = ref('')
const adopting = ref(false)
const adoptErr = ref('')
let timer = null

const tabs = [
  { key: 'file', labelKey: 'composeDetail.tabFile' },
  { key: 'containers', labelKey: 'composeDetail.tabContainers' },
  { key: 'logs', labelKey: 'composeDetail.tabLogs' },
]

const editable = computed(() => !!data.value?.yaml)
const status = computed(() => {
  const cs = data.value?.containers || []
  if (!cs.length) return 'stopped'
  const running = cs.filter((c) => c.State === 'running').length
  if (running === 0) return 'stopped'
  if (running === cs.length) return 'running'
  return 'partial'
})
const name = (c) => containerName(c)
const ports = (c) => humanPorts(c.Ports)

async function load() {
  try {
    data.value = await api(`/compose/${project.value}`)
    yamlText.value = data.value.yaml || ''
  } catch (e) {
    toastErr(e.message)
  }
}

async function act(action) {
  try {
    await api(`/compose/${project.value}/${action}`, { method: 'POST' })
    toastOk({ start: t('compose.toastStarted'), stop: t('compose.toastStopped'), restart: t('compose.toastRestarted') }[action])
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

// 接管外部创建的栈:粘贴 yaml 保存到面板 → 变为可管理
async function adopt() {
  adopting.value = true
  adoptErr.value = ''
  try {
    await api(`/compose/${project.value}/adopt`, { method: 'POST', json: { yaml: adoptText.value } })
    toastOk(t('composeDetail.adopted'))
    adoptOpen.value = false
    load()
  } catch (e) {
    adoptErr.value = e.message
  } finally {
    adopting.value = false
  }
}

async function save() {
  saving.value = true
  saveFailed.value = false
  outputLines.value = []
  try {
    await composeStream(`/compose/${project.value}`, { project: project.value, yaml: yamlText.value }, (line) => {
      outputLines.value.push(line)
    })
    toastOk(t('composeDetail.toastDeployOk'))
    load()
  } catch (e) {
    saveFailed.value = true
    outputLines.value.push(`❌ ${e.message}`)
    toastErr(e.message)
  } finally {
    saving.value = false
  }
}

async function down(volumes) {
  const ok = await confirm(t('composeDetail.confirmDown', { project: project.value }), {
    title: t('composeDetail.downTitle'),
    confirmText: t('composeDetail.down'),
  })
  if (!ok) return
  try {
    await api(`/compose/${project.value}/down${volumes ? '?volumes=true' : ''}`, { method: 'POST' })
    toastOk(t('composeDetail.toastDownOk'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

async function remove() {
  const ok = await confirm(t('composeDetail.confirmRemove', { project: project.value }), {
    title: t('composeDetail.removeTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/compose/${project.value}`, { method: 'DELETE' })
    toastOk(t('composeDetail.toastRemoved'))
    window.location.href = '/compose'
  } catch (e) {
    toastErr(e.message)
  }
}

function formatYaml() {
  load()
}

onMounted(() => {
  load()
  timer = setInterval(load, 8000)
})
onBeforeUnmount(() => clearInterval(timer))
watch(project, () => {
  tab.value = 'file'
  load()
})
</script>
