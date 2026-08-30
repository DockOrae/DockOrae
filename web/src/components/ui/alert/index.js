import { cva } from "class-variance-authority";

export { default as Alert } from "./Alert.vue";
export { default as AlertDescription } from "./AlertDescription.vue";
export { default as AlertTitle } from "./AlertTitle.vue";

/** Docker Manager 风格提示条 */
export const alertVariants = cva(
  "relative w-full rounded-[0.6rem] border px-4 py-3 text-sm grid has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr] grid-cols-[0_1fr] has-[>svg]:gap-x-3 gap-y-0.5 items-start [&>svg]:size-4 [&>svg]:translate-y-0.5 [&>svg]:text-current",
  {
    variants: {
      variant: {
        default: "bg-surface2 text-text border-line [&>svg]:text-muted",
        destructive: "text-danger bg-danger/12 border-danger/30 [&>svg]:text-danger",
        warning: "text-warn bg-warn/12 border-warn/30 [&>svg]:text-warn",
        info: "text-info bg-info/12 border-info/30 [&>svg]:text-info",
        success: "text-ok bg-ok/12 border-ok/30 [&>svg]:text-ok",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
