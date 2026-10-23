import { useEffect, useReducer } from "react";

export function useCountdown(deadline: number) {
  const [, tick] = useReducer((value: number) => value + 1, 0);
  useEffect(() => {
    if (deadline <= Date.now()) return;
    const timer = setInterval(() => {
      tick();
      if (Date.now() >= deadline) clearInterval(timer);
    }, 250);
    return () => clearInterval(timer);
  }, [deadline]);
  return Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
}
