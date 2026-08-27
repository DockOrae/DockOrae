<template>
  <div class="space-y-4 fade-up">
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <input v-model="keyword" class="input !w-64 !pl-9" :placeholder="t('images.searchPh')" />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <button class="btn btn-ghost btn-sm" @click="prune"><Icon name="trash" size="13" /> {{ t('images.prune') }}</button>
        <button class="btn btn-brand btn-sm" @click="pullOpen = true"><Icon name="download" size="13" /> {{ t('images.pullImage') }}</button>
      </div>
    </div>

    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th class="th">{{ t('images.thImage') }}</th>
              <th class="th">{{ t('images.thId') }}</th>
              <th class="th">{{ t('images.thSize') }}</th>
              <th class="th">{{ t('images.thCreated') }}</th>
              <th class="th w-28">{{ t('images.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="img in filtered" :key="img.Id">
              <td class="td font-medium">{{ tag(img) }}</td>
              <td class="td font-mono text-[12px] text-muted">{{ shortId(img.Id) }}</td>
              <td class="td text-muted">{{ fmt(img.Size) }}</td>
              <td class="td text-muted text-[12px]">{{ formatDate(img.Created) }}</td>
              <td class="td">
                <div class="flex items-center gap-1">
                  <button class="btn btn-icon btn-sm" :title="t('images.detail')" @click="showDetail(img)">
                    <Icon name="eye" size="13" />
                  </button>
                  <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="remove(img)">
                    <Icon name="trash" size="13" />
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!filtered.length">
              <td colspan="5" class="td text-center text-muted py-10">{{ t('images.noImages') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 拉取镜像 -->
    <Modal :model-value="pullOpen" :title="t('images.pullTitle')" @close="closePull">
      <div class="space-y-3">
        <div>
          <label class="label">{{ t('images.imageName') }}</label>
          <input v-model="pullForm.from_image" class="input" :placeholder="t('images.imageNamePh')" />
        </div>
        <div>
          <label class="label">{{ t('images.tag') }}</label>
          <input v-model="pullForm.tag" class="input !w-40" :placeholder="t('images.tagPh')" />
        </div>
        <div v-if="pullProgress.length" class="code-panel border border-line rounded-lg p-3 h-44 overflow-y-auto font-mono text-[11px] leading-relaxed">
          <div v-for="(l, i) in pullProgress" :key="i" class="whitespace-pre-wrap break-all text-muted">{{ l }}</div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-ghost btn-sm" @click="closePull">{{ t('common.close') }}</button>
        <button class="btn btn-brand btn-sm" :disabled="pulling || !pullForm.from_image" @click="pull">
          <span v-if="pulling" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
          {{ pulling ? t('images.pulling') : t('images.pullStart') }}
        </button>
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
const pullProgress = ref([])
const detailOpen = ref(false)
const detailJson = ref('')
const confirm = useConfirm()

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
  pullProgress.value = []
  try {
    await pullImageStream(
      { from_image: pullForm.value.from_image, tag: pullForm.value.tag || 'latest' },
      (line) => {
        const status = line.status || line.error || line.id || ''
        if (line.progressDetail?.current != null && line.progressDetail?.total) {
          const pct = Math.round((line.progressDetail.current / line.progressDetail.total) * 100)
          pullProgress.value.push(`${status} ${pct}%`)
        } else if (status) {
          pullProgress.value.push(status)
        }
        if (pullProgress.value.length > 300) pullProgress.value.shift()
      }
    )
    toastOk(t('images.toastPullDone'))
    pullOpen.value = false
    load()
  } catch (e) {
    pullProgress.value.push(`❌ ${e.message}`)
    toastErr(e.message)
  } finally {
    pulling.value = false
  }
}

function closePull() {
  if (pulling.value) return
  pullOpen.value = false
  pullProgress.value = []
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
