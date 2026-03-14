import { expect, test } from "@playwright/test";

import { installMockControlPlane } from "./mockControlPlane";

test.describe("visual design guards", () => {
  test.use({ viewport: { width: 1536, height: 960 } });

  test("shell layout baseline", async ({ page }) => {
    await installMockControlPlane(page);
    await page.goto("/");
    await expect(page.getByTestId("run-run-seed")).toBeVisible();
    await page.getByTestId("run-run-seed").click();
    await expect(page.getByTestId("app-layout")).toBeVisible();
    await expect(page.getByTestId("runs-list")).toContainText("seed run");
    await expect(page.getByTestId("events-list").locator(".event-row")).toHaveCount(0);
  });

  test("graph fallback baseline", async ({ page }) => {
    const mock = await installMockControlPlane(page);
    await page.goto("/");
    await expect(page.getByTestId("run-run-seed")).toBeVisible();
    await page.getByTestId("run-run-seed").click();

    mock.setFallback("run-seed", true);
    mock.setGraphNodeState("run-seed", "run-seed-coder", "failed", "failed");

    await page.evaluate(() => {
      const emit = (window as unknown as { __emitSSE: (event: unknown) => void }).__emitSSE;
      emit({
        id: "evt-fallback-visual",
        run_id: "run-seed",
        type: "run.fallback.serial",
        source: "scheduler",
        payload: { reason: "invariant violation" },
        created_at: "2026-03-01T10:05:00.000Z"
      });
    });

    await expect(page.getByTestId("fallback-banner")).toBeVisible();
    await page.getByTestId("tab-graph").click();
    await expect(page.getByTestId("graph-node-run-seed-coder")).toContainText("failed");
    await expect(page.getByTestId("graph-header")).toContainText("nodes:");
  });

  test("style token guard", async ({ page }) => {
    await installMockControlPlane(page);
    await page.goto("/");
    await expect(page.getByTestId("run-run-seed")).toBeVisible();

    const tokens = await page.evaluate(() => {
      const styles = getComputedStyle(document.documentElement);
      const heading = getComputedStyle(document.querySelector("h1") as HTMLElement);
      return {
        accent: styles.getPropertyValue("--accent").trim(),
        text: styles.getPropertyValue("--text").trim(),
        h1Spacing: heading.letterSpacing,
        h1Size: heading.fontSize
      };
    });

    expect(tokens.accent).toBe("#1677ff");
    expect(tokens.text).toBe("#111826");
    expect(Number.parseFloat(tokens.h1Spacing)).toBeGreaterThan(1);
    expect(Number.parseFloat(tokens.h1Spacing)).toBeLessThan(1.2);
    expect(Number.parseFloat(tokens.h1Size)).toBeGreaterThan(18);
    expect(Number.parseFloat(tokens.h1Size)).toBeLessThan(19);
  });
});
