<template>
  <button
    class="dm-switch"
    type="button"
    :aria-label="t('theme.toggle')"
    :title="t('theme.toggle')"
    @click="toggleThemeWithTransition($event)"
  >
    <span class="dm-switch-check">
      <span class="dm-switch-icon">
        <Icon :name="theme === 'dark' ? 'moon' : 'sun'" size="12" filled class="dm-switch-svg" />
      </span>
    </span>
  </button>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { theme, toggleThemeWithTransition } from '../store'

const { t } = useI18n()
</script>

<style scoped>
/* 样式参照 valaxy-theme-press 的 PressSwitchAppearance(40x22 滑块开关) */
.dm-switch {
  position: relative;
  display: block;
  box-sizing: border-box;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  border: 1px solid var(--dm-line);
  background-color: var(--dm-surface2);
  cursor: pointer;
  padding: 0;
  transition:
    border-color 0.2s,
    background-color 0.2s;
}
.dm-switch:hover {
  border-color: #8e8e8e;
}
.dm-switch-check {
  position: absolute;
  top: 1px;
  left: 1px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background-color: #fff;
  box-shadow:
    0 1px 2px rgba(0, 0, 0, 0.04),
    0 1px 2px rgba(0, 0, 0, 0.06);
  transition:
    background-color 0.2s,
    transform 0.2s;
}
html[data-theme='dark'] .dm-switch-check {
  background-color: #1a1a1a;
  transform: translateX(18px);
}
.dm-switch-icon {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
}
.dm-switch-svg {
  color: rgba(60, 60, 60, 0.7);
}
html[data-theme='dark'] .dm-switch-svg {
  color: rgba(255, 255, 255, 0.87);
}
</style>
