<script setup>
import { cn } from "@/lib/utils";

const props = defineProps({
  class: {
    type: [Boolean, null, String, Object, Array],
    required: false,
    skipCheck: true,
  },
});
</script>

<template>
  <tbody data-slot="table-body" :class="cn('', props.class)">
    <slot />
  </tbody>
</template>
