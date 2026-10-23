import { useEffect } from "react";

export function usePublicMeta() {
  useEffect(() => {
    const previousTitle = document.title;
    document.title = "Shared file · SafeDrop";
    const tags = [["robots", "noindex, nofollow, noarchive"], ["referrer", "no-referrer"]];
    const cleanup = tags.map(([name, content]) => {
      const existing = document.head.querySelector<HTMLMetaElement>('meta[name="' + name + '"]');
      const tag = existing ?? document.createElement("meta");
      const previous = tag.getAttribute("content");
      tag.name = name; tag.content = content;
      if (!existing) document.head.appendChild(tag);
      return () => { if (!existing) tag.remove(); else if (previous === null) tag.removeAttribute("content"); else tag.content = previous; };
    });
    return () => { document.title = previousTitle; cleanup.forEach((restore) => restore()); };
  }, []);
}
