<template>
  <LogViewer :stream="`/containers/${id}/logs`" follow />
</template>

<script setup>
import LogViewer from '../../components/LogViewer.vue'

defineProps({ id: { type: String, required: true } })
</script>
