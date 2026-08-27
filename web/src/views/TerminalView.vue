<template>
  <div class="space-y-4 fade-up">
    <!-- 分段按钮(仿 1Panel:一行占满,选中 2px 主题色边框) -->
    <SegmentedTabs :tabs="tabs" :active="active" @update:active="active = $event" />

    <!-- 终端 -->
    <div v-show="active === 'terminal'" class="fade-up">
      <TerminalTab />
    </div>
    <!-- 主机 -->
    <div v-show="active === 'host'" class="fade-up">
      <HostTab />
    </div>
    <!-- 快速命令 -->
    <div v-show="active === 'command'" class="fade-up">
      <CommandTab />
    </div>
    <!-- 配置 -->
    <div v-show="active === 'setting'" class="fade-up">
      <SettingTab />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import SegmentedTabs from '../components/SegmentedTabs.vue'
import TerminalTab from '../components/terminal/TerminalTab.vue'
import HostTab from '../components/terminal/HostTab.vue'
import CommandTab from '../components/terminal/CommandTab.vue'
import SettingTab from '../components/terminal/SettingTab.vue'

const { t } = useI18n()
const active = ref('terminal')
const tabs = [
  { key: 'terminal', labelKey: 'terminal.title' },
  { key: 'host', labelKey: 'terminal.host' },
  { key: 'command', labelKey: 'terminal.quickCommand' },
  { key: 'setting', labelKey: 'terminal.configTitle' },
]
</script>
