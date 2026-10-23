import * as Dropdown from "@radix-ui/react-dropdown-menu";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export const DropdownMenu = Dropdown.Root;
export const DropdownMenuTrigger = Dropdown.Trigger;
export const DropdownMenuLabel = Dropdown.Label;

export function DropdownMenuContent({ className, sideOffset = 8, ...props }: ComponentProps<typeof Dropdown.Content>) {
  return <Dropdown.Portal>
    <Dropdown.Content sideOffset={sideOffset} className={cn(
      "z-50 min-w-52 rounded-lg border bg-popover p-2 text-popover-foreground shadow-lg outline-none", className,
    )} {...props} />
  </Dropdown.Portal>;
}

export function DropdownMenuItem({ className, ...props }: ComponentProps<typeof Dropdown.Item>) {
  return <Dropdown.Item className={cn(
    "flex min-h-11 cursor-default items-center rounded-md px-3 text-sm outline-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground data-[disabled]:opacity-50", className,
  )} {...props} />;
}

export function DropdownMenuSeparator() {
  return <Dropdown.Separator className="my-1 h-px bg-border" />;
}
