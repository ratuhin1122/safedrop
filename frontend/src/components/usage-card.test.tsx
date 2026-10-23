import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { UsageCard } from "@/components/usage-card";
it("warns only above 80% and shows the configured upload limit", () => {
  const usage = { bytes_today: 80, bytes_limit: 100, uploads_this_hour: 7, uploads_limit: 20 };
  const { rerender } = render(<UsageCard signedIn usage={usage} />);
  expect(screen.queryByText(/Over 80%/)).not.toBeInTheDocument();
  expect(screen.getByText("7 / 20")).toBeVisible();
  rerender(<UsageCard signedIn usage={{ ...usage, bytes_today: 81 }} />);
  expect(screen.getByText(/Over 80%/)).toBeVisible();
  expect(screen.getByRole("progressbar")).toHaveClass("text-warning");
});
it("clamps the ring above the limit", () => {
  render(<UsageCard signedIn usage={{ bytes_today: 120, bytes_limit: 100, uploads_this_hour: 1, uploads_limit: 20 }} />);
  expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "100");
  expect(screen.getByRole("progressbar")).toHaveClass("text-destructive");
});
