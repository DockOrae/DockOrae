<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    :fill="filled ? 'currentColor' : 'none'"
    :stroke="filled ? 'none' : 'currentColor'"
    :stroke-width="filled ? 0 : 2"
    stroke-linecap="round"
    stroke-linejoin="round"
    v-html="icons.template(name)"
  />
</template>

<script setup>
import icons from '../icons'

defineProps({
  name: { type: String, required: true },
  size: { type: [Number, String], default: 16 },
  /** fill 型图标(如 Remix Icon) */
  filled: { type: Boolean, default: false },
})
</script>
