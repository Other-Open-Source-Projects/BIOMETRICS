import { Page, Request, Route } from "@playwright/test";

type RunStatus = "queued" | "running" | "paused" | "cancelled" | "completed" | "failed";

type Run = {
  id: string;
  project_id: string;
  goal: string;
  status: RunStatus;
  mode: string;
  scheduler_mode: "dag_parallel_v1" | "serial";
  max_parallelism: number;
  fallback_triggered?: boolean;
  created_at: string;
};

type Task = {
  id: string;
  name: string;
  agent: string;
  status: string;
  lifecycle_state: string;
  priority: number;
  max_attempts: number;
  attempt: number;
};

type TaskGraph = {
  run_id: string;
  nodes: Array<{
    id: string;
    name: string;
    agent: string;
    depends_on?: string[];
    priority: number;
    status: string;
    lifecycle_state: string;
  }>;
  edges: Array<{ from: string; to: string }>;
  critical_path: string[];
  created_at: string;
};

type OrchestratorRun = {
  id: string;
  plan_id: string;
  project_id: string;
  goal: string;
  strategy_mode: "deterministic" | "adaptive" | "arena";
  status: "queued" | "running" | "paused" | "cancelled" | "completed" | "failed";
  current_step_id?: string;
  steps: Array<{
    id: string;
    name: string;
    status: "pending" | "running" | "completed" | "failed";
  }>;
  optimizer_recommendation_id?: string;
  optimizer_confidence?: "low" | "medium" | "high";
  created_at: string;
  updated_at: string;
};

type OptimizerRecommendation = {
  id: string;
  project_id: string;
  goal: string;
  strategy_mode: "deterministic" | "adaptive" | "arena";
  scheduler_mode: "dag_parallel_v1" | "serial";
  max_parallelism: number;
  model_preference: string;
  fallback_chain: string[];
  model_id?: string;
  context_budget: number;
  confidence: "low" | "medium" | "high";
  rationale: string;
  status: "generated" | "applied" | "rejected";
  applied_run_id?: string;
  rejected_reason?: string;
  predicted_gates: {
    quality_pass: boolean;
    time_pass: boolean;
    cost_pass: boolean;
    regression_pass: boolean;
    all_pass: boolean;
    gate_pass_count: number;
    predicted_quality_score: number;
    predicted_time_improvement_percent: number;
    predicted_cost_improvement_percent: number;
    predicted_cost_per_success: number;
    predicted_regression_detected: boolean;
    predicted_composite_score: number;
  };
  created_at: string;
  updated_at: string;
};

type CallRecord = {
  method: string;
  path: string;
  body?: unknown;
};

export type MockControlPlane = {
  calls: CallRecord[];
  setFallback(runID: string, active: boolean): void;
  setGraphNodeState(runID: string, nodeID: string, status: string, lifecycleState?: string): void;
};

async function parseBody(request: Request): Promise<unknown> {
  const raw = request.postData();
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw);
  } catch {
    return raw;
  }
}

function json(route: Route, status: number, payload: unknown): Promise<void> {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(payload)
  });
}

