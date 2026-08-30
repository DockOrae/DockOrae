<script setup>
import { DialogClose } from "reka-ui";
import { cn } from "@/lib/utils";
import { Button } from '@/components/ui/button';

const props = defineProps({
  class: {
    type: [Boolean, null, String, Object, Array],
    required: false,
    skipCheck: true,
  },
  showCloseButton: { type: Boolean, required: false, default: false },
});
</script>

<template>
  <div
    data-slot="dialog-footer"
    :class="
      cn('flex items-center justify-end gap-2 px-5 py-3.5 border-t border-line bg-surface/60 rounded-b-[0.9rem]', props.class)
    "
  >
    <slot />
    <DialogClose v-if="showCloseButton" as-child>
      <Button variant="ghost"> Close </Button>
    </DialogClose>
  </div>
</template>
