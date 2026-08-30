<template>
  <div class="space-y-4 fade-up">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <Input v-model="keyword" class="!w-64 !pl-9" :placeholder="t('containers.searchPh')" />
      </div>
      <Select v-model="stateFilter" nullable-value="__all__" class="!w-36">
        <SelectTrigger class="!w-36">
          <SelectValue :placeholder="t('common.allStates')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__all__">{{ t('common.allStates') }}</SelectItem>
          <SelectItem value="running">{{ t('common.running') }}</SelectItem>
          <SelectItem value="exited">{{ t('common.exited') }}</SelectItem>
          <SelectItem value="paused">{{ t('common.paused') }}</SelectItem>
          <SelectItem value="restarting">{{ t('common.restarting') }}</SelectItem>
        </SelectContent>
      </Select>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="sm" @click="load">
          <Icon name="refresh" size="13" /> {{ t('common.refresh') }}
        </Button>
        <router-link :class="buttonVariants({ variant: 'brand', size: 'sm' })" to="/containers/new">
          <Icon name="plus" size="14" /> {{ t('containers.newContainer') }}
        </router-link>
      </div>
    </div>

    <!-- 表格 -->
    <Card class="overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{{ t('containers.thName') }}</TableHead>
            <TableHead>{{ t('containers.thImage') }}</TableHead>
            <TableHead>{{ t('containers.thStatus') }}</TableHead>
            <TableHead>{{ t('containers.thPorts') }}</TableHead>
            <TableHead>{{ t('containers.thCreated') }}</TableHead>
            <TableHead class="w-44">{{ t('containers.thActions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="c in filtered" :key="c.Id" class="cursor-pointer" @click="$router.push('/containers/' + c.Id)">
            <TableCell class="font-medium">{{ name(c) }}</TableCell>
            <TableCell class="text-muted">{{ c.Image }}</TableCell>
            <TableCell><StatusBadge :state="c.State" /></TableCell>
            <TableCell class="text-muted text-[12px] max-w-[260px] truncate">{{ ports(c) }}</TableCell>
            <TableCell class="text-muted text-[12px]">{{ formatDate(c.Created) }}</TableCell>
            <TableCell @click.stop>
              <div class="flex items-center gap-1">
                <Button v-if="c.State !== 'running'" variant="icon" :title="t('common.start')" @click="act(c, 'start')">
                  <Icon name="play" size="13" class="text-ok" />
                </Button>
                <Button v-if="c.State === 'running' && !c.State.includes('paused')" variant="icon" :title="t('common.pause')" @click="act(c, 'pause')">
                  <Icon name="pause" size="13" />
                </Button>
                <Button v-if="c.State === 'running'" variant="icon" :title="t('common.restart')" @click="act(c, 'restart')">
                  <Icon name="restart" size="13" />
                </Button>
                <Button v-if="c.State === 'paused'" variant="icon" :title="t('common.unpause')" @click="act(c, 'unpause')">
                  <Icon name="play" size="13" class="text-ok" />
                </Button>
                <Button variant="icon" class="text-danger" :title="t('common.delete')" @click="remove(c)">
                  <Icon name="trash" size="13" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
          <TableRow v-if="!filtered.length">
            <TableCell colspan="6" class="text-center text-muted py-10">{{ t('containers.noContainers') }}</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </Card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
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
