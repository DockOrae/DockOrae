<template>
  <div class="space-y-4 fade-up">
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <Input v-model="keyword" class="!w-64 !pl-9" :placeholder="t('images.searchPh')" />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="sm" @click="prune"><Icon name="trash" size="13" /> {{ t('images.prune') }}</Button>
        <Button variant="brand" size="sm" @click="pullOpen = true"><Icon name="download" size="13" /> {{ t('images.pullImage') }}</Button>
      </div>
    </div>

    <Card class="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('images.thImage') }}</TableHead>
            <TableHead>{{ t('images.thId') }}</TableHead>
            <TableHead>{{ t('images.thSize') }}</TableHead>
            <TableHead>{{ t('images.thCreated') }}</TableHead>
            <TableHead class="w-28">{{ t('images.thActions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="img in filtered" :key="img.Id">
            <TableCell class="font-medium">{{ tag(img) }}</TableCell>
            <TableCell class="font-mono text-[12px] text-muted">{{ shortId(img.Id) }}</TableCell>
            <TableCell class="text-muted">{{ fmt(img.Size) }}</TableCell>
            <TableCell class="text-muted text-[12px]">{{ formatDate(img.Created) }}</TableCell>
            <TableCell>
              <div class="flex items-center gap-1">
                <Button variant="icon" :title="t('images.detail')" @click="showDetail(img)">
                  <Icon name="eye" size="13" />
                </Button>
                <Button variant="icon" class="text-danger" :title="t('common.delete')" @click="remove(img)">
                  <Icon name="trash" size="13" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
          <TableRow v-if="!filtered.length">
            <TableCell colspan="5" class="text-center text-muted py-10">{{ t('images.noImages') }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>

    <!-- 拉取镜像 -->
    <Modal :model-value="pullOpen" :title="t('images.pullTitle')" @close="closePull">
      <div class="space-y-3">
        <div>
          <Label>{{ t('images.imageName') }}</Label>
          <Input v-model="pullForm.from_image" :placeholder="t('images.imageNamePh')" />
        </div>
        <div>
          <Label>{{ t('images.tag') }}</Label>
          <Input v-model="pullForm.tag" class="!w-40" :placeholder="t('images.tagPh')" />
        </div>
        <div v-if="pulling || overall || layerList.length" class="rounded-lg border border-line p-3">
          <!-- 整体进度条 -->
          <div class="flex items-center gap-2 mb-2">
            <div class="flex-1 h-1.5 rounded-full bg-surface2 overflow-hidden">
              <div class="h-full bg-brand transition-all duration-300" :style="{ width: overallPct + '%' }" />
            </div>
            <span class="text-[11px] text-muted font-mono w-10 text-right">{{ overallPct }}%</span>
          </div>
          <div v-if="overall" class="text-[11px] text-muted mb-2 break-all">{{ overall }}</div>
          <div v-if="pullError" class="text-[11px] text-danger mb-2 break-all">{{ pullError }}</div>
          <!-- 层列表 -->
          <div class="max-h-44 overflow-y-auto space-y-1">
            <div v-for="l in layerList" :key="l.id" class="flex items-center gap-2 text-[11px] font-mono">
              <span :class="l.status === 'Pull complete' ? 'text-ok' : 'text-muted'">
                {{ l.status === 'Pull complete' ? '✓' : (l.status.includes('Downloading') || l.status.includes('Extracting') || l.status.includes('Verifying') ? '⟳' : '○') }}
              </span>
              <span class="text-muted w-16 truncate shrink-0">{{ shortId(l.id) }}</span>
              <span class="flex-1 truncate" :class="l.status === 'Pull complete' ? 'text-ok' : 'text-muted'">{{ l.status }}</span>
              <span v-if="l.progress" class="text-muted shrink-0">{{ l.progress }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" size="sm" @click="closePull">{{ t('common.close') }}</Button>
        <Button variant="brand" size="sm" :disabled="pulling || !pullForm.from_image" @click="pull">
          <span v-if="pulling" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
          {{ pulling ? t('images.pulling') : t('images.pullStart') }}
        </Button>
      </template>
    </Modal>

    <!-- 镜像详情 -->
    <Modal :model-value="detailOpen" :title="t('images.detailTitle')" @close="detailOpen = false">
      <pre class="code-panel border border-line rounded-lg p-3 text-[11px] font-mono overflow-auto max-h-96 text-muted">{{ detailJson }}</pre>
    </Modal>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, pullImageStream } from '../api'
import { shortId, formatDate, formatBytes } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const images = ref([])
const keyword = ref('')
const pullOpen = ref(false)
const pulling = ref(false)
const pullForm = ref({ from_image: '', tag: 'latest' })
const layers = ref({})
const overall = ref('')
const pullError = ref('')
const detailOpen = ref(false)
const detailJson = ref('')
const confirm = useConfirm()

// 层列表(按发现顺序)
const layerList = computed(() => Object.values(layers.value))
// 整体进度:已完成的层 / 总层数
const overallPct = computed(() => {
  const list = layerList.value
  if (!list.length) return 0
  const done = list.filter((l) => l.status === 'Pull complete').length
  return Math.round((done / list.length) * 100)
})

const filtered = computed(() => {
  if (!keyword.value) return images.value
  const k = keyword.value.toLowerCase()
  return images.value.filter((i) => (i.RepoTags || []).join(' ').toLowerCase().includes(k) || i.Id.includes(k))
})

const tag = (i) => (i.RepoTags && i.RepoTags.length ? i.RepoTags[0] : `<none>:<none>`)
const fmt = (n) => formatBytes(n, 0)

async function load() {
  images.value = await api('/images')
}

async function pull() {
  pulling.value = true
  layers.value = {}
  overall.value = ''
  pullError.value = ''
  try {
    await pullImageStream(
      { from_image: pullForm.value.from_image, tag: pullForm.value.tag || 'latest' },
      (line) => {
        const id = line.id || ''
        const status = line.status || ''
        if (line.error) {
          pullError.value = line.error
          overall.value = line.error
          return
        }
        if (id) {
          const l = layers.value[id] || { id, status: '', progress: '', pct: 0 }
          l.status = status
          if (line.progressDetail?.current != null && line.progressDetail?.total) {
            l.pct = Math.round((line.progressDetail.current / line.progressDetail.total) * 100)
            l.progress = `${formatBytes(line.progressDetail.current)} / ${formatBytes(line.progressDetail.total)}`
          } else if (line.progress) {
            l.progress = line.progress
          }
          layers.value = { ...layers.value, [id]: l }
        } else if (status) {
          // 整体状态行(Pulling from / Status: Downloaded newer image / Digest / Pull complete)
          overall.value = status
        }
      }
    )
    toastOk(t('images.toastPullDone'))
    pullOpen.value = false
    load()
  } catch (e) {
    pullError.value = e.message
    toastErr(e.message)
  } finally {
    pulling.value = false
  }
}

function closePull() {
  if (pulling.value) return
  pullOpen.value = false
  layers.value = {}
  overall.value = ''
  pullError.value = ''
}

async function remove(img) {
  const name = tag(img)
  const ok = await confirm(t('images.confirmDelete', { name }), {
    title: t('images.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/images/${img.Id}?force=true`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

async function showDetail(img) {
  try {
    const d = await api(`/images/${img.Id}`)
    detailJson.value = JSON.stringify(d, null, 2)
    detailOpen.value = true
  } catch (e) {
    toastErr(e.message)
  }
}

async function prune() {
  const ok = await confirm(t('images.confirmPrune'), {
    title: t('images.pruneTitle'),
    confirmText: t('images.pruneBtn'),
  })
  if (!ok) return
  try {
    const r = await api('/images/prune', { method: 'POST' })
    toastOk(t('images.toastPruned', { count: (r.ImagesDeleted || []).length }))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
