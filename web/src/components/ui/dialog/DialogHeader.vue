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
  <div
    data-slot="dialog-header"
    :class="cn('flex items-center justify-between gap-2 px-5 py-4 border-b border-line', props.class)"
  >
    <slot />
  </div>
</template>
