import type { SVGProps } from "react";

type Name = "drop" | "sun" | "moon" | "user" | "chevron" | "close";
const paths: Record<Name, string> = {
  drop: "M12 3C9 7 5 10 5 14a7 7 0 0 0 14 0c0-4-4-7-7-11ZM9 15a3 3 0 0 0 3 3",
  sun: "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5",
  moon: "M20.5 13A8.5 8.5 0 0 1 11 3.5 8.5 8.5 0 1 0 20.5 13Z",
  user: "M8 8a4 4 0 1 0 8 0 4 4 0 0 0-8 0ZM4 21v-2a8 6 0 0 1 16 0v2",
  chevron: "m8 10 4 4 4-4",
  close: "m6 6 12 12M6 18 18 6",
};

export function Icon({ name, ...props }: SVGProps<SVGSVGElement> & { name: Name }) {
  return <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor"
    strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props}>
    <path d={paths[name]} />
  </svg>;
}
