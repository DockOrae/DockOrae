import { cva } from "class-variance-authority";

export { default as Badge } from "./Badge.vue";

/** Docker Manager 风格徽章(等价旧 .badge + StatusBadge 配色) */
export const badgeVariants = cva(
  "inline-flex items-center gap-[0.35rem] rounded-full border px-[0.6rem] py-[0.16rem] text-[0.72rem] font-semibold leading-[1.4] w-fit whitespace-nowrap shrink-0 transition-colors",
  {
    variants: {
      variant: {
        default: "border-transparent bg-surface2 text-muted",
        brand: "border-brand/30 bg-brand/12 text-brand",
        success: "border-ok/30 bg-ok/12 text-ok",
        warning: "border-warn/30 bg-warn/12 text-warn",
        destructive: "border-danger/30 bg-danger/12 text-danger",
        info: "border-info/30 bg-info/12 text-info",
        outline: "border-line text-muted",
        secondary: "border-transparent bg-surface2 text-muted",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
