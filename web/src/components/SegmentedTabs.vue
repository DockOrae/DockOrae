<template>
  <div class="w-full flex bg-surface2/50 border border-line rounded-lg p-1 gap-1">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      class="flex-1 px-4 py-2 rounded-md text-[13px] font-medium transition-all duration-200 border-2"
      :class="
        active === tab.key
          ? 'text-brand border-brand bg-surface shadow-sm'
          : 'text-muted border-transparent hover:text-text hover:bg-surface/60'
      "
      @click="$emit('update:active', tab.key)"
    >
      {{ t(tab.labelKey) }}
    </button>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

defineProps({
  tabs: { type: Array, required: true }, // [{key, labelKey}]
  active: { type: String, default: '' },
})
defineEmits(['update:active'])

const { t } = useI18n()
</script>