export async function installMockControlPlane(page: Page): Promise<MockControlPlane> {
  const now = "2026-03-01T10:00:00.000Z";
  let runCounter = 0;
  const calls: CallRecord[] = [];

  const runs = new Map<string, Run>();
  const tasksByRun = new Map<string, Task[]>();
  const graphsByRun = new Map<string, TaskGraph>();
  const orchestratorRuns = new Map<string, OrchestratorRun>();
  const optimizerRecommendations = new Map<string, OptimizerRecommendation>();
  let optimizerCounter = 0;
  let orchestratorCounter = 0;

  const seedRun: Run = {
    id: "run-seed",
    project_id: "biometrics",
    goal: "seed run",
    status: "running",
    mode: "autonomous",
    scheduler_mode: "dag_parallel_v1",
    max_parallelism: 8,
    fallback_triggered: false,
    created_at: now
  };
  runs.set(seedRun.id, seedRun);

  tasksByRun.set(seedRun.id, [
    {
      id: "run-seed-planner",
      name: "planner",
      agent: "planner",
      status: "completed",
      lifecycle_state: "completed",
      priority: 100,
      max_attempts: 3,
      attempt: 1
    },
    {
      id: "run-seed-coder",
      name: "coder",
      agent: "coder",
      status: "running",
      lifecycle_state: "running",
      priority: 90,
      max_attempts: 3,
      attempt: 1
    }
  ]);

  graphsByRun.set(seedRun.id, {
    run_id: seedRun.id,
    nodes: [
      {
        id: "run-seed-planner",
        name: "planner",
        agent: "planner",
        priority: 100,
        status: "completed",
        lifecycle_state: "completed"
      },
      {
        id: "run-seed-coder",
        name: "coder",
        agent: "coder",
        depends_on: ["run-seed-planner"],
        priority: 90,
        status: "running",
        lifecycle_state: "running"
      }
    ],
    edges: [{ from: "run-seed-planner", to: "run-seed-coder" }],
    critical_path: ["run-seed-planner", "run-seed-coder"],
    created_at: now
  });

  optimizerRecommendations.set("opt-rec-seed", {
    id: "opt-rec-seed",
    project_id: "biometrics",
    goal: "Seed recommendation",
    strategy_mode: "adaptive",
    scheduler_mode: "dag_parallel_v1",
    max_parallelism: 8,
    model_preference: "codex",
    fallback_chain: ["gemini", "nim"],
    context_budget: 32000,
    confidence: "medium",
    rationale: "seed recommendation for panel",
    status: "generated",
    predicted_gates: {
      quality_pass: true,
      time_pass: true,
      cost_pass: false,
      regression_pass: true,
      all_pass: false,
      gate_pass_count: 3,
      predicted_quality_score: 0.92,
      predicted_time_improvement_percent: 27.4,
      predicted_cost_improvement_percent: 18.1,
      predicted_cost_per_success: 0.0032,
      predicted_regression_detected: false,
      predicted_composite_score: 0.89
    },
    created_at: now,
    updated_at: now
  });

  await page.route("**/health/ready", async (route) => {
    if (route.request().method() !== "GET") {
      return json(route, 405, { error: "method not allowed" });
    }
    return json(route, 200, {
      ok: true,
      codex_auth_ready: true,
      provider_status: ["codex:ok", "gemini:ok", "nim:ok"],
      skills_loaded: 2,
      skills_errors: 0,
      skills_system_ready: true
    });
  });

  await page.addInitScript(() => {
    class MockEventSource {
      static instances: MockEventSource[] = [];

      url: string;
      readyState = 1;
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      onopen: ((event: Event) => void) | null = null;
      private listeners = new Map<string, Set<(event: MessageEvent) => void>>();

      constructor(url: string) {
        this.url = url;
        MockEventSource.instances.push(this);
      }

      addEventListener(type: string, listener: (event: MessageEvent) => void): void {
        const set = this.listeners.get(type) ?? new Set<(event: MessageEvent) => void>();
        set.add(listener);
        this.listeners.set(type, set);
      }

      removeEventListener(type: string, listener: (event: MessageEvent) => void): void {
        const set = this.listeners.get(type);
        if (!set) {
          return;
        }
        set.delete(listener);
      }

      close(): void {
        this.readyState = 2;
      }

      emit(type: string, payload: unknown): void {
        const data = typeof payload === "string" ? payload : JSON.stringify(payload);
        const message = new MessageEvent(type, { data });
        const listeners = this.listeners.get(type);
        if (listeners) {
          listeners.forEach((listener) => listener(message));
        }
        if (type === "message" && this.onmessage) {
          this.onmessage(message);
        }
      }
    }

    (window as unknown as { EventSource: unknown }).EventSource = MockEventSource;

    (window as unknown as { __emitSSE: (event: unknown) => void }).__emitSSE = (event: unknown) => {
      const instances = MockEventSource.instances;
      for (const instance of instances) {
        if (typeof event === "object" && event && "type" in (event as Record<string, unknown>)) {
          instance.emit((event as Record<string, string>).type, event);
        }
        instance.emit("message", event);
      }
    };
    (window as unknown as { __eventSourceCount: () => number }).__eventSourceCount = () => MockEventSource.instances.length;
  });

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method();
    const path = url.pathname;
    const body = await parseBody(request);
    calls.push({ method, path, body });

    if (method === "GET" && path === "/api/v1/projects") {
      return json(route, 200, [{ id: "biometrics", name: "BIOMETRICS" }]);
    }
    if (method === "GET" && path === "/api/v1/blueprints") {
      return json(route, 200, [
        {
          id: "universal-2026",
          name: "Universal 2026",
          version: "2026.02.1",
          description: "Curated profile",
          modules: [
            { id: "engine", name: "Engine", description: "Backend runtime" },
            { id: "webapp", name: "WebApp", description: "SaaS module" }
          ]
        }
      ]);
    }
    if (method === "GET" && path.startsWith("/api/v1/blueprints/")) {
      return json(route, 200, {
        id: "universal-2026",
        name: "Universal 2026",
        version: "2026.02.1",
        description: "Curated profile",
        modules: [
          { id: "engine", name: "Engine", description: "Backend runtime" },
          { id: "webapp", name: "WebApp", description: "SaaS module" }
        ]
      });
    }
    if (method === "GET" && path === "/api/v1/runs") {
      const allRuns = Array.from(runs.values()).sort((a, b) => b.created_at.localeCompare(a.created_at));
      return json(route, 200, allRuns);
    }
    if (method === "GET" && path === "/api/v1/models") {
      return json(route, 200, {
        default_primary: "codex",
        default_chain: ["gemini", "nim"],
        providers: [
          { id: "codex", name: "Codex", status: "ready", default: true, available: true, model_id: "gpt-5-codex" },
          { id: "gemini", name: "Gemini", status: "ready", default: false, available: true, model_id: "gemini-2.5-pro" },
          { id: "nim", name: "NIM", status: "ready", default: false, available: true, model_id: "llama-3.1-70b" }
        ]
      });
    }
    if (method === "GET" && path === "/api/v1/skills") {
      return json(route, 200, [
        {
          name: "skill-creator",
          description: "Create codex-compatible skills",
          short_description: "create skills",
          path_to_skill_md: "/mock/skills/skill-creator/SKILL.md",
          scope: "system",
          enabled: true
        },
        {
          name: "skill-installer",
          description: "Install codex-compatible skills",
          short_description: "install skills",
          path_to_skill_md: "/mock/skills/skill-installer/SKILL.md",
          scope: "system",
          enabled: true
        }
      ]);
    }
    if (method === "GET" && path === "/api/v1/orchestrator/capabilities") {
      return json(route, 200, {
        strategy_modes: ["deterministic", "adaptive", "arena"],
        policy_presets: ["strict", "balanced", "velocity"],
        max_parallelism: 32,
        resume_from_step: true,
        arena_mode: true,
        eval_support: true,
        decision_explain: true,
        audit_trail: true,
        idempotent_step_ids: true
      });
    }
    if (method === "GET" && path === "/api/v1/orchestrator/optimizer/recommendations") {
      const projectID = url.searchParams.get("project_id") ?? "";
      const status = url.searchParams.get("status") ?? "";
      const limit = Number(url.searchParams.get("limit") ?? "20");
      const list = Array.from(optimizerRecommendations.values())
        .filter((rec) => !projectID || rec.project_id === projectID)
        .filter((rec) => !status || rec.status === status)
        .sort((a, b) => b.created_at.localeCompare(a.created_at))
        .slice(0, Number.isFinite(limit) && limit > 0 ? limit : 20);
      return json(route, 200, list);
    }
    if (method === "POST" && path === "/api/v1/orchestrator/optimizer/recommendations") {
      optimizerCounter += 1;
      const payload = (body ?? {}) as Record<string, unknown>;
      const strategy = optimizerCounter % 2 === 0 ? "arena" : "adaptive";
      const confidence = strategy === "arena" ? "high" : "medium";
      const recommendation: OptimizerRecommendation = {
        id: `opt-rec-${optimizerCounter}`,
        project_id: String(payload.project_id ?? "biometrics"),
        goal: String(payload.goal ?? "Generate recommendation"),
        strategy_mode: strategy,
        scheduler_mode: "dag_parallel_v1",
        max_parallelism: strategy === "arena" ? 10 : 8,
        model_preference: "codex",
        fallback_chain: ["gemini", "nim"],
        context_budget: strategy === "arena" ? 42000 : 36000,
        confidence,
        rationale: "mock recommendation",
        status: "generated",
        predicted_gates: {
          quality_pass: true,
          time_pass: true,
          cost_pass: strategy === "arena",
          regression_pass: true,
          all_pass: strategy === "arena",
          gate_pass_count: strategy === "arena" ? 4 : 3,
          predicted_quality_score: strategy === "arena" ? 0.94 : 0.91,
          predicted_time_improvement_percent: strategy === "arena" ? 29.8 : 26.3,
          predicted_cost_improvement_percent: strategy === "arena" ? 22.7 : 18.5,
          predicted_cost_per_success: strategy === "arena" ? 0.0027 : 0.0033,
          predicted_regression_detected: false,
          predicted_composite_score: strategy === "arena" ? 0.93 : 0.88
        },
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      optimizerRecommendations.set(recommendation.id, recommendation);
      return json(route, 201, recommendation);
    }
    if (method === "GET" && path.startsWith("/api/v1/orchestrator/optimizer/recommendations/")) {
      const recommendationID = path.split("/")[6] ?? "";
      const recommendation = optimizerRecommendations.get(recommendationID);
      if (!recommendation) {
        return json(route, 404, { error: "recommendation not found" });
      }
      return json(route, 200, recommendation);
    }
    if (method === "POST" && path.match(/^\/api\/v1\/orchestrator\/optimizer\/recommendations\/[^/]+\/apply$/)) {
      const recommendationID = path.split("/")[6] ?? "";
      const recommendation = optimizerRecommendations.get(recommendationID);
      if (!recommendation) {
        return json(route, 404, { error: "recommendation not found" });
      }
      recommendation.status = "applied";
      recommendation.updated_at = new Date().toISOString();
      orchestratorCounter += 1;
      const runID = `orc-run-${orchestratorCounter}`;
      recommendation.applied_run_id = runID;
      const run: OrchestratorRun = {
        id: runID,
        plan_id: `orc-plan-${orchestratorCounter}`,
        project_id: recommendation.project_id,
        goal: recommendation.goal,
        strategy_mode: recommendation.strategy_mode,
        status: "running",
        current_step_id: "execute",
        steps: [
          { id: "plan", name: "plan", status: "completed" },
          { id: "execute", name: "execute", status: "running" },
          { id: "evaluate", name: "evaluate", status: "pending" }
        ],
        optimizer_recommendation_id: recommendation.id,
        optimizer_confidence: recommendation.confidence,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      orchestratorRuns.set(runID, run);
      optimizerRecommendations.set(recommendationID, recommendation);
      return json(route, 200, { recommendation_id: recommendationID, recommendation, run });
    }
    if (method === "POST" && path.match(/^\/api\/v1\/orchestrator\/optimizer\/recommendations\/[^/]+\/reject$/)) {
      const recommendationID = path.split("/")[6] ?? "";
      const recommendation = optimizerRecommendations.get(recommendationID);
      if (!recommendation) {
        return json(route, 404, { error: "recommendation not found" });
      }
      recommendation.status = "rejected";
      recommendation.rejected_reason = ((body as { reason?: string } | undefined)?.reason ?? "manual override").toString();
      recommendation.updated_at = new Date().toISOString();
      optimizerRecommendations.set(recommendationID, recommendation);
      return json(route, 200, recommendation);
    }
    if (method === "GET" && path.match(/^\/api\/v1\/orchestrator\/runs\/[^/]+$/)) {
      const runID = path.split("/")[5] ?? "";
      const run = orchestratorRuns.get(runID);
      if (!run) {
        return json(route, 404, { error: "orchestrator run not found" });
      }
      return json(route, 200, run);
    }
    if (method === "GET" && path.match(/^\/api\/v1\/orchestrator\/runs\/[^/]+\/scorecard$/)) {
      return json(route, 200, {
        run_id: path.split("/")[5] ?? "orc-run-1",
        quality_score: 0.92,
        median_time_to_green_seconds: 830,
        cost_per_success: 0.0031,
        critical_policy_violations: 0,
        success_rate: 0.99,
        timeouts: 0,
        dispatch_p95_ms: 170,
        fallback_rate: 0,
        backpressure_per_run: 1.2,
        composite_score: 0.91,
        thresholds: { quality_score_gte_0_90: true }
      });
    }
    if (method === "GET" && path === "/api/v1/evals/leaderboard") {
      return json(route, 200, [
        {
          strategy: "adaptive",
          runs: 3,
          quality_score: 0.91,
          median_time_to_green_seconds: 860,
          cost_per_success: 0.38,
          composite_score: 0.9,
          updated_at: now
        },
        {
          strategy: "deterministic",
          runs: 3,
          quality_score: 0.89,
          median_time_to_green_seconds: 1010,
          cost_per_success: 0.44,
          composite_score: 0.83,
          updated_at: now
        }
      ]);
    }
    if (method === "POST" && path === "/api/v1/auth/codex/login") {
      return json(route, 200, { ok: true });
    }
    if (method === "GET" && path === "/api/v1/auth/codex/status") {
      return json(route, 200, { ready: true });
    }
    if (method === "POST" && path === "/api/v1/runs") {
      runCounter += 1;
      const payload = (body ?? {}) as Record<string, unknown>;
      const runID = `run-${runCounter}`;
      const schedulerMode = (payload.scheduler_mode as "dag_parallel_v1" | "serial") || "dag_parallel_v1";
      const mode = payload.mode === "supervised" ? "supervised" : "autonomous";
      const run: Run = {
        id: runID,
        project_id: String(payload.project_id ?? "biometrics"),
        goal: String(payload.goal ?? "run"),
        status: "running",
        mode,
        scheduler_mode: schedulerMode,
        max_parallelism: schedulerMode === "serial" ? 1 : Number(payload.max_parallelism ?? 8),
        fallback_triggered: false,
        created_at: new Date().toISOString()
      };
      runs.set(runID, run);
      tasksByRun.set(runID, [
        {
          id: `${runID}-planner`,
          name: "planner",
          agent: "planner",
          status: "completed",
          lifecycle_state: "completed",
          priority: 100,
          max_attempts: 3,
          attempt: 1
        },
        {
          id: `${runID}-coder`,
          name: "coder",
          agent: "coder",
          status: "running",
          lifecycle_state: "running",
          priority: 90,
          max_attempts: 3,
          attempt: 1
        }
      ]);
      graphsByRun.set(runID, {
        run_id: runID,
        nodes: [
          {
            id: `${runID}-planner`,
            name: "planner",
            agent: "planner",
            priority: 100,
            status: "completed",
            lifecycle_state: "completed"
          },
          {
            id: `${runID}-coder`,
            name: "coder",
            agent: "coder",
            depends_on: [`${runID}-planner`],
            priority: 90,
            status: "running",
            lifecycle_state: "running"
          }
        ],
        edges: [{ from: `${runID}-planner`, to: `${runID}-coder` }],
        critical_path: [`${runID}-planner`, `${runID}-coder`],
        created_at: new Date().toISOString()
      });
      return json(route, 201, run);
    }

    const runMatch = path.match(/^\/api\/v1\/runs\/([^/]+)(?:\/(tasks|graph|attempts|pause|resume|cancel))?$/);
    if (runMatch) {
      const runID = runMatch[1];
      const action = runMatch[2] ?? "";
      const run = runs.get(runID);

      if (!run) {
        return json(route, 404, { error: "run not found" });
      }

      if (method === "GET" && action === "") {
        return json(route, 200, run);
      }
      if (method === "GET" && action === "tasks") {
        return json(route, 200, tasksByRun.get(runID) ?? []);
      }
      if (method === "GET" && action === "graph") {
        return json(route, 200, graphsByRun.get(runID));
      }
      if (method === "GET" && action === "attempts") {
        const attempts = (tasksByRun.get(runID) ?? []).map((task, index) => ({
          id: `${runID}-attempt-${index + 1}`,
          run_id: runID,
          task_id: task.id,
          agent: task.agent,
          status: task.status,
          started_at: now,
          finished_at: now
        }));
        return json(route, 200, attempts);
      }
      if (method === "POST" && action === "pause") {
        run.status = "paused";
        return json(route, 200, { status: run.status, run_id: runID });
      }
      if (method === "POST" && action === "resume") {
        run.status = "running";
        return json(route, 200, { status: run.status, run_id: runID });
      }
      if (method === "POST" && action === "cancel") {
        run.status = "cancelled";
        return json(route, 200, { status: run.status, run_id: runID });
      }
    }

    if (method === "GET" && path === "/api/v1/fs/tree") {
      const targetPath = url.searchParams.get("path") ?? ".";
      if (targetPath.includes("..")) {
        return json(route, 400, { error: "path escapes workspace" });
      }
      return json(route, 200, [
        { name: "README.md", path: "README.md", isDir: false },
        { name: "docs", path: "docs", isDir: true }
      ]);
    }

    if (method === "GET" && path === "/api/v1/fs/file") {
      const targetPath = url.searchParams.get("path") ?? "";
      if (!targetPath || targetPath.includes("..")) {
        return json(route, 400, { error: "path escapes workspace" });
      }
      return route.fulfill({
        status: 200,
        contentType: "text/plain; charset=utf-8",
        body: `mock file content for ${targetPath}`
      });
    }

    if (method === "POST" && path.match(/^\/api\/v1\/projects\/[^/]+\/bootstrap$/)) {
      return json(route, 200, { profile_id: "universal-2026", changed_files: 2 });
    }

    return json(route, 404, { error: "not found" });
  });

  return {
    calls,
    setFallback(runID: string, active: boolean): void {
      const run = runs.get(runID);
      if (!run) {
        return;
      }
      run.fallback_triggered = active;
    },
    setGraphNodeState(runID: string, nodeID: string, status: string, lifecycleState?: string): void {
      const graph = graphsByRun.get(runID);
      if (!graph) {
        return;
      }
      graph.nodes = graph.nodes.map((node) => {
        if (node.id !== nodeID) {
          return node;
        }
        return {
          ...node,
          status,
          lifecycle_state: lifecycleState ?? status
        };
      });
    }
  };
}
