import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex min-h-11 shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring",
  { variants: {
    variant: {
      default: "bg-primary text-primary-foreground hover:opacity-90",
      ghost: "text-foreground hover:bg-accent hover:text-accent-foreground",
      outline: "border bg-card text-card-foreground hover:bg-accent",
    },
    size: { default: "px-5 py-2.5", icon: "size-11 p-0" },
  }, defaultVariants: { variant: "default", size: "default" } },
);

type Props = ComponentProps<"button"> & VariantProps<typeof buttonVariants> & { asChild?: boolean };

export function Button({ className, variant, size, asChild, ...props }: Props) {
  const Component = asChild ? Slot : "button";
  return <Component className={cn(buttonVariants({ variant, size }), className)} {...props} />;
}
