import { useEffect, useState, type RefObject } from "react";

export function useNearViewport(target: RefObject<HTMLElement | null>) {
  const [visible, setVisible] = useState(() => typeof IntersectionObserver === "undefined");
  useEffect(() => {
    if (visible || !target.current) return;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) { setVisible(true); observer.disconnect(); }
    }, { rootMargin: "200px" });
    observer.observe(target.current);
    return () => observer.disconnect();
  }, [visible, target]);
  return visible;
}
