<template>
  <div class="card p-5">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3 text-[13px]">
      <template v-for="item in rows" :key="item.label">
        <div v-if="item.value" class="flex items-start gap-2 min-w-0">
          <Icon :name="item.icon" size="14" class="text-muted mt-0.5 shrink-0" />
          <span class="text-muted shrink-0">{{ item.label }}</span>
          <span class="ml-auto text-right font-mono text-[12px] break-all min-w-0">{{ item.value }}</span>
        </div>
      </template>
    </div>

    <!-- 端口 -->
    <div v-if="portList.length" class="mt-5">
      <h4 class="text-xs text-muted font-semibold mb-2 uppercase tracking-wide">{{ t('overview.portBindings') }}</h4>
      <div class="flex flex-wrap gap-2">
        <span v-for="p in portList" :key="p" class="px-2.5 py-1 rounded-lg bg-surface2 border border-line font-mono text-[12px]">{{ p }}</span>
      </div>
    </div>

    <!-- 环境变量 -->
    <div v-if="envList.length" class="mt-5">
      <h4 class="text-xs text-muted font-semibold mb-2 uppercase tracking-wide">{{ t('overview.envVars') }}</h4>
      <div class="flex flex-wrap gap-2">
        <span v-for="e in envList" :key="e" class="px-2.5 py-1 rounded-lg bg-surface2 border border-line font-mono text-[12px]">{{ e }}</span>
      </div>
    </div>

    <!-- 挂载 -->
    <div v-if="mounts.length" class="mt-5">
      <h4 class="text-xs text-muted font-semibold mb-2 uppercase tracking-wide">{{ t('overview.mounts') }}</h4>
      <table class="table">
        <thead>
          <tr>
            <th class="th">{{ t('overview.mountType') }}</th>
            <th class="th">{{ t('overview.mountSource') }}</th>
            <th class="th">{{ t('overview.mountDest') }}</th>
            <th class="th">{{ t('overview.mountMode') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in mounts" :key="m.Name || m.Source">
            <td class="td">{{ m.Type }}</td>
            <td class="td font-mono text-[12px]">{{ m.Source || m.Name || '-' }}</td>
            <td class="td font-mono text-[12px]">{{ m.Destination }}</td>
            <td class="td">{{ m.RW === false ? 'ro' : 'rw' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 网络 -->
    <div v-if="netList.length" class="mt-5">
      <h4 class="text-xs text-muted font-semibold mb-2 uppercase tracking-wide">{{ t('overview.networks') }}</h4>
      <div class="flex flex-wrap gap-2">
        <span v-for="n in netList" :key="n" class="px-2.5 py-1 rounded-lg bg-surface2 border border-line font-mono text-[12px]">{{ n }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../../components/Icon.vue'
import { formatDate } from '../../util'

const { t } = useI18n()
const props = defineProps({ data: { type: Object, default: null } })

const rows = computed(() => {
  const d = props.data || {}
  const cfg = d.Config || {}
  const state = d.State || {}
  return [
    { label: t('overview.containerId'), icon: 'box', value: d.Id },
    { label: t('overview.image'), icon: 'image', value: cfg.Image },
    { label: t('overview.status'), icon: 'stats', value: state.Status },
    { label: t('overview.restartCount'), icon: 'restart', value: d.RestartCount },
    { label: t('overview.created'), icon: 'clock', value: formatDate(d.Created) },
    { label: t('overview.entrypoint'), icon: 'terminal', value: (cfg.Entrypoint || []).join(' ') },
    { label: t('overview.command'), icon: 'terminal', value: (cfg.Cmd || []).join(' ') },
    { label: t('overview.workdir'), icon: 'box', value: cfg.WorkingDir },
    { label: t('overview.user'), icon: 'key', value: cfg.User },
    { label: t('overview.hostname'), icon: 'info', value: cfg.Hostname },
    { label: t('overview.restartPolicy'), icon: 'restart', value: d.HostConfig?.RestartPolicy?.Name },
    { label: t('overview.networkMode'), icon: 'network', value: d.HostConfig?.NetworkMode },
  ]
})

const portList = computed(() => {
  const ports = props.data?.NetworkSettings?.Ports || {}
  return Object.entries(ports)
    .map(([k, v]) => (v && v.length ? v.map((p) => `${(p.HostIp || '0.0.0.0')}:${p.HostPort || ''}->${k}`).join(', ') : k))
    .filter(Boolean)
})

const envList = computed(() => (props.data?.Config?.Env || []).slice(0, 40))

const mounts = computed(() => props.data?.Mounts || [])

const netList = computed(() => {
  const nets = props.data?.NetworkSettings?.Networks || {}
  return Object.keys(nets)
})
</script>
