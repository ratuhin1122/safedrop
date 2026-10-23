export type TurnstileOptions = {
  sitekey: string; theme: "light" | "dark"; size: "flexible";
  callback: (token: string) => void;
  "expired-callback": () => void;
  "timeout-callback": () => void;
  "error-callback": () => boolean;
};
export type TurnstileApi = { render: (element: HTMLElement, options: TurnstileOptions) => string; remove: (id: string) => void };
declare global { interface Window { turnstile?: TurnstileApi } }

let pending: Promise<TurnstileApi> | undefined;

export function loadTurnstile(): Promise<TurnstileApi> {
  if (window.turnstile) return Promise.resolve(window.turnstile);
  if (pending) return pending;
  pending = new Promise<TurnstileApi>((resolve, reject) => {
    const script = document.createElement("script");
    // Official widget runtime; API requests still use VITE_API_URL exclusively.
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    const fail = () => {
      clearTimeout(timer);
      script.remove();
      pending = undefined;
      reject(new Error("Security check could not load. Check your connection and try again."));
    };
    const timer = setTimeout(fail, 15000);
    script.onerror = fail;
    script.onload = () => {
      if (!window.turnstile) { fail(); return; }
      clearTimeout(timer);
      resolve(window.turnstile);
    };
    document.head.append(script);
  });
  return pending;
}
