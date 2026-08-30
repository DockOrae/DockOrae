<script setup>
import { useVModel } from "@vueuse/core";
import { cn } from "@/lib/utils";

const props = defineProps({
  class: {
    type: [Boolean, null, String, Object, Array],
    required: false,
    skipCheck: true,
  },
  defaultValue: { type: [String, Number], required: false },
  modelValue: { type: [String, Number], required: false },
});

const emits = defineEmits(["update:modelValue"]);

const modelValue = useVModel(props, "modelValue", emits, {
  passive: true,
  defaultValue: props.defaultValue,
});
</script>

<template>
  <textarea
    v-model="modelValue"
    data-slot="textarea"
    :class="
      cn(
        'border-input bg-input placeholder:text-muted-foreground focus-visible:border-brand focus-visible:ring-brand/30 focus-visible:ring-2 aria-invalid:ring-destructive/20 aria-invalid:border-destructive flex field-sizing-content min-h-16 w-full rounded-[0.6rem] border px-3 py-2 text-[0.875rem] font-mono leading-[1.5] shadow-none transition-[color,box-shadow] outline-none disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
