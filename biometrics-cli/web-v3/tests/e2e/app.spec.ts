import { expect, test } from "@playwright/test";

import { installMockControlPlane } from "./mockControlPlane";

test("slash commands drive run control actions", async ({ page }) => {
  const mock = await installMockControlPlane(page);
  await page.goto("/");
  await expect(page.getByTestId("run-run-seed")).toBeVisible();
  await page.getByTestId("run-run-seed").click();

  const composer = page.getByTestId("composer-input");
  const send = page.getByTestId("composer-send");
  await composer.fill("/run Implement contracts");
  await send.click();

  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && entry.path === "/api/v1/runs").length)
    .toBeGreaterThan(0);
  await expect
    .poll(() =>
      mock.calls.find((entry) => entry.method === "POST" && entry.path === "/api/v1/runs" && (entry.body as { mode?: string })?.mode === "autonomous")
    )
    .toBeTruthy();
  await expect(composer).toHaveValue("");

  await composer.fill("/pause");
  await send.click();
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/runs\/.+\/pause$/.test(entry.path)).length)
    .toBe(1);
  await expect(composer).toHaveValue("");

  await composer.fill("/resume");
  await send.click();
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/runs\/.+\/resume$/.test(entry.path)).length)
    .toBe(1);
  await expect(composer).toHaveValue("");

  await composer.fill("/cancel");
  await send.click();
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/runs\/.+\/cancel$/.test(entry.path)).length)
    .toBe(1);
  await expect(composer).toHaveValue("");

  await composer.fill("/retry-failed");
  await send.click();

  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && entry.path === "/api/v1/runs").length)
    .toBeGreaterThan(1);
});

test("supervised mode posts payload and renders checkpoint banner when paused", async ({ page }) => {
  const mock = await installMockControlPlane(page);
  await page.goto("/");
  await expect(page.getByTestId("run-run-seed")).toBeVisible();

  await page.getByTestId("run-mode-select").selectOption("supervised");

  const composer = page.getByTestId("composer-input");
  const send = page.getByTestId("composer-send");
  await composer.fill("/run Supervised release control");
  await send.click();

  await expect
    .poll(() =>
      mock.calls.find(
        (entry) => entry.method === "POST" && entry.path === "/api/v1/runs" && (entry.body as { mode?: string })?.mode === "supervised"
      )
    )
    .toBeTruthy();

  await expect(page.getByTestId("run-run-1")).toBeVisible();
  await page.getByTestId("run-run-1").click();
  await page.getByTestId("action-pause").click();
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/runs\/.+\/pause$/.test(entry.path)).length)
    .toBe(1);

  await page.evaluate(() => {
    const emit = (window as unknown as { __emitSSE: (event: unknown) => void }).__emitSSE;
    emit({
      id: "evt-supervision-1",
      run_id: "run-1",
      type: "run.supervision.checkpoint",
      source: "scheduler",
      payload: { checkpoint: "after-planner", reason: "planner completed" },
      created_at: new Date().toISOString()
    });
  });

  await expect(page.getByTestId("supervision-banner")).toContainText("after-planner");
});

test("fallback banner and graph state update from live events", async ({ page }) => {
  const mock = await installMockControlPlane(page);
  await page.goto("/");
  await expect(page.getByTestId("run-run-seed")).toBeVisible();
  await page.getByTestId("run-run-seed").click();
  await expect.poll(async () => page.evaluate(() => (window as unknown as { __eventSourceCount: () => number }).__eventSourceCount())).toBeGreaterThan(0);

  await page.getByTestId("tab-graph").click();
  await expect(page.getByTestId("graph-node-run-seed-coder")).toContainText("running");

  mock.setFallback("run-seed", true);
  mock.setGraphNodeState("run-seed", "run-seed-coder", "failed", "failed");

  await page.evaluate(() => {
    const emit = (window as unknown as { __emitSSE: (event: unknown) => void }).__emitSSE;
    const createdAt = new Date().toISOString();
    emit({
      id: "evt-fallback-1",
      run_id: "run-seed",
      type: "run.fallback.serial",
      source: "scheduler",
      payload: { reason: "invariant violation" },
      created_at: createdAt
    });
    emit({
      id: "evt-task-failed-1",
      run_id: "run-seed",
      type: "task.failed",
      source: "tester",
      payload: { task: "run-seed-coder" },
      created_at: createdAt
    });
  });

  await expect(page.getByTestId("fallback-banner")).toBeVisible();
  await expect(page.getByTestId("graph-node-run-seed-coder")).toContainText("failed");
});

test("handles 5000 event storm and blocks traversal in files flow", async ({ page }) => {
  const mock = await installMockControlPlane(page);
  await page.goto("/");
  await expect(page.getByTestId("run-run-seed")).toBeVisible();
  await page.getByTestId("run-run-seed").click();
  await expect.poll(async () => page.evaluate(() => (window as unknown as { __eventSourceCount: () => number }).__eventSourceCount())).toBeGreaterThan(0);

  await page.evaluate(() => {
    const emit = (window as unknown as { __emitSSE: (event: unknown) => void }).__emitSSE;
    for (let i = 0; i < 5000; i += 1) {
      emit({
        id: `evt-storm-${i}`,
        run_id: "run-seed",
        type: "task.ready",
        source: "scheduler",
        payload: { task: `task-${i}` },
        created_at: new Date().toISOString()
      });
    }
  });

  await expect.poll(() => page.locator("[data-testid='events-list'] .event-row").count()).toBeGreaterThan(100);

  const composer = page.getByTestId("composer-input");
  await composer.fill("/pause");
  await composer.press("Enter");
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/runs\/.+\/pause$/.test(entry.path)).length)
    .toBe(1);

  await page.getByTestId("tab-files").click();
  await page.getByTestId("file-path-input").fill("../");
  await page.getByTestId("file-path-open").click();
  await expect(page.getByTestId("files-error")).toContainText("Access denied");
});

test("optimizer flow generates recommendation, applies it, and starts orchestrator run", async ({ page }) => {
  const mock = await installMockControlPlane(page);
  await page.goto("/");

  const panel = page.getByTestId("optimizer-panel");
  await expect(panel).toBeVisible();

  await panel.getByRole("button", { name: "Generate Recommendation" }).click();
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "POST" && entry.path === "/api/v1/orchestrator/optimizer/recommendations").length)
    .toBe(1);

  await expect(page.getByTestId("optimizer-gate-quality")).toContainText("PASS");
  await expect(page.getByTestId("optimizer-gate-time")).toContainText("PASS");
  await expect(page.getByTestId("optimizer-gate-cost")).toContainText(/PASS|FAIL/);
  await expect(page.getByTestId("optimizer-gate-regression")).toContainText("PASS");

  await panel.getByRole("button", { name: "Apply & Start" }).click();

  await expect
    .poll(() =>
      mock.calls.filter((entry) => entry.method === "POST" && /\/api\/v1\/orchestrator\/optimizer\/recommendations\/[^/]+\/apply$/.test(entry.path)).length
    )
    .toBe(1);
  await expect
    .poll(() => mock.calls.filter((entry) => entry.method === "GET" && /\/api\/v1\/orchestrator\/runs\/orc-run-\d+$/.test(entry.path)).length)
    .toBeGreaterThan(0);

  await expect(panel.getByRole("button", { name: "Apply & Start" })).toHaveCount(0);
  await expect(panel.getByText("applied")).toBeVisible();
});
