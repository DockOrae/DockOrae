<template>
  <div class="space-y-4 fade-up">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <input v-model="keyword" class="input !w-64 !pl-9" :placeholder="t('containers.searchPh')" />
      </div>
      <select v-model="stateFilter" class="input !w-36">
        <option value="">{{ t('common.allStates') }}</option>
        <option value="running">{{ t('common.running') }}</option>
        <option value="exited">{{ t('common.exited') }}</option>
        <option value="paused">{{ t('common.paused') }}</option>
        <option value="restarting">{{ t('common.restarting') }}</option>
      </select>
      <div class="ml-auto flex items-center gap-2">
        <button class="btn btn-ghost btn-sm" @click="load">
          <Icon name="refresh" size="13" /> {{ t('common.refresh') }}
        </button>
        <router-link to="/containers/new" class="btn btn-brand btn-sm">
          <Icon name="plus" size="14" /> {{ t('containers.newContainer') }}
        </router-link>
      </div>
    </div>

    <!-- 表格 -->
    <div class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th class="th">{{ t('containers.thName') }}</th>
              <th class="th">{{ t('containers.thImage') }}</th>
              <th class="th">{{ t('containers.thStatus') }}</th>
              <th class="th">{{ t('containers.thPorts') }}</th>
              <th class="th">{{ t('containers.thCreated') }}</th>
              <th class="th w-44">{{ t('containers.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in filtered" :key="c.Id" class="cursor-pointer" @click="$router.push('/containers/' + c.Id)">
              <td class="td font-medium">{{ name(c) }}</td>
              <td class="td text-muted">{{ c.Image }}</td>
              <td class="td"><StatusBadge :state="c.State" /></td>
              <td class="td text-muted text-[12px] max-w-[260px] truncate">{{ ports(c) }}</td>
              <td class="td text-muted text-[12px]">{{ formatDate(c.Created) }}</td>
              <td class="td" @click.stop>
                <div class="flex items-center gap-1">
                  <button v-if="c.State !== 'running'" class="btn btn-icon btn-sm" :title="t('common.start')" @click="act(c, 'start')">
                    <Icon name="play" size="13" class="text-ok" />
                  </button>
                  <button v-if="c.State === 'running'" class="btn btn-icon btn-sm" :title="t('common.stop')" @click="act(c, 'stop')">
                    <Icon name="stop" size="13" class="text-warn" />
                  </button>
                  <button v-if="c.State === 'running'" class="btn btn-icon btn-sm" :title="t('common.restart')" @click="act(c, 'restart')">
                    <Icon name="restart" size="13" />
                  </button>
                  <button v-if="c.State === 'running' && !c.State.includes('paused')" class="btn btn-icon btn-sm" :title="t('common.pause')" @click="act(c, 'pause')">
                    <Icon name="pause" size="13" />
                  </button>
                  <button v-if="c.State === 'paused'" class="btn btn-icon btn-sm" :title="t('common.unpause')" @click="act(c, 'unpause')">
                    <Icon name="play" size="13" class="text-ok" />
                  </button>
                  <button class="btn btn-icon btn-sm text-danger" :title="t('common.delete')" @click="remove(c)">
                    <Icon name="trash" size="13" />
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!filtered.length">
              <td colspan="6" class="td text-center text-muted py-10">{{ t('containers.noContainers') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { api } from '../api'
import { containerName, humanPorts, formatDate } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const containers = ref([])
const keyword = ref('')
const stateFilter = ref('')
const confirm = useConfirm()

const filtered = computed(() => {
  let list = containers.value
  if (stateFilter.value) list = list.filter((c) => c.State === stateFilter.value)
  if (keyword.value) {
    const k = keyword.value.toLowerCase()
    list = list.filter((c) => c.Names?.[0]?.toLowerCase().includes(k) || c.Image?.toLowerCase().includes(k))
  }
  return list
})

async function load() {
  containers.value = await api('/containers')
}
const name = (c) => containerName(c)
const ports = (c) => humanPorts(c.Ports)

async function act(c, action) {
  try {
    await api(`/containers/${c.Id}/${action}`, { method: 'POST' })
    toastOk(actionMap[action])
  } catch (e) {
    toastErr(e.message)
  }
}

const actionMap = {
  start: () => t('containers.toastStarted'),
  stop: () => t('containers.toastStopped'),
  restart: () => t('containers.toastRestarted'),
  pause: () => t('containers.toastPaused'),
  unpause: () => t('containers.toastResumed'),
}

async function remove(c) {
  const ok = await confirm(t('containers.confirmDelete', { name: name(c) }), {
    title: t('containers.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/containers/${c.Id}?force=true`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(load)
</script>
