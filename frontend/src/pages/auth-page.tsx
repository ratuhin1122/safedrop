import { useEffect, useRef, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { ApiError } from "@/api/client";
import { AuthFields } from "@/components/auth-fields";
import { Turnstile } from "@/components/turnstile";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { turnstileSiteKey } from "@/lib/env";
import { loginFormSchema, registerFormSchema, returnPath, type AuthFormValues } from "@/lib/auth-form";

export function AuthPage({ creating = false }: { creating?: boolean }) {
  const auth = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const destination = returnPath(location.state);
  const challengeRequired = creating && Boolean(turnstileSiteKey);
  const [resetKey, setResetKey] = useState(0);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  const { register, handleSubmit, watch, setValue, setError, clearErrors, formState: { errors, isSubmitting } } = useForm<AuthFormValues>({
    resolver: zodResolver(creating ? registerFormSchema : loginFormSchema),
    defaultValues: { email: "", password: "", turnstile_token: "" },
  });
  const password = watch("password");
  const onToken = (token: string) => {
    setValue("turnstile_token", token);
    if (token) clearErrors("turnstile_token");
  };

  async function submit(values: AuthFormValues) {
    clearErrors("root");
    if (challengeRequired && !values.turnstile_token) {
      setError("turnstile_token", { message: "Complete the security check before creating an account." });
      return;
    }
    controller.current = new AbortController();
    try {
      if (creating) await auth.signUp({ ...values, turnstile_token: challengeRequired ? values.turnstile_token : undefined }, controller.current.signal);
      else await auth.signIn(values, controller.current.signal);
      navigate(destination, { replace: true });
    } catch (error) {
      if (controller.current.signal.aborted) return;
      let message = error instanceof Error ? error.message : "Authentication failed. Please try again.";
      if (error instanceof ApiError && error.status === 429 && error.retryAfterSeconds !== null) message += ` Try again in ${error.retryAfterSeconds} seconds.`;
      setError("root", { message });
      if (challengeRequired) { setValue("turnstile_token", ""); setResetKey((value) => value + 1); }
    }
  }

  if (auth.isAuthenticated) return <Navigate to={destination} replace />;
  const title = creating ? "Create account" : "Welcome back";
  return <section className="mx-auto max-w-md rounded-xl border bg-card px-6 py-8 sm:p-10" aria-labelledby="auth-title">
    <h1 id="auth-title" className="text-3xl font-semibold tracking-tight">{title}</h1>
    <p className="mb-8 mt-3 text-sm leading-relaxed text-muted-foreground">
      {creating ? "A little space for the things you share." : "Sign in to your SafeDrop space."}
    </p>
    <form noValidate onSubmit={handleSubmit(submit)} aria-busy={isSubmitting} className="space-y-6">
      <fieldset disabled={isSubmitting} className="min-w-0 space-y-5">
        <legend className="sr-only">{creating ? "Create your account" : "Sign in"}</legend>
        <AuthFields register={register} errors={errors} creating={creating} password={password} />
      </fieldset>
      {challengeRequired && <Turnstile siteKey={turnstileSiteKey} onToken={onToken} resetKey={resetKey} />}
      {errors.turnstile_token && <p role="alert" className="text-sm text-destructive">{errors.turnstile_token.message}</p>}
      {errors.root && <p role="alert" className="text-sm text-destructive">{errors.root.message}</p>}
      <Button type="submit" disabled={isSubmitting} className="w-full">
        {isSubmitting ? (creating ? "Creating account…" : "Signing in…") : (creating ? "Create account" : "Sign in")}
      </Button>
      <p className="sr-only" role="status">{isSubmitting ? "Please wait while we verify your account." : ""}</p>
    </form>
    <p className="mt-7 text-center text-sm text-muted-foreground">
      {creating ? "Already have an account? " : "New to SafeDrop? "}
      <Link to={creating ? "/login" : "/register"} state={location.state} className="rounded-sm font-medium text-primary underline underline-offset-4">
        {creating ? "Sign in" : "Create account"}
      </Link>
    </p>
  </section>;
}
