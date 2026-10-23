import { useState } from "react";
import type { FieldErrors, UseFormRegister } from "react-hook-form";
import { Input } from "@/components/ui/input";
import { passwordHint, type AuthFormValues } from "@/lib/auth-form";

type Props = { register: UseFormRegister<AuthFormValues>; errors: FieldErrors<AuthFormValues>; creating: boolean; password: string };

export function AuthFields({ register, errors, creating, password }: Props) {
  const [visible, setVisible] = useState(false);
  return <>
    <div className="space-y-2">
      <label htmlFor="email" className="text-sm font-medium">Email address</label>
      <Input id="email" type="email" autoComplete="email" autoCapitalize="none" spellCheck={false}
        aria-invalid={Boolean(errors.email)} aria-describedby={errors.email ? "email-error" : undefined}
        {...register("email")} />
      {errors.email && <p id="email-error" role="alert" className="text-sm text-destructive">{errors.email.message}</p>}
    </div>
    <div className="space-y-2">
      <label htmlFor="password" className="text-sm font-medium">Password</label>
      <div className="relative">
        <Input id="password" type={visible ? "text" : "password"} className="pr-20"
          autoComplete={creating ? "new-password" : "current-password"}
          aria-invalid={Boolean(errors.password)} aria-describedby={`${creating ? "password-hint " : ""}${errors.password ? "password-error" : ""}`.trim() || undefined}
          {...register("password")} />
        <button type="button" className="absolute inset-y-1 right-1 min-w-16 rounded-md px-3 text-sm text-primary hover:bg-accent"
          aria-label={visible ? "Hide password" : "Show password"} aria-pressed={visible} onClick={() => setVisible(!visible)}>
          {visible ? "Hide" : "Show"}
        </button>
      </div>
      {creating && <p id="password-hint" className="text-sm text-muted-foreground">{passwordHint(password)}</p>}
      {errors.password && <p id="password-error" role="alert" className="text-sm text-destructive">{errors.password.message}</p>}
    </div>
  </>;
}
