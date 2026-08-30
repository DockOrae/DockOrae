<template>
  <div class="space-y-4 fade-up">
    <div class="flex items-center justify-between">
      <p class="text-[13px] text-muted">{{ t('compose.count', { count: stacks.length }) }}</p>
      <Button v-if="!licenseActive" variant="ghost" size="sm" class="!text-amber-400 border-amber-400/40" :title="t('license.requiredHint')" @click="$router.push('/settings#license')">
        <Icon name="lock" size="14" /> {{ t('license.required') }}
      </Button>
      <Button v-else variant="brand" size="sm" @click="createOpen = true"><Icon name="plus" size="14" /> {{ t('compose.newStack') }}</Button>
    </div>

    <Card v-if="!stacks.length" class="p-12 text-center text-muted text-sm">
      <Icon name="compose" size="32" class="mx-auto mb-3 opacity-40" />
      {{ t('compose.noStacks') }}
    </Card>

    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <Card v-for="s in stacks" :key="s.project" class="p-5 hover:border-brand/40 transition-colors">
        <div class="flex items-center gap-3">
          <span class="w-10 h-10 rounded-xl bg-brand/10 text-brand flex items-center justify-center font-bold text-sm">
            {{ s.project[0].toUpperCase() }}
          </span>
          <div class="min-w-0">
            <router-link :to="'/compose/' + s.project" class="font-semibold text-[14px] hover:text-brand truncate block">
              {{ s.project }}
            </router-link>
            <div class="text-[11px] text-muted">{{ t('compose.servicesRunning', { running: s.running, total: s.services }) }}</div>
          </div>
          <span class="ml-auto"><StatusBadge :state="s.status" /></span>
        </div>
        <div class="mt-4 flex items-center gap-1.5 flex-wrap">
          <Button v-if="s.status !== 'running'" variant="ok" size="sm" @click="stackAct(s, 'start')"><Icon name="play" size="12" /> {{ t('common.start') }}</Button>
          <Button v-if="s.status === 'running'" variant="ghost" size="sm" @click="stackAct(s, 'stop')"><Icon name="stop" size="12" /> {{ t('common.stop') }}</Button>
          <Button variant="ghost" size="sm" @click="stackAct(s, 'restart')"><Icon name="restart" size="12" /> {{ t('common.restart') }}</Button>
          <router-link :class="buttonVariants({ variant: 'ghost', size: 'sm' })" :to="'/compose/' + s.project"><Icon name="edit" size="12" /> {{ t('common.manage') }}</router-link>
          <Button variant="icon" class="text-danger ml-auto" :title="t('compose.confirmDeleteTitle')" @click="remove(s)">
            <Icon name="trash" size="13" />
          </Button>
        </div>
      </Card>
    </div>

    <!-- 新建栈 -->
    <Modal :model-value="createOpen" :title="t('compose.createTitle')" @close="createOpen = false">
      <div class="space-y-3">
        <div>
          <Label>{{ t('compose.projectName') }}</Label>
          <Input v-model="form.project" :placeholder="t('compose.projectPh')" />
        </div>
        <div>
          <Label>{{ t('compose.yamlLabel') }}</Label>
          <Textarea
            v-model="form.yaml"
            rows="12"
            spellcheck="false"
            :placeholder="t('compose.yamlPh')"
          />
        </div>
        <!-- 部署过程实时输出 -->
        <div v-if="deploying || output" class="code-panel border border-line rounded-lg p-3 max-h-52 overflow-y-auto font-mono text-[11px] whitespace-pre-wrap" :class="deployFailed ? 'text-danger' : 'text-muted'">
          <template v-if="deploying">
            <div class="flex items-center gap-2 mb-1.5 text-brand">
              <span class="inline-block w-3 h-3 border-2 border-brand/30 border-t-brand rounded-full animate-spin" />
              {{ t('compose.deploying') }}
            </div>
          </template>
          <div v-for="(l, i) in outputLines" :key="i" class="leading-relaxed break-all">{{ l }}</div>
          <div v-if="deployFailed" class="text-danger font-semibold pt-1">{{ t('compose.deployFailed') }}</div>
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" size="sm" :disabled="deploying" @click="createOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="brand" size="sm" :disabled="!form.project || !form.yaml.trim() || deploying" @click="deploy">
          <span v-if="deploying" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
          {{ deploying ? t('compose.deploying') : t('compose.deploy') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card } from '@/components/ui/card'
import { api, composeStream } from '../api'
import { licenseActive } from '../store'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const stacks = ref([])
const createOpen = ref(false)
const deploying = ref(false)
const deployFailed = ref(false)
const outputLines = ref([])
const form = reactive({ project: '', yaml: '' })
const confirm = useConfirm()

async function load() {
  stacks.value = await api('/compose')
}

async function deploy() {
  deploying.value = true
  deployFailed.value = false
  outputLines.value = []
  try {
    await composeStream('/compose', { project: form.project, yaml: form.yaml }, (line) => {
      outputLines.value.push(line)
    })
    toastOk(t('compose.toastDeployOk'))
    createOpen.value = false
    form.project = ''
    form.yaml = ''
    load()
  } catch (e) {
    deployFailed.value = true
    outputLines.value.push(`❌ ${e.message}`)
    toastErr(e.message)
  } finally {
    deploying.value = false
  }
}

async function stackAct(s, action) {
  try {
    await api(`/compose/${s.project}/${action}`, { method: 'POST' })
    toastOk({ start: t('compose.toastStarted'), stop: t('compose.toastStopped'), restart: t('compose.toastRestarted') }[action])
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

async function remove(s) {
  const ok = await confirm(t('compose.confirmDelete', { project: s.project }), {
    title: t('compose.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/compose/${s.project}`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
