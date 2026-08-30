import { cva } from "class-variance-authority";

export { default as Button } from "./Button.vue";

/**
 * Docker Manager 风格按钮(等价旧 .btn / .btn-brand / .btn-ghost / .btn-ok / .btn-danger / .btn-icon)
 * 尺寸/圆角/悬停浮起/禁用态完全复刻 main.css 旧实现
 */
export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-[0.4rem] whitespace-nowrap rounded-[0.6rem] text-[0.85rem] font-medium border border-transparent transition-all duration-200 outline-none select-none cursor-pointer disabled:cursor-not-allowed disabled:opacity-45 disabled:pointer-events-none hover:-translate-y-px hover:shadow-[0_4px_14px_rgba(0,0,0,0.18)] active:translate-y-0 active:shadow-none focus-visible:ring-2 focus-visible:ring-brand/50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "text-text bg-transparent",
        brand: "bg-brand text-white hover:bg-brand-strong",
        primary: "bg-brand text-white hover:bg-brand-strong",
        ghost: "border-line text-muted bg-transparent hover:border-[#3a4456] hover:text-text hover:bg-surface2",
        outline: "border-line text-muted bg-transparent hover:border-[#3a4456] hover:text-text hover:bg-surface2",
        ok: "bg-ok/12 text-ok border-ok/30 hover:bg-ok/25",
        warning: "bg-warn/12 text-warn border-warn/30 hover:bg-warn/25",
        destructive: "bg-danger/12 text-danger border-danger/30 hover:bg-danger/25",
        icon: "p-[0.38rem] rounded-[0.5rem] border-line text-muted bg-transparent hover:border-[#3a4456] hover:text-text hover:bg-surface2",
      },
      size: {
        default: "px-[0.9rem] py-[0.45rem]",
        sm: "px-[0.6rem] py-[0.28rem] text-[0.78rem] rounded-[0.5rem]",
        icon: "p-[0.38rem] rounded-[0.5rem]",
        "icon-sm": "p-[0.38rem] rounded-[0.5rem]",
        lg: "px-6 py-2.5 text-[0.9rem]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
