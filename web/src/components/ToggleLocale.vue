<template>
  <div class="locale-wrap" ref="wrapRef">
    <button
      type="button"
      class="locale-trigger"
      :title="t('lang.toggle')"
      :aria-label="t('lang.toggle')"
      @click.stop="open = !open"
    >
      <span class="locale-flag" aria-hidden="true">{{ current?.flag }}</span>
      <Icon name="translate" size="16" class="locale-icon" />
    </button>

    <Transition name="dm-drop">
      <div v-if="open" class="locale-pop" @click.stop>
        <button
          v-for="l in LANGS"
          :key="l.code"
          type="button"
          class="locale-item"
          :class="{ active: locale === l.code }"
          @click="onPick(l.code)"
        >
          <span aria-hidden="true">{{ l.flag }}</span>
          <span>{{ l.label }}</span>
          <span v-if="locale === l.code" class="locale-check">✓</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { LANGS, setLang } from '../i18n'

const { t, locale } = useI18n()
const open = ref(false)
const wrapRef = ref(null)

const current = computed(() => LANGS.find((l) => l.code === locale.value) || LANGS[0])

function onPick(code) {
  setLang(code)
  open.value = false
}

function onDocClick(e) {
  if (open.value && wrapRef.value && !wrapRef.value.contains(e.target)) open.value = false
}
onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
</script>

<style scoped>
.locale-wrap {
  position: relative;
}
.locale-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dm-muted);
  font-size: 13px;
  cursor: pointer;
  transition: color 0.2s, background 0.2s;
}
.locale-trigger:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}
.locale-flag {
  font-size: 15px;
  line-height: 1;
}
.locale-pop {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 60;
  min-width: 190px;
  max-height: 65vh;
  overflow-y: auto;
  padding: 5px;
  border-radius: 12px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface);
  box-shadow: 0 10px 32px rgba(0, 0, 0, 0.22);
}
.locale-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dm-muted);
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}
.locale-item:hover {
  background: var(--dm-surface2);
  color: var(--dm-text);
}
.locale-item.active {
  color: var(--dm-brand);
  background: color-mix(in srgb, var(--dm-brand) 10%, transparent);
  font-weight: 600;
}
.locale-check {
  margin-left: auto;
  font-size: 12px;
}

.dm-drop-enter-active,
.dm-drop-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.dm-drop-enter-from,
.dm-drop-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}
</style>
