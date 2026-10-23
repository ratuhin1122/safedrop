import { act, render, screen } from "@testing-library/react";
import { useRef } from "react";
import { expect, it, vi } from "vitest";
import { useNearViewport } from "@/hooks/use-near-viewport";

it("defers image work until near the viewport and cleans up its observer", () => {
  let intersect!: IntersectionObserverCallback;
  const disconnect = vi.fn();
  vi.stubGlobal("IntersectionObserver", class {
    constructor(callback: IntersectionObserverCallback) { intersect = callback; }
    observe = vi.fn(); disconnect = disconnect;
  });
  function Harness() {
    const ref = useRef<HTMLDivElement>(null);
    const visible = useNearViewport(ref);
    return <div ref={ref}>{visible ? "Load image" : "Wait"}</div>;
  }
  const view = render(<Harness />);
  expect(screen.getByText("Wait")).toBeInTheDocument();
  act(() => intersect([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver));
  expect(screen.getByText("Load image")).toBeInTheDocument();
  view.unmount();
  expect(disconnect).toHaveBeenCalled();
});
