<template>
  <button
    type="button"
    class="dm-locale-trigger"
    :title="t('lang.toggle')"
    :aria-label="t('lang.toggle')"
    @click="toggleLocales"
  >
    <Icon
      name="translate"
      size="20"
      filled
      class="dm-locale-icon"
      :class="{ flip: locale === 'en' }"
    />
  </button>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { setLang } from '../i18n'

const { t, locale } = useI18n()

/** 与 OpenList-Docs 一致:点击 translate 图标直接在中英之间切换 */
function toggleLocales() {
  setLang(locale.value === 'zh-CN' ? 'en' : 'zh-CN')
}
</script>

<style scoped>
/* 样式参照 valaxy-theme-press 的 PressToggleLocale(legacy 模式) */
.dm-locale-trigger {
  display: flex;
  align-items: center;
  background: transparent;
  color: var(--dm-muted);
  font-weight: 500;
  font-size: 14px;
  cursor: pointer;
  padding: 4px;
  border-radius: 8px;
  border: none;
  transition: color 0.2s;
  line-height: 1;
}
.dm-locale-trigger:hover {
  color: var(--dm-text);
}
.dm-locale-icon {
  transition: transform 0.4s ease;
}
.dm-locale-icon.flip {
  transform: rotateY(180deg);
}
</style>
