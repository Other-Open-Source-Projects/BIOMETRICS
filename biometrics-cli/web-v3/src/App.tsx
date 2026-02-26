import { useEffect, useMemo, useRef, useState } from "react";

type SchedulerMode = "dag_parallel_v1" | "serial";
type RunMode = "autonomous" | "supervised";
type SkillSelectionMode = "auto" | "explicit" | "off";
type StrategyMode = "deterministic" | "adaptive" | "arena";
type PolicyPreset = "strict" | "balanced" | "velocity";
type GraphFilter = "all" | "failed" | "retrying" | "blocked";
type IconName =
  | "project"
  | "scheduler"
  | "blueprint"
  | "orchestrator"
  | "arena"
  | "score"
  | "shield"
  | "runs"
  | "timeline"
  | "command"
  | "emergency"
  | "tasks"
  | "graph"
  | "files"
  | "diff"
  | "terminal"
  | "skills"
  | "play"
  | "pause"
  | "resume"
  | "cancel"
  | "retry"
  | "status";

type Run = {
  id: string;
  project_id: string;
  goal: string;
  status: string;
  mode: string;
  skills?: string[];
  skill_selection_mode?: SkillSelectionMode;
  scheduler_mode?: SchedulerMode;
  max_parallelism?: number;
  fallback_triggered?: boolean;
  model_preference?: string;
  fallback_chain?: string[];
  model_id?: string;
  context_budget?: number;
  blueprint_profile?: string;
  blueprint_modules?: string[];
  bootstrap?: boolean;
  created_at: string;
};

type Task = {
  id: string;
  name: string;
  agent: string;
  status: string;
  lifecycle_state?: string;
  priority?: number;
  max_attempts?: number;
  attempt: number;
  last_error?: string;
  started_at?: string;
  finished_at?: string;
};

type EventItem = {
  id: string;
  run_id?: string;
  type: string;
  source: string;
  payload?: Record<string, string>;
  created_at: string;
};

type TaskGraphNode = {
  id: string;
  name: string;
  agent: string;
  depends_on?: string[];
  priority: number;
  status: string;
  lifecycle_state: string;
};

type TaskGraphEdge = {
  from: string;
  to: string;
};

type TaskGraph = {
  run_id: string;
  nodes: TaskGraphNode[];
  edges: TaskGraphEdge[];
  critical_path?: string[];
  created_at: string;
};

type Project = {
  id: string;
  name: string;
};

type BlueprintModule = {
  id: string;
  name: string;
  description: string;
};

type BlueprintProfile = {
  id: string;
  name: string;
  version: string;
  description: string;
  modules: BlueprintModule[];
};

type ModelProvider = {
  id: string;
  name: string;
  status: string;
  default: boolean;
  available: boolean;
  model_id?: string;
  description?: string;
};

type ModelCatalog = {
  default_primary: string;
  default_chain: string[];
  providers: ModelProvider[];
};

type SkillMetadata = {
  name: string;
  description: string;
  short_description?: string;
  path_to_skill_md: string;
  scope: "repo" | "user" | "system" | "admin";
  enabled: boolean;
};

type OrchestratorCapabilities = {
  strategy_modes: StrategyMode[];
  policy_presets: string[];
  max_parallelism: number;
  resume_from_step: boolean;
  arena_mode: boolean;
  eval_support: boolean;
  decision_explain: boolean;
  audit_trail: boolean;
  idempotent_step_ids: boolean;
};

type OrchestratorStep = {
  id: string;
  name: string;
  status: "pending" | "running" | "completed" | "failed";
  depends_on?: string[];
  error?: string;
};

type OrchestratorRun = {
  id: string;
  plan_id: string;
  project_id: string;
  goal: string;
  strategy_mode: StrategyMode;
  status: string;
  current_step_id?: string;
  underlying_run_id?: string;
  steps: OrchestratorStep[];
  created_at: string;
  updated_at: string;
  error?: string;
};

type OrchestratorScorecard = {
  run_id: string;
  underlying_run_id?: string;
  quality_score: number;
  median_time_to_green_seconds: number;
  cost_per_success: number;
  critical_policy_violations: number;
  success_rate: number;
  timeouts: number;
  dispatch_p95_ms: number;
  fallback_rate: number;
  backpressure_per_run: number;
  composite_score: number;
  thresholds: Record<string, boolean>;
};

type EvalRun = {
  id: string;
  name: string;
  candidate_strategy_mode: StrategyMode;
  baseline_strategy_mode: StrategyMode;
  sample_size: number;
  status: string;
  regression_detected: boolean;
  metrics: {
    quality_score: number;
    median_time_to_green_seconds: number;
    cost_per_success: number;
    composite_score: number;
  };
};

type LeaderboardEntry = {
  strategy: string;
  runs: number;
  quality_score: number;
  median_time_to_green_seconds: number;
  cost_per_success: number;
  composite_score: number;
  updated_at: string;
};

type OptimizerValidation = {
  id: string;
  recommendation_id: string;
  eval_run_id?: string;
  status: string;
  quality_pass: boolean;
  time_pass: boolean;
  cost_pass: boolean;
  regression_pass: boolean;
  all_pass: boolean;
  summary?: string;
  created_at: string;
  updated_at: string;
};

type OptimizerRecommendation = {
  id: string;
  project_id: string;
  goal: string;
  strategy_mode: StrategyMode;
  scheduler_mode: SchedulerMode;
  max_parallelism: number;
  model_preference: string;
  fallback_chain?: string[];
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
    predicted_cost_per_success?: number;
    predicted_regression_detected: boolean;
    predicted_composite_score: number;
  };
  validation?: OptimizerValidation;
  created_at: string;
  updated_at: string;
};

type OptimizerApplyResult = {
  recommendation_id: string;
  recommendation?: OptimizerRecommendation;
  run: OrchestratorRun;
};

type Tab = "files" | "diff" | "terminal" | "tasks" | "graph";
type RefreshKey = "tasks" | "runs" | "graph";

type QuickCommand = {
  label: string;
  value: string;
  icon: IconName;
};

type TabItem = {
  id: Tab;
  label: string;
  icon: IconName;
};

const SSE_EVENT_TYPES = [
  "run.created",
  "run.started",
  "run.paused",
  "run.resumed",
  "run.cancelled",
  "run.completed",
  "run.failed",
  "task.started",
  "task.ready",
  "task.blocked",
  "task.completed",
  "task.failed",
  "task.retry.scheduled",
  "diff.produced",
  "agent.restarted",
  "run.backpressure",
  "run.fallback.serial",
  "run.supervision.checkpoint",
  "run.graph.materialized",
  "auth.codex.login.started",
  "auth.codex.login.succeeded",
  "auth.codex.login.failed",
  "model.selected",
  "model.fallback.triggered",
  "model.fallback.exhausted",
  "context.compiled",
  "skills.loaded",
  "skill.selected",
  "skill.invocation.blocked",
  "skill.install.started",
  "skill.install.succeeded",
  "skill.install.failed",
  "skill.create.started",
  "skill.create.succeeded",
  "skill.create.failed",
  "orchestrator.plan.generated",
  "orchestrator.step.started",
  "orchestrator.step.completed",
  "orchestrator.step.failed",
  "orchestrator.decision.explained",
  "orchestrator.resume.applied",
  "eval.run.started",
  "eval.run.completed",
  "eval.run.failed",
  "eval.metric.regression.detected",
  "optimizer.recommendation.generated",
  "optimizer.recommendation.applied",
  "optimizer.recommendation.rejected",
  "optimizer.validation.completed",
  "blueprint.selected",
  "blueprint.bootstrap.started",
  "blueprint.module.applied",
  "blueprint.module.skipped",
  "blueprint.bootstrap.completed"
] as const;

const TABS: TabItem[] = [
  { id: "files", label: "files", icon: "files" },
  { id: "diff", label: "diff", icon: "diff" },
  { id: "terminal", label: "terminal", icon: "terminal" },
  { id: "tasks", label: "tasks", icon: "tasks" },
  { id: "graph", label: "graph", icon: "graph" }
];

const QUICK_COMMANDS: QuickCommand[] = [
  { label: "run", value: "/run Build hardened release candidate with full verification", icon: "play" },
  { label: "orchestrate", value: "/orchestrate Build apex orchestrator run with arena strategy", icon: "orchestrator" },
  { label: "eval", value: "/eval-run", icon: "score" },
  { label: "pause", value: "/pause", icon: "pause" },
  { label: "resume", value: "/resume", icon: "resume" },
  { label: "cancel", value: "/cancel", icon: "cancel" },
  { label: "retry failed", value: "/retry-failed", icon: "retry" }
];

const MAX_SEEN_EVENT_IDS = 20000;

function normalizeStatusClass(value: string | undefined): string {
  return (value ?? "unknown").toLowerCase().replace(/[^a-z0-9_-]/g, "-");
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleTimeString();
}

function eventTone(type: string): "neutral" | "warn" | "danger" | "ok" {
  if (type.includes("failed") || type.includes("cancelled") || type.includes("regression")) {
    return "danger";
  }
  if (type.includes("fallback") || type.includes("backpressure") || type.includes("blocked")) {
    return "warn";
  }
  if (type.includes("checkpoint")) {
    return "warn";
  }
  if (type.includes("completed") || type.includes("started") || type.includes("ready")) {
    return "ok";
  }
  return "neutral";
}

function HeaderTitle({ icon, children }: { icon: IconName; children: string }) {
  return (
    <h2 className="title-with-icon">
      <AppIcon name={icon} />
      {children}
    </h2>
  );
}

function AppIcon({ name }: { name: IconName }) {
  switch (name) {
    case "project":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 7h16v12H4z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M9 7V5h6v2" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "scheduler":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M12 8v5l3 2" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "blueprint":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 4h14v16H5z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M8 8h8M8 12h8M8 16h5" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "orchestrator":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="6" cy="6" r="2" fill="currentColor" />
          <circle cx="18" cy="6" r="2" fill="currentColor" />
          <circle cx="6" cy="18" r="2" fill="currentColor" />
          <circle cx="18" cy="18" r="2" fill="currentColor" />
          <path d="M8 6h8M6 8v8M18 8v8M8 18h8" fill="none" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case "arena":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 7h7v7H4zM13 10h7v7h-7z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M11 10l2-2" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "score":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 19V9M10 19V5M15 19v-7M20 19v-4" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "shield":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3l7 3v5c0 5-3 8.5-7 10-4-1.5-7-5-7-10V6z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M9 12l2 2 4-4" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "runs":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 6h16M4 12h16M4 18h16" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "timeline":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 12h14" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <circle cx="7" cy="12" r="2" fill="currentColor" />
          <circle cx="17" cy="12" r="2" fill="currentColor" />
        </svg>
      );
    case "command":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 5h16v14H4z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M8 9l3 3-3 3M13 15h3" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "emergency":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 3l8 4v5c0 5-3.5 8.2-8 9-4.5-.8-8-4-8-9V7z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M12 8v5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
          <circle cx="12" cy="16.5" r="1" fill="currentColor" />
        </svg>
      );
    case "tasks":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 7h16M4 12h16M4 17h16" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <circle cx="7" cy="7" r="1" fill="currentColor" />
          <circle cx="7" cy="12" r="1" fill="currentColor" />
          <circle cx="7" cy="17" r="1" fill="currentColor" />
        </svg>
      );
    case "graph":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="6" cy="6" r="2" fill="currentColor" />
          <circle cx="18" cy="6" r="2" fill="currentColor" />
          <circle cx="12" cy="18" r="2" fill="currentColor" />
          <path d="M8 7.2l8 0M7.5 8l3.6 8M16.5 8l-3.6 8" fill="none" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      );
    case "files":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 6h8l2 2h6v10H4z" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "diff":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M7 5v14M17 5v14M5 9h4M15 15h4" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "terminal":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 6h16v12H4z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M8 10l2 2-2 2M12 14h4" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    case "skills":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M6 4h12v16H6z" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M9 8h6M9 12h6M9 16h4" fill="none" stroke="currentColor" strokeWidth="1.8" />
        </svg>
      );
    case "play":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M8 6l10 6-10 6z" fill="currentColor" />
        </svg>
      );
    case "pause":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M8 6h3v12H8zM13 6h3v12h-3z" fill="currentColor" />
        </svg>
      );
    case "resume":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M9 6l8 6-8 6z" fill="currentColor" />
        </svg>
      );
    case "cancel":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M7 7l10 10M17 7L7 17" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        </svg>
      );
    case "retry":
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M7 7h5v5" fill="none" stroke="currentColor" strokeWidth="1.8" />
          <path d="M17 12a5 5 0 10-2 4" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      );
    default:
      return (
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="4" fill="currentColor" />
        </svg>
      );
  }
}

export default function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [graph, setGraph] = useState<TaskGraph | null>(null);
  const [blueprints, setBlueprints] = useState<BlueprintProfile[]>([]);
  const [modelCatalog, setModelCatalog] = useState<ModelCatalog | null>(null);
  const [skillsCatalog, setSkillsCatalog] = useState<SkillMetadata[]>([]);
  const [codexAuthReady, setCodexAuthReady] = useState<boolean | null>(null);
  const [orchestratorCapabilities, setOrchestratorCapabilities] = useState<OrchestratorCapabilities | null>(null);
  const [orchestratorRun, setOrchestratorRun] = useState<OrchestratorRun | null>(null);
  const [orchestratorScorecard, setOrchestratorScorecard] = useState<OrchestratorScorecard | null>(null);
  const [strategyMode, setStrategyMode] = useState<StrategyMode>("adaptive");
  const [policyPreset, setPolicyPreset] = useState<PolicyPreset>("balanced");
  const [objectiveQuality, setObjectiveQuality] = useState(0.5);
  const [objectiveSpeed, setObjectiveSpeed] = useState(0.3);
  const [objectiveCost, setObjectiveCost] = useState(0.2);
  const [evalRun, setEvalRun] = useState<EvalRun | null>(null);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [optimizerRecommendations, setOptimizerRecommendations] = useState<OptimizerRecommendation[]>([]);
  const [optimizerActiveID, setOptimizerActiveID] = useState("");
  const [optimizerBusy, setOptimizerBusy] = useState(false);

  const [selectedProject, setSelectedProject] = useState("biometrics");
  const [selectedRun, setSelectedRun] = useState("");
  const [selectedBlueprintProfile, setSelectedBlueprintProfile] = useState("");
  const [selectedBlueprintModules, setSelectedBlueprintModules] = useState<string[]>([]);
  const [enableBootstrap, setEnableBootstrap] = useState(false);
  const [runMode, setRunMode] = useState<RunMode>("autonomous");
  const [skillSelectionMode, setSkillSelectionMode] = useState<SkillSelectionMode>("auto");
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [schedulerMode, setSchedulerMode] = useState<SchedulerMode>("dag_parallel_v1");
  const [maxParallelism, setMaxParallelism] = useState(8);
  const [modelPreference, setModelPreference] = useState("codex");
  const [fallbackChainInput, setFallbackChainInput] = useState("gemini,nim");
  const [modelID, setModelID] = useState("");
  const [contextBudget, setContextBudget] = useState(24000);

  const [composer, setComposer] = useState("");
  const [activeTab, setActiveTab] = useState<Tab>("tasks");
  const [graphFilter, setGraphFilter] = useState<GraphFilter>("all");
  const [files, setFiles] = useState<Array<{ name: string; path: string; isDir: boolean }>>([]);
  const [filePath, setFilePath] = useState(".");
  const [fileContent, setFileContent] = useState("Select a file to view content.");
  const [filesError, setFilesError] = useState("");

  const eventBufferRef = useRef<EventItem[]>([]);
  const eventFlushHandleRef = useRef<number | null>(null);
  const seenEventIDsRef = useRef<Set<string>>(new Set());
  const seenEventOrderRef = useRef<string[]>([]);
  const refreshTimerRef = useRef<Record<RefreshKey, number | null>>({
    tasks: null,
    runs: null,
    graph: null
  });

  const selectedRunObject = useMemo(() => runs.find((run) => run.id === selectedRun), [runs, selectedRun]);

  const selectedProfile = useMemo(
    () => blueprints.find((profile) => profile.id === selectedBlueprintProfile),
    [blueprints, selectedBlueprintProfile]
  );
  const selectedOptimizerRecommendation = useMemo(
    () => optimizerRecommendations.find((rec) => rec.id === optimizerActiveID) ?? optimizerRecommendations[0] ?? null,
    [optimizerRecommendations, optimizerActiveID]
  );

  const visibleEvents = useMemo(() => events.slice(-500), [events]);
  const supervisionCheckpointEvent = useMemo(() => {
    if (!selectedRunObject || selectedRunObject.mode !== "supervised" || selectedRunObject.status !== "paused") {
      return null;
    }

    for (let i = events.length - 1; i >= 0; i -= 1) {
      const event = events[i];
      if (event.type === "run.supervision.checkpoint") {
        return event;
      }
    }

    return null;
  }, [events, selectedRunObject]);

  const graphNodes = useMemo(() => {
    if (!graph) {
      return [];
    }
    return graph.nodes.filter((node) => {
      if (graphFilter === "all") {
        return true;
      }
      if (graphFilter === "failed") {
        return node.status === "failed";
      }
      if (graphFilter === "retrying") {
        return node.lifecycle_state === "retrying";
      }
      if (graphFilter === "blocked") {
        return node.lifecycle_state === "blocked";
      }
      return true;
    });
  }, [graph, graphFilter]);

  const runStats = useMemo(() => {
    let running = 0;
    let completed = 0;
    let failed = 0;
    let paused = 0;

    for (const run of runs) {
      if (run.status === "running") running += 1;
      else if (run.status === "completed") completed += 1;
      else if (run.status === "failed" || run.status === "cancelled") failed += 1;
      else if (run.status === "paused") paused += 1;
    }

    return {
      total: runs.length,
      running,
      completed,
      failed,
      paused
    };
  }, [runs]);

  const taskStats = useMemo(() => {
    let running = 0;
    let completed = 0;
    let failed = 0;
    let blocked = 0;

    for (const task of tasks) {
      if (task.status === "running") running += 1;
      else if (task.status === "completed") completed += 1;
      else if (task.status === "failed" || task.status === "cancelled") failed += 1;
      if (task.lifecycle_state === "blocked") blocked += 1;
    }

    return {
      total: tasks.length,
      running,
      completed,
      failed,
      blocked
    };
  }, [tasks]);

  useEffect(() => {
    void loadProjects();
    void loadRuns();
    void loadBlueprints();
    void loadModels();
    void loadSkills();
    void loadReadiness();
    void loadOrchestratorCapabilities();
    void loadEvalLeaderboard();
    void loadOptimizerRecommendations();
  }, []);

  useEffect(() => {
    if (!orchestratorRun) {
      return;
    }
    const handle = window.setInterval(() => {
      void loadOrchestratorRun(orchestratorRun.id);
      void loadOrchestratorScorecard(orchestratorRun.id);
    }, 1200);
    return () => window.clearInterval(handle);
  }, [orchestratorRun]);

  useEffect(() => {
    if (!evalRun) {
      return;
    }
    const handle = window.setInterval(() => {
      void loadEvalRun(evalRun.id);
      void loadEvalLeaderboard();
    }, 1400);
    return () => window.clearInterval(handle);
  }, [evalRun]);

  useEffect(() => {
    if (!selectedRun) {
      return;
    }

    setEvents([]);
    eventBufferRef.current = [];
    seenEventIDsRef.current.clear();
    seenEventOrderRef.current = [];

    void loadTasks(selectedRun);
    void loadGraph(selectedRun);

    const source = streamEvents(selectedRun);
    return () => {
      source.close();
      clearRefreshTimers();
    };
  }, [selectedRun]);

  useEffect(() => {
    void loadOptimizerRecommendations();
  }, [selectedProject]);

  useEffect(() => {
    void loadFiles(filePath);
  }, [filePath]);

  useEffect(() => {
    return () => {
      clearRefreshTimers();
      if (eventFlushHandleRef.current !== null) {
        window.cancelAnimationFrame(eventFlushHandleRef.current);
        eventFlushHandleRef.current = null;
      }
      eventBufferRef.current = [];
    };
  }, []);

  async function loadProjects() {
    try {
      const res = await fetch("/api/v1/projects");
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as Project[];
      setProjects(data);
      if ((!selectedProject || !data.some((project) => project.id === selectedProject)) && data.length > 0) {
        setSelectedProject(data[0].id);
      }
    } catch {
      // keep current UI state
    }
  }

  async function loadRuns() {
    try {
      const res = await fetch("/api/v1/runs?limit=40");
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as Run[];
      setRuns(data);
      if (!selectedRun && data.length > 0) {
        setSelectedRun(data[0].id);
      }
    } catch {
      // keep current UI state
    }
  }

  async function loadBlueprints() {
    try {
      const res = await fetch("/api/v1/blueprints");
      if (!res.ok) {
        setBlueprints([]);
        return;
      }
      const data = (await res.json()) as BlueprintProfile[];
      setBlueprints(data);
    } catch {
      setBlueprints([]);
    }
  }

  async function loadModels() {
    try {
      const res = await fetch("/api/v1/models");
      if (!res.ok) {
        setModelCatalog(null);
        return;
      }
      const data = (await res.json()) as ModelCatalog;
      setModelCatalog(data);
      if (!modelPreference && data.default_primary) {
        setModelPreference(data.default_primary);
      }
      if (!fallbackChainInput && data.default_chain.length > 0) {
        setFallbackChainInput(data.default_chain.join(","));
      }
    } catch {
      setModelCatalog(null);
    }
  }

  async function loadSkills() {
    try {
      const res = await fetch("/api/v1/skills");
      if (!res.ok) {
        setSkillsCatalog([]);
        return;
      }
      const data = (await res.json()) as SkillMetadata[];
      setSkillsCatalog(data);
    } catch {
      setSkillsCatalog([]);
    }
  }

  async function loadReadiness() {
    try {
      const res = await fetch("/health/ready");
      if (!res.ok) {
        setCodexAuthReady(false);
        return;
      }
      const data = (await res.json()) as { codex_auth_ready?: boolean };
      setCodexAuthReady(Boolean(data.codex_auth_ready));
    } catch {
      setCodexAuthReady(false);
    }
  }

  function normalizedObjective() {
    const q = Math.max(0, objectiveQuality);
    const s = Math.max(0, objectiveSpeed);
    const c = Math.max(0, objectiveCost);
    const sum = q + s + c;
    if (sum <= 0) {
      return { quality: 0.5, speed: 0.3, cost: 0.2 };
    }
    return {
      quality: q / sum,
      speed: s / sum,
      cost: c / sum
    };
  }

  function policyProfileFromPreset(preset: PolicyPreset) {
    if (preset === "strict") {
      return {
        exfiltration: "strict",
        secrets: "strict",
        filesystem: "workspace",
        network: "restricted",
        approvals: "required"
      };
    }
    if (preset === "velocity") {
      return {
        exfiltration: "balanced",
        secrets: "balanced",
        filesystem: "workspace",
        network: "restricted",
        approvals: "never"
      };
    }
    return {
      exfiltration: "balanced",
      secrets: "balanced",
      filesystem: "workspace",
      network: "restricted",
      approvals: "on-risk"
    };
  }

  async function loadOrchestratorCapabilities() {
    try {
      const res = await fetch("/api/v1/orchestrator/capabilities");
      if (!res.ok) {
        setOrchestratorCapabilities(null);
        return;
      }
      const data = (await res.json()) as OrchestratorCapabilities;
      setOrchestratorCapabilities(data);
    } catch {
      setOrchestratorCapabilities(null);
    }
  }

  async function createOrchestratorRun(goal: string) {
    const objective = normalizedObjective();
    const safeParallelism = Math.max(1, Math.min(32, maxParallelism));
    const normalizedFallbackChain = fallbackChainInput
      .split(",")
      .map((entry) => entry.trim().toLowerCase())
      .filter((entry, index, list) => entry.length > 0 && list.indexOf(entry) === index);

    const payload = {
      project_id: selectedProject,
      goal,
      strategy_mode: strategyMode,
      agent_profiles: [
        {
          name: "planner",
          allowed_tools: ["read", "graph", "metrics"],
          max_parallelism: Math.max(1, Math.min(4, safeParallelism)),
          model_policy: "quality"
        },
        {
          name: "coder",
          allowed_tools: ["read", "write", "test"],
          max_parallelism: safeParallelism,
          model_policy: "balanced"
        },
        {
          name: "reviewer",
          allowed_tools: ["read", "test", "comment"],
          max_parallelism: Math.max(1, Math.min(6, safeParallelism)),
          model_policy: "quality"
        }
      ],
      policy_profile: policyProfileFromPreset(policyPreset),
      objective,
      skills: selectedSkills,
      skill_selection_mode: skillSelectionMode,
      scheduler_mode: schedulerMode,
      max_parallelism: schedulerMode === "serial" ? 1 : safeParallelism,
      model_preference: modelPreference.trim().toLowerCase() || "codex",
      fallback_chain: normalizedFallbackChain,
      model_id: modelID.trim(),
      context_budget: Math.max(1000, Math.min(200000, contextBudget || 24000))
    };

    try {
      const res = await fetch("/api/v1/orchestrator/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as OrchestratorRun;
      setOrchestratorRun(data);
      setOrchestratorScorecard(null);
      await loadOrchestratorRun(data.id);
    } catch {
      // keep UI state
    }
  }

  async function loadOrchestratorRun(runID: string) {
    try {
      const res = await fetch(`/api/v1/orchestrator/runs/${runID}`);
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as OrchestratorRun;
      setOrchestratorRun(data);
    } catch {
      // keep UI state
    }
  }

  async function loadOrchestratorScorecard(runID: string) {
    try {
      const res = await fetch(`/api/v1/orchestrator/runs/${runID}/scorecard`);
      if (!res.ok) {
        return;
      }
      const score = (await res.json()) as OrchestratorScorecard;
      setOrchestratorScorecard(score);
    } catch {
      // keep UI state
    }
  }

  async function resumeOrchestratorFromExecute(runID: string) {
    try {
      const res = await fetch(`/api/v1/orchestrator/runs/${runID}/resume-from-step`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ step_id: "execute" })
      });
      if (!res.ok) {
        return;
      }
      const run = (await res.json()) as OrchestratorRun;
      setOrchestratorRun(run);
      await loadOrchestratorScorecard(run.id);
    } catch {
      // keep UI state
    }
  }

  async function startEvalRun() {
    try {
      const res = await fetch("/api/v1/evals/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: "apex-suite",
          candidate_strategy_mode: strategyMode,
          baseline_strategy_mode: "deterministic",
          sample_size: 500,
          quality_target: 0.9,
          speed_improvement_min: 0.25,
          cost_reduction_min: 0.2
        })
      });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as EvalRun;
      setEvalRun(data);
      await loadEvalRun(data.id);
      await loadEvalLeaderboard();
    } catch {
      // keep UI state
    }
  }

  async function loadEvalRun(runID: string) {
    try {
      const res = await fetch(`/api/v1/evals/runs/${runID}`);
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as EvalRun;
      setEvalRun(data);
    } catch {
      // keep UI state
    }
  }

  async function loadEvalLeaderboard() {
    try {
      const res = await fetch("/api/v1/evals/leaderboard");
      if (!res.ok) {
        setLeaderboard([]);
        return;
      }
      const data = (await res.json()) as LeaderboardEntry[];
      setLeaderboard(data);
    } catch {
      setLeaderboard([]);
    }
  }

  async function loadOptimizerRecommendations() {
    try {
      const res = await fetch(`/api/v1/orchestrator/optimizer/recommendations?project_id=${encodeURIComponent(selectedProject)}&limit=20`);
      if (!res.ok) {
        setOptimizerRecommendations([]);
        return;
      }
      const data = (await res.json()) as OptimizerRecommendation[];
      setOptimizerRecommendations(data);
      if (data.length > 0 && !optimizerActiveID) {
        setOptimizerActiveID(data[0].id);
      }
    } catch {
      setOptimizerRecommendations([]);
    }
  }

  async function generateOptimizerRecommendation() {
    const goal = composer.trim() || selectedRunObject?.goal || "Improve Apex gates for release readiness";
    setOptimizerBusy(true);
    try {
      const res = await fetch("/api/v1/orchestrator/optimizer/recommendations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          project_id: selectedProject,
          goal,
          objective: normalizedObjective()
        })
      });
      if (!res.ok) {
        return;
      }
      const rec = (await res.json()) as OptimizerRecommendation;
      setOptimizerActiveID(rec.id);
      await loadOptimizerRecommendations();
    } catch {
      // keep UI state
    } finally {
      setOptimizerBusy(false);
    }
  }

  async function applyOptimizerRecommendation(recommendationID: string) {
    setOptimizerBusy(true);
    try {
      const res = await fetch(`/api/v1/orchestrator/optimizer/recommendations/${recommendationID}/apply`, {
        method: "POST"
      });
      if (!res.ok) {
        return;
      }
      const result = (await res.json()) as OptimizerApplyResult;
      setOrchestratorRun(result.run);
      setOptimizerActiveID(result.recommendation_id || result.recommendation?.id || recommendationID);
      await loadOptimizerRecommendations();
      await loadOrchestratorRun(result.run.id);
    } catch {
      // keep UI state
    } finally {
      setOptimizerBusy(false);
    }
  }

  async function rejectOptimizerRecommendation(recommendationID: string) {
    setOptimizerBusy(true);
    try {
      const res = await fetch(`/api/v1/orchestrator/optimizer/recommendations/${recommendationID}/reject`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: "manual override by operator" })
      });
      if (!res.ok) {
        return;
      }
      await loadOptimizerRecommendations();
    } catch {
      // keep UI state
    } finally {
      setOptimizerBusy(false);
    }
  }

  async function codexLogin() {
    try {
      await fetch("/api/v1/auth/codex/login", { method: "POST" });
      await loadReadiness();
      await loadModels();
    } catch {
      // keep UI state
    }
  }

  async function loadTasks(runID: string) {
    try {
      const res = await fetch(`/api/v1/runs/${runID}/tasks`);
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as Task[];
      setTasks(data);
    } catch {
      // keep current UI state
    }
  }

  async function loadGraph(runID: string) {
    try {
      const res = await fetch(`/api/v1/runs/${runID}/graph`);
      if (!res.ok) {
        setGraph(null);
        return;
      }
      const data = (await res.json()) as TaskGraph;
      setGraph(data);
    } catch {
      setGraph(null);
    }
  }

  async function loadFiles(path: string) {
    try {
      const res = await fetch(`/api/v1/fs/tree?path=${encodeURIComponent(path)}`);
      if (!res.ok) {
        setFiles([]);
        setFilesError("Access denied or invalid path.");
        return;
      }
      const data = (await res.json()) as Array<{ name: string; path: string; isDir: boolean }>;
      setFiles(data);
      setFilesError("");
    } catch {
      setFiles([]);
      setFilesError("Access denied or invalid path.");
    }
  }

  async function openFile(path: string) {
    try {
      const res = await fetch(`/api/v1/fs/file?path=${encodeURIComponent(path)}`);
      if (!res.ok) {
        setFileContent("Failed to load file.");
        return;
      }
      setFileContent(await res.text());
    } catch {
      setFileContent("Failed to load file.");
    }
  }

  function clearRefreshTimers() {
    for (const key of Object.keys(refreshTimerRef.current) as RefreshKey[]) {
      const handle = refreshTimerRef.current[key];
      if (handle !== null) {
        window.clearTimeout(handle);
        refreshTimerRef.current[key] = null;
      }
    }
  }

  function scheduleRefresh(kind: RefreshKey, delayMs: number, fn: () => Promise<void>) {
    if (refreshTimerRef.current[kind] !== null) {
      return;
    }
    refreshTimerRef.current[kind] = window.setTimeout(() => {
      refreshTimerRef.current[kind] = null;
      void fn();
    }, delayMs);
  }

  function enqueueEvent(event: EventItem) {
    eventBufferRef.current.push(event);
    if (eventFlushHandleRef.current !== null) {
      return;
    }
    eventFlushHandleRef.current = window.requestAnimationFrame(() => {
      const chunk = eventBufferRef.current;
      eventBufferRef.current = [];
      eventFlushHandleRef.current = null;
      if (chunk.length === 0) {
        return;
      }
      setEvents((prev) => [...prev, ...chunk].slice(-5000));
    });
  }

  function isDuplicateEventID(eventID: string): boolean {
    if (!eventID) {
      return false;
    }

    if (seenEventIDsRef.current.has(eventID)) {
      return true;
    }

    seenEventIDsRef.current.add(eventID);
    seenEventOrderRef.current.push(eventID);

    if (seenEventOrderRef.current.length > MAX_SEEN_EVENT_IDS) {
      const dropped = seenEventOrderRef.current.shift();
      if (dropped) {
        seenEventIDsRef.current.delete(dropped);
      }
    }

    return false;
  }

  function processIncomingEvent(event: EventItem, runID: string) {
    if (event.id && isDuplicateEventID(event.id)) {
      return;
    }

    enqueueEvent(event);

    if (event.type.startsWith("task.")) {
      scheduleRefresh("tasks", 250, async () => loadTasks(runID));
    }

    if (
      event.type.startsWith("task.") ||
      event.type === "run.graph.materialized" ||
      event.type === "run.fallback.serial"
    ) {
      scheduleRefresh("graph", 350, async () => loadGraph(runID));
    }

    if (event.type.startsWith("run.")) {
      scheduleRefresh("runs", 300, loadRuns);
    }

    if (event.type.startsWith("optimizer.")) {
      void loadOptimizerRecommendations();
    }
  }

  function streamEvents(runID: string) {
    const source = new EventSource(`/api/v1/events?run_id=${encodeURIComponent(runID)}&limit=200`);

    const handleMessage = (raw: MessageEvent) => {
      if (typeof raw.data !== "string") {
        return;
      }
      try {
        const event = JSON.parse(raw.data) as EventItem;
        processIncomingEvent(event, runID);
      } catch {
        // ignore malformed payload
      }
    };

    source.onmessage = handleMessage;

    for (const eventType of SSE_EVENT_TYPES) {
      source.addEventListener(eventType, (raw: Event) => {
        handleMessage(raw as MessageEvent);
      });
    }

    source.onerror = () => {
      source.close();
    };

    return source;
  }

  async function createRun(goal: string) {
    const safeParallelism = Math.max(1, Math.min(32, maxParallelism));
    const normalizedFallbackChain = fallbackChainInput
      .split(",")
      .map((entry) => entry.trim().toLowerCase())
      .filter((entry, index, list) => entry.length > 0 && list.indexOf(entry) === index);

    const payload: Record<string, unknown> = {
      project_id: selectedProject,
      goal,
      mode: runMode,
      skill_selection_mode: skillSelectionMode,
      skills: selectedSkills,
      scheduler_mode: schedulerMode,
      max_parallelism: schedulerMode === "serial" ? 1 : safeParallelism,
      model_preference: modelPreference.trim().toLowerCase() || "codex",
      fallback_chain: normalizedFallbackChain,
      model_id: modelID.trim(),
      context_budget: Math.max(1000, Math.min(200000, contextBudget || 24000))
    };

    if (selectedBlueprintProfile) {
      payload.blueprint_profile = selectedBlueprintProfile;
    }
    if (selectedBlueprintModules.length > 0) {
      payload.blueprint_modules = selectedBlueprintModules;
    }
    if (enableBootstrap) {
      payload.bootstrap = true;
    }

    try {
      const res = await fetch("/api/v1/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      if (!res.ok) {
        return;
      }
      const run = (await res.json()) as Run;
      setSelectedRun(run.id);
      setEvents([]);
      await loadRuns();
      await loadGraph(run.id);
    } catch {
      // keep UI state
    }
  }

  async function runAction(action: "pause" | "resume" | "cancel", runID: string) {
    try {
      await fetch(`/api/v1/runs/${runID}/${action}`, { method: "POST" });
      await loadRuns();
      await loadTasks(runID);
    } catch {
      // keep UI state
    }
  }

  function toggleModule(moduleID: string) {
    setSelectedBlueprintModules((prev) => {
      if (prev.includes(moduleID)) {
        return prev.filter((id) => id !== moduleID);
      }
      return [...prev, moduleID].sort();
    });
  }

  function toggleSkill(skillName: string) {
    setSelectedSkills((prev) => {
      if (prev.includes(skillName)) {
        return prev.filter((name) => name !== skillName);
      }
      return [...prev, skillName].sort();
    });
  }

  function describeEvent(event: EventItem): string {
    const payload = event.payload;
    if (!payload) {
      return "";
    }

    if (event.type === "blueprint.selected") {
      const profile = payload.profile ?? "";
      const modules = payload.modules ? ` modules=${payload.modules}` : "";
      return `selected profile ${profile}${modules}`.trim();
    }

    if (event.type === "blueprint.bootstrap.started") {
      return `bootstrap started for ${payload.profile ?? "profile"}`;
    }

    if (event.type === "blueprint.module.applied" || event.type === "blueprint.module.skipped") {
      return `${payload.module ?? "module"} (${event.type.endsWith("applied") ? "applied" : "skipped"})`;
    }

    if (event.type === "blueprint.bootstrap.completed") {
      return payload.changed_files ? `changed: ${payload.changed_files}` : "bootstrap completed";
    }

    if (event.type === "run.graph.materialized") {
      return `graph nodes=${payload.nodes ?? "0"} edges=${payload.edges ?? "0"}`;
    }

    if (event.type === "run.backpressure") {
      return `queue=${payload.queue_size ?? "?"} max_parallelism=${payload.max_parallelism ?? "?"}`;
    }

    if (event.type === "run.fallback.serial") {
      return `fallback reason: ${payload.reason ?? "unknown"}`;
    }

    if (event.type === "run.supervision.checkpoint") {
      const checkpoint = payload.checkpoint ?? "checkpoint";
      const reason = payload.reason ? ` (${payload.reason})` : "";
      return `${checkpoint}${reason}`;
    }

    if (event.type === "auth.codex.login.started") {
      return "codex login flow started";
    }

    if (event.type === "auth.codex.login.succeeded") {
      return payload.user ? `codex login succeeded (${payload.user})` : "codex login succeeded";
    }

    if (event.type === "auth.codex.login.failed") {
      return payload.error ?? "codex login failed";
    }

    if (event.type === "model.selected") {
      const model = payload.model_id ? ` (${payload.model_id})` : "";
      return `model provider ${payload.provider ?? "unknown"}${model}`;
    }

    if (event.type === "model.fallback.triggered") {
      return `fallback ${payload.from_provider ?? "?"} -> ${payload.to_provider ?? "?"} (${payload.error_class ?? "unknown"})`;
    }

    if (event.type === "model.fallback.exhausted") {
      return `fallback exhausted: ${payload.trail ?? "no trail"}`;
    }

    if (event.type === "context.compiled") {
      return `context bytes=${payload.used_bytes ?? "0"} budget=${payload.context_budget ?? "0"} sources=${payload.selected_sources ?? ""}`;
    }

    if (event.type === "skills.loaded") {
      return `skills loaded=${payload.loaded ?? "0"} errors=${payload.errors ?? "0"} mode=${payload.mode ?? "auto"}`;
    }

    if (event.type === "skill.selected") {
      return `${payload.name ?? "skill"} selected (${payload.scope ?? "unknown"})`;
    }

    if (event.type === "skill.invocation.blocked") {
      return `${payload.name ?? "skill"} blocked: ${payload.reason ?? "unknown"}`;
    }

    if (event.type === "skill.install.started" || event.type === "skill.create.started") {
      return `${event.type.includes("install") ? "install" : "create"} started ${payload.name ?? ""}`.trim();
    }

    if (event.type === "skill.install.succeeded" || event.type === "skill.create.succeeded") {
      return payload.message ?? "skill operation succeeded";
    }

    if (event.type === "skill.install.failed" || event.type === "skill.create.failed") {
      return payload.error ?? "skill operation failed";
    }

    if (event.type === "orchestrator.plan.generated") {
      return `plan ${payload.plan_id ?? ""} strategy=${payload.strategy_mode ?? "deterministic"} steps=${payload.steps ?? "0"}`.trim();
    }

    if (event.type === "orchestrator.step.started" || event.type === "orchestrator.step.completed") {
      return `${payload.step_name ?? payload.step_id ?? "step"} ${event.type.endsWith("started") ? "started" : "completed"}`;
    }

    if (event.type === "orchestrator.step.failed") {
      return `${payload.step_name ?? payload.step_id ?? "step"} failed: ${payload.error ?? "unknown"}`;
    }

    if (event.type === "orchestrator.resume.applied") {
      return `resumed from ${payload.step_id ?? "step"}`;
    }

    if (event.type === "orchestrator.decision.explained") {
      return `${payload.decision ?? "decision"} ${payload.reason ?? payload.selection ?? payload.result ?? ""}`.trim();
    }

    if (event.type === "eval.run.started") {
      return `eval run ${payload.eval_run_id ?? ""} started (candidate=${payload.candidate_strategy ?? "?"})`.trim();
    }

    if (event.type === "eval.run.completed") {
      return `eval completed score=${payload.composite_score ?? "?"} regression=${payload.regression_detected ?? "false"}`;
    }

    if (event.type === "eval.metric.regression.detected") {
      return `regression ${payload.metric ?? "metric"} candidate=${payload.candidate ?? "?"} baseline=${payload.baseline ?? "?"}`;
    }

    if (event.type === "optimizer.recommendation.generated") {
      return `recommendation ${payload.recommendation_id ?? ""} strategy=${payload.strategy_mode ?? "?"} gates=${payload.gate_pass_count ?? "0"}`.trim();
    }

    if (event.type === "optimizer.recommendation.applied") {
      return `applied ${payload.recommendation_id ?? ""} to run ${payload.orchestrator_run_id ?? "?"}`.trim();
    }

    if (event.type === "optimizer.recommendation.rejected") {
      return `rejected ${payload.recommendation_id ?? ""} (${payload.reason ?? "no reason"})`.trim();
    }

    if (event.type === "optimizer.validation.completed") {
      return `validation ${payload.status ?? "completed"} all_pass=${payload.all_pass ?? "false"}`;
    }

    if (event.type === "task.retry.scheduled") {
      return `${payload.task ?? "task"} retry attempt=${payload.attempt ?? "?"} in ${payload.backoff_seconds ?? "?"}s`;
    }

    if (event.type === "task.blocked") {
      return `${payload.task_id ?? "task"} waiting_deps=${payload.waiting_deps ?? "?"}`;
    }

    if (event.type === "task.ready") {
      return `${payload.task ?? "task"} is ready`;
    }

    return payload.summary ?? payload.error ?? "";
  }

  async function executeCommand(command: string) {
    if (command.startsWith("/run ")) {
      await createRun(command.replace("/run", "").trim());
      return;
    }

    if (command.startsWith("/orchestrate ")) {
      await createOrchestratorRun(command.replace("/orchestrate", "").trim());
      return;
    }

    if (command === "/eval-run") {
      await startEvalRun();
      return;
    }

    if (command === "/pause" && selectedRun) {
      await runAction("pause", selectedRun);
      return;
    }

    if (command === "/resume" && selectedRun) {
      await runAction("resume", selectedRun);
      return;
    }

    if (command === "/cancel" && selectedRun) {
      await runAction("cancel", selectedRun);
      return;
    }

    if (command === "/retry-failed") {
      const targetRunID = selectedRunObject?.id ?? selectedRun;
      const targetGoal = selectedRunObject?.goal ?? "selected run";
      if (targetRunID) {
        await createRun(`Retry failed tasks for run ${targetRunID}: ${targetGoal}`);
      }
    }
  }

  async function handleCommand() {
    const command = composer.trim();
    if (!command) {
      return;
    }
    await executeCommand(command);
    setComposer("");
  }

  function handleQuickCommand(command: string) {
    setComposer(command);
    void executeCommand(command).finally(() => {
      setComposer("");
    });
  }

  return (
    <div className="app-shell" data-testid="app-layout">
      <div className="background-shape shape-1" aria-hidden="true" />
      <div className="background-shape shape-2" aria-hidden="true" />

      <aside className="column sidebar-left">
        <section className="panel brand-panel">
          <div className="brand-mark">B3</div>
          <div>
            <h1>BIOMETRICS</h1>
            <p>Autonomous Engineering Control Plane</p>
          </div>
        </section>

        <section className="panel quick-stats">
          <div className="stat-item">
            <span>Runs</span>
            <strong>{runStats.total}</strong>
          </div>
          <div className="stat-item">
            <span>Running</span>
            <strong>{runStats.running}</strong>
          </div>
          <div className="stat-item">
            <span>Completed</span>
            <strong>{runStats.completed}</strong>
          </div>
          <div className="stat-item">
            <span>Failed</span>
            <strong>{runStats.failed}</strong>
          </div>
        </section>

        <section className="panel">
          <HeaderTitle icon="project">Project</HeaderTitle>
          <select value={selectedProject} onChange={(e) => setSelectedProject(e.target.value)}>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
        </section>

        <section className="panel">
          <HeaderTitle icon="scheduler">Scheduler</HeaderTitle>
          <div className="panel-head">
            <span className="chip">{codexAuthReady ? "codex ready" : "codex login required"}</span>
            <button type="button" onClick={() => void codexLogin()}>
              Codex Login
            </button>
          </div>
          <label className="field-stack">
            <span>Run Mode</span>
            <select data-testid="run-mode-select" value={runMode} onChange={(e) => setRunMode(e.target.value as RunMode)}>
              <option value="autonomous">autonomous</option>
              <option value="supervised">supervised</option>
            </select>
          </label>

          <label className="field-stack">
            <span>Mode</span>
            <select value={schedulerMode} onChange={(e) => setSchedulerMode(e.target.value as SchedulerMode)}>
              <option value="dag_parallel_v1">dag_parallel_v1</option>
              <option value="serial">serial</option>
            </select>
          </label>

          <label className="field-stack">
            <span>Max Parallelism</span>
            <input
              type="number"
              min={1}
              max={32}
              value={schedulerMode === "serial" ? 1 : maxParallelism}
              disabled={schedulerMode === "serial"}
              onChange={(e) => setMaxParallelism(Number(e.target.value) || 1)}
            />
          </label>

          <label className="field-stack">
            <span>Primary Model Provider</span>
            <select value={modelPreference} onChange={(e) => setModelPreference(e.target.value)}>
              {(modelCatalog?.providers ?? []).map((provider) => (
                <option key={provider.id} value={provider.id}>
                  {provider.id} ({provider.status})
                </option>
              ))}
              {!modelCatalog && <option value="codex">codex</option>}
            </select>
          </label>

          <label className="field-stack">
            <span>Fallback Chain (comma-separated)</span>
            <input value={fallbackChainInput} onChange={(e) => setFallbackChainInput(e.target.value)} />
          </label>

          <label className="field-stack">
            <span>Model ID (optional)</span>
            <input value={modelID} onChange={(e) => setModelID(e.target.value)} />
          </label>

          <label className="field-stack">
            <span>Context Budget</span>
            <input
              type="number"
              min={1000}
              max={200000}
              value={contextBudget}
              onChange={(e) => setContextBudget(Number(e.target.value) || 24000)}
            />
          </label>
        </section>

        <section className="panel">
          <HeaderTitle icon="skills">Skills</HeaderTitle>
          <label className="field-stack">
            <span>Selection Mode</span>
            <select
              value={skillSelectionMode}
              onChange={(e) => setSkillSelectionMode(e.target.value as SkillSelectionMode)}
            >
              <option value="auto">auto</option>
              <option value="explicit">explicit</option>
              <option value="off">off</option>
            </select>
          </label>
          <div className="module-grid">
            {skillsCatalog.map((skill) => (
              <label key={skill.name} className="toggle-row module-row">
                <input
                  type="checkbox"
                  checked={selectedSkills.includes(skill.name)}
                  disabled={!skill.enabled || skillSelectionMode === "off"}
                  onChange={() => toggleSkill(skill.name)}
                />
                <span>{skill.name}</span>
              </label>
            ))}
          </div>
        </section>

        <section className="panel">
          <HeaderTitle icon="blueprint">Blueprint</HeaderTitle>
          <select
            value={selectedBlueprintProfile}
            onChange={(e) => {
              setSelectedBlueprintProfile(e.target.value);
              setSelectedBlueprintModules([]);
            }}
          >
            <option value="">No profile</option>
            {blueprints.map((profile) => (
              <option key={profile.id} value={profile.id}>
                {profile.name} ({profile.version})
              </option>
            ))}
          </select>

          <label className="toggle-row">
            <input
              type="checkbox"
              checked={enableBootstrap}
              onChange={(e) => setEnableBootstrap(e.target.checked)}
              disabled={!selectedBlueprintProfile}
            />
            <span>Bootstrap on run start</span>
          </label>

          <div className="module-grid">
            {selectedProfile?.modules.map((module) => (
              <label key={module.id} className="toggle-row module-row">
                <input
                  type="checkbox"
                  checked={selectedBlueprintModules.includes(module.id)}
                  onChange={() => toggleModule(module.id)}
                />
                <span>{module.name}</span>
              </label>
            ))}
          </div>
        </section>

        <section className="panel grow">
          <div className="panel-head">
            <HeaderTitle icon="runs">Runs</HeaderTitle>
            <span className="chip">{runs.length}</span>
          </div>
          <div className="run-list" data-testid="runs-list">
            {runs.map((run) => {
              const statusClass = normalizeStatusClass(run.status);
              return (
                <button
                  key={run.id}
                  data-testid={`run-${run.id}`}
                  className={`run-card ${selectedRun === run.id ? "active" : ""}`}
                  onClick={() => setSelectedRun(run.id)}
                >
                  <div className="run-card-head">
                    <span className="mono">{run.project_id}</span>
                    <span className={`status-pill ${statusClass}`}>{run.status}</span>
                  </div>
                  <p>{run.goal}</p>
                  <small>
                    {run.mode} · {run.scheduler_mode ?? "dag_parallel_v1"} @ {run.max_parallelism ?? 8}
                  </small>
                  <small>
                    model: {run.model_preference ?? "codex"}
                    {run.model_id ? ` (${run.model_id})` : ""}
                    {run.fallback_chain && run.fallback_chain.length > 0 ? ` -> ${run.fallback_chain.join(" -> ")}` : ""}
                  </small>
                  <small>
                    skills: {run.skill_selection_mode ?? "auto"}
                    {run.skills && run.skills.length > 0 ? ` (${run.skills.join(", ")})` : ""}
                  </small>
                  {run.blueprint_profile && <small>Blueprint: {run.blueprint_profile}</small>}
                </button>
              );
            })}
          </div>
        </section>
      </aside>

      <main className="column center-column">
        <section className="panel timeline-panel grow">
          <div className="panel-head">
            <HeaderTitle icon="timeline">Run Timeline</HeaderTitle>
            <span className="chip">{events.length}</span>
          </div>

          {selectedRunObject && (
            <div className="run-hero">
              <div className="run-hero-head">
                <span className="mono">run {selectedRunObject.id.slice(0, 8)}</span>
                <span className={`status-pill ${normalizeStatusClass(selectedRunObject.status)}`}>
                  {selectedRunObject.status}
                </span>
              </div>
              <p>{selectedRunObject.goal}</p>
              <div className="run-hero-meta">
                <small>
                  {selectedRunObject.mode} · {selectedRunObject.scheduler_mode ?? "dag_parallel_v1"} @{" "}
                  {selectedRunObject.max_parallelism ?? 8}
                </small>
                <small>
                  model: {selectedRunObject.model_preference ?? "codex"}
                  {selectedRunObject.model_id ? ` (${selectedRunObject.model_id})` : ""}
                  {selectedRunObject.fallback_chain && selectedRunObject.fallback_chain.length > 0
                    ? ` -> ${selectedRunObject.fallback_chain.join(" -> ")}`
                    : ""}
                </small>
                <small>
                  skills: {selectedRunObject.skill_selection_mode ?? "auto"}
                  {selectedRunObject.skills && selectedRunObject.skills.length > 0
                    ? ` (${selectedRunObject.skills.join(", ")})`
                    : ""}
                </small>
                <small>project: {selectedRunObject.project_id}</small>
                {selectedRunObject.blueprint_profile && <small>blueprint: {selectedRunObject.blueprint_profile}</small>}
                <small>created: {formatTime(selectedRunObject.created_at)}</small>
              </div>
            </div>
          )}

          {selectedRunObject?.fallback_triggered && (
            <div className="fallback-banner" data-testid="fallback-banner">
              Serial fallback active for this run
            </div>
          )}
          {supervisionCheckpointEvent && (
            <div className="supervision-banner" data-testid="supervision-banner">
              Supervision checkpoint: {supervisionCheckpointEvent.payload?.checkpoint ?? "checkpoint"}.
              {supervisionCheckpointEvent.payload?.reason ? ` ${supervisionCheckpointEvent.payload.reason}.` : ""} Use
              resume to continue.
            </div>
          )}

          <div className="events" data-testid="events-list">
            {visibleEvents.map((event) => (
              <article key={event.id} className={`event-row tone-${eventTone(event.type)}`}>
                <div className="event-head">
                  <code>{formatTime(event.created_at)}</code>
                  <strong>{event.type}</strong>
                  <span>{event.source}</span>
                </div>
                <p>{describeEvent(event)}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="panel composer-panel">
          <div className="composer-main">
            <input
              data-testid="composer-input"
              value={composer}
              onChange={(e) => setComposer(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  void handleCommand();
                }
              }}
              placeholder="/run ... | /orchestrate ... | /eval-run | /pause | /resume | /cancel | /retry-failed"
            />
            <button data-testid="composer-send" onClick={() => void handleCommand()}>
              <span className="button-inline">
                <AppIcon name="command" />
                Execute
              </span>
            </button>
          </div>
          <div className="command-rail">
            {QUICK_COMMANDS.map((command) => (
              <button
                key={command.label}
                type="button"
                className="ghost-command"
                onClick={() => handleQuickCommand(command.value)}
              >
                <span className="button-inline">
                  <AppIcon name={command.icon} />
                  {command.label}
                </span>
              </button>
            ))}
          </div>
        </section>
      </main>

      <aside className="column sidebar-right">
        <section className="panel emergency-panel">
          <div className="panel-head">
            <HeaderTitle icon="emergency">Emergency Controls</HeaderTitle>
            <span className="chip warn">manual</span>
          </div>

          <div className="task-snapshot">
            <div>
              <span>tasks</span>
              <strong>{taskStats.total}</strong>
            </div>
            <div>
              <span>running</span>
              <strong>{taskStats.running}</strong>
            </div>
            <div>
              <span>completed</span>
              <strong>{taskStats.completed}</strong>
            </div>
            <div>
              <span>blocked</span>
              <strong>{taskStats.blocked}</strong>
            </div>
          </div>

          <div className="actions">
            <button data-testid="action-pause" onClick={() => selectedRun && void runAction("pause", selectedRun)}>
              <span className="button-inline">
                <AppIcon name="pause" />
                Pause
              </span>
            </button>
            <button data-testid="action-resume" onClick={() => selectedRun && void runAction("resume", selectedRun)}>
              <span className="button-inline">
                <AppIcon name="resume" />
                Resume
              </span>
            </button>
            <button
              data-testid="action-cancel"
              className="danger"
              onClick={() => selectedRun && void runAction("cancel", selectedRun)}
            >
              <span className="button-inline">
                <AppIcon name="cancel" />
                Cancel
              </span>
            </button>
          </div>
        </section>

        <section className="panel orchestrator-panel">
          <div className="panel-head">
            <HeaderTitle icon="orchestrator">Orchestrator Control</HeaderTitle>
            <span className="chip">{orchestratorCapabilities ? "ready" : "offline"}</span>
          </div>

          <label className="field-stack">
            <span>Strategy Mode</span>
            <select value={strategyMode} onChange={(e) => setStrategyMode(e.target.value as StrategyMode)}>
              {(orchestratorCapabilities?.strategy_modes ?? ["deterministic", "adaptive", "arena"]).map((mode) => (
                <option key={mode} value={mode}>
                  {mode}
                </option>
              ))}
            </select>
          </label>

          <label className="field-stack">
            <span>Policy Preset</span>
            <select value={policyPreset} onChange={(e) => setPolicyPreset(e.target.value as PolicyPreset)}>
              {(orchestratorCapabilities?.policy_presets ?? ["strict", "balanced", "velocity"]).map((preset) => (
                <option key={preset} value={preset}>
                  {preset}
                </option>
              ))}
            </select>
          </label>

          <div className="orchestrator-objective-grid">
            <label className="field-stack">
              <span>Quality</span>
              <input type="number" min={0} step={0.05} max={1} value={objectiveQuality} onChange={(e) => setObjectiveQuality(Number(e.target.value) || 0)} />
            </label>
            <label className="field-stack">
              <span>Speed</span>
              <input type="number" min={0} step={0.05} max={1} value={objectiveSpeed} onChange={(e) => setObjectiveSpeed(Number(e.target.value) || 0)} />
            </label>
            <label className="field-stack">
              <span>Cost</span>
              <input type="number" min={0} step={0.05} max={1} value={objectiveCost} onChange={(e) => setObjectiveCost(Number(e.target.value) || 0)} />
            </label>
          </div>

          <div className="actions">
            <button type="button" onClick={() => void createOrchestratorRun(composer.trim() || "Execute apex orchestration objective")}>
              <span className="button-inline">
                <AppIcon name="orchestrator" />
                Start Apex Run
              </span>
            </button>
            <button type="button" onClick={() => void startEvalRun()}>
              <span className="button-inline">
                <AppIcon name="score" />
                Run Eval
              </span>
            </button>
            {orchestratorRun && (
              <button type="button" onClick={() => void resumeOrchestratorFromExecute(orchestratorRun.id)}>
                <span className="button-inline">
                  <AppIcon name="resume" />
                  Resume Execute
                </span>
              </button>
            )}
          </div>

          {orchestratorRun && (
            <div className="orchestrator-status">
              <div className="orchestrator-status-head">
                <strong>{orchestratorRun.strategy_mode}</strong>
                <span className={`status-pill ${normalizeStatusClass(orchestratorRun.status)}`}>{orchestratorRun.status}</span>
              </div>
              <small className="mono">{orchestratorRun.id}</small>
              <small>current step: {orchestratorRun.current_step_id ?? "n/a"}</small>
              <small>underlying run: {orchestratorRun.underlying_run_id ?? "n/a"}</small>
              <div className="orchestrator-step-list">
                {orchestratorRun.steps.map((step) => (
                  <div key={step.id} className="orchestrator-step-row">
                    <span>{step.name}</span>
                    <span className={`status-pill ${normalizeStatusClass(step.status)}`}>{step.status}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {orchestratorScorecard && (
            <div className="orchestrator-scorecard">
              <div className="orchestrator-score-row">
                <span>quality</span>
                <strong>{orchestratorScorecard.quality_score.toFixed(3)}</strong>
              </div>
              <div className="orchestrator-score-row">
                <span>time to green</span>
                <strong>{Math.round(orchestratorScorecard.median_time_to_green_seconds)}s</strong>
              </div>
              <div className="orchestrator-score-row">
                <span>cost / success</span>
                <strong>{orchestratorScorecard.cost_per_success.toFixed(5)}</strong>
              </div>
              <div className="orchestrator-score-row">
                <span>composite</span>
                <strong>{orchestratorScorecard.composite_score.toFixed(3)}</strong>
              </div>
            </div>
          )}

          {evalRun && (
            <div className="orchestrator-status">
              <div className="orchestrator-status-head">
                <strong>eval {evalRun.name}</strong>
                <span className={`status-pill ${normalizeStatusClass(evalRun.status)}`}>{evalRun.status}</span>
              </div>
              <small>
                {evalRun.candidate_strategy_mode} vs {evalRun.baseline_strategy_mode} · n={evalRun.sample_size}
              </small>
              <small>regression: {evalRun.regression_detected ? "yes" : "no"}</small>
            </div>
          )}

          {leaderboard.length > 0 && (
            <div className="orchestrator-leaderboard">
              {leaderboard.slice(0, 3).map((entry, index) => (
                <div key={entry.strategy + index} className="orchestrator-score-row">
                  <span>
                    {index + 1}. {entry.strategy}
                  </span>
                  <strong>{entry.composite_score.toFixed(3)}</strong>
                </div>
              ))}
            </div>
          )}

          <div className="orchestrator-status" data-testid="optimizer-panel">
            <div className="orchestrator-status-head">
              <strong>Apex Optimizer</strong>
              <span className="chip">{optimizerRecommendations.length}</span>
            </div>
            <small>Shadow mode: recommendations are generated first, apply is always manual.</small>
            <div className="actions">
              <button type="button" disabled={optimizerBusy} onClick={() => void generateOptimizerRecommendation()}>
                <span className="button-inline">
                  <AppIcon name="score" />
                  Generate Recommendation
                </span>
              </button>
              {selectedOptimizerRecommendation?.status === "generated" && (
                <button
                  type="button"
                  disabled={optimizerBusy}
                  onClick={() => void applyOptimizerRecommendation(selectedOptimizerRecommendation.id)}
                >
                  <span className="button-inline">
                    <AppIcon name="play" />
                    Apply & Start
                  </span>
                </button>
              )}
              {selectedOptimizerRecommendation?.status === "generated" && (
                <button
                  type="button"
                  className="danger"
                  disabled={optimizerBusy}
                  onClick={() => void rejectOptimizerRecommendation(selectedOptimizerRecommendation.id)}
                >
                  <span className="button-inline">
                    <AppIcon name="cancel" />
                    Reject
                  </span>
                </button>
              )}
            </div>

            {selectedOptimizerRecommendation && (
              <>
                <small className="mono">{selectedOptimizerRecommendation.id}</small>
                <small>
                  {selectedOptimizerRecommendation.strategy_mode} · {selectedOptimizerRecommendation.scheduler_mode} @{" "}
                  {selectedOptimizerRecommendation.max_parallelism}
                </small>
                <small>confidence: {selectedOptimizerRecommendation.confidence}</small>
                <div className="optimizer-gate-chips">
                  <span
                    data-testid="optimizer-gate-quality"
                    className={`status-pill ${selectedOptimizerRecommendation.predicted_gates.quality_pass ? "completed" : "failed"}`}
                  >
                    quality {selectedOptimizerRecommendation.predicted_gates.quality_pass ? "PASS" : "FAIL"}
                  </span>
                  <span
                    data-testid="optimizer-gate-time"
                    className={`status-pill ${selectedOptimizerRecommendation.predicted_gates.time_pass ? "completed" : "failed"}`}
                  >
                    time {selectedOptimizerRecommendation.predicted_gates.time_pass ? "PASS" : "FAIL"}
                  </span>
                  <span
                    data-testid="optimizer-gate-cost"
                    className={`status-pill ${selectedOptimizerRecommendation.predicted_gates.cost_pass ? "completed" : "failed"}`}
                  >
                    cost {selectedOptimizerRecommendation.predicted_gates.cost_pass ? "PASS" : "FAIL"}
                  </span>
                  <span
                    data-testid="optimizer-gate-regression"
                    className={`status-pill ${selectedOptimizerRecommendation.predicted_gates.regression_pass ? "completed" : "failed"}`}
                  >
                    regression {selectedOptimizerRecommendation.predicted_gates.regression_pass ? "PASS" : "FAIL"}
                  </span>
                </div>
                <small>
                  gates: {selectedOptimizerRecommendation.predicted_gates.gate_pass_count}/4 · all_pass=
                  {String(selectedOptimizerRecommendation.predicted_gates.all_pass)}
                </small>
                <small>
                  predicted q={selectedOptimizerRecommendation.predicted_gates.predicted_quality_score.toFixed(3)} · time=
                  {selectedOptimizerRecommendation.predicted_gates.predicted_time_improvement_percent.toFixed(2)}% · cost=
                  {selectedOptimizerRecommendation.predicted_gates.predicted_cost_improvement_percent.toFixed(2)}% · cps=
                  {(selectedOptimizerRecommendation.predicted_gates.predicted_cost_per_success ?? 0).toFixed(5)}
                </small>
                <small>{selectedOptimizerRecommendation.rationale}</small>
                {selectedOptimizerRecommendation.validation && (
                  <small>
                    validation: {selectedOptimizerRecommendation.validation.status} · all_pass=
                    {String(selectedOptimizerRecommendation.validation.all_pass)}
                  </small>
                )}
              </>
            )}

            {optimizerRecommendations.length > 0 && (
              <div className="orchestrator-step-list">
                {optimizerRecommendations.slice(0, 3).map((rec) => (
                  <button key={rec.id} type="button" className="ghost-command" onClick={() => setOptimizerActiveID(rec.id)}>
                    <span className="button-inline">
                      <span>{rec.strategy_mode}</span>
                      <span className={`status-pill ${normalizeStatusClass(rec.status)}`}>{rec.status}</span>
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </section>

        <section className="panel tabs-panel grow">
          <div className="tab-buttons">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                data-testid={`tab-${tab.id}`}
                className={tab.id === activeTab ? "active" : ""}
                onClick={() => setActiveTab(tab.id)}
              >
                <span className="button-inline">
                  <AppIcon name={tab.icon} />
                  {tab.label}
                </span>
              </button>
            ))}
          </div>

          {activeTab === "tasks" && (
            <div className="tab-content">
              {tasks.map((task) => {
                const statusClass = normalizeStatusClass(task.status);
                return (
                  <div className="task-row" key={task.id}>
                    <strong>{task.name}</strong>
                    <span>{task.agent}</span>
                    <span className={`status-pill ${statusClass}`}>{task.status}</span>
                    <small>attempt {task.attempt}</small>
                    <small className="task-meta">state: {task.lifecycle_state ?? "n/a"}</small>
                  </div>
                );
              })}
            </div>
          )}

          {activeTab === "graph" && (
            <div className="tab-content graph" data-testid="graph-tab">
              {!graph && <p>Graph not available yet for this run.</p>}
              {graph && (
                <>
                  <div className="graph-header" data-testid="graph-header">
                    <small>nodes: {graph.nodes.length}</small>
                    <small>edges: {graph.edges.length}</small>
                    <small>critical path: {graph.critical_path?.length ?? 0}</small>
                  </div>

                  <div className="graph-filter">
                    <label>
                      Filter
                      <select value={graphFilter} onChange={(e) => setGraphFilter(e.target.value as GraphFilter)}>
                        <option value="all">all</option>
                        <option value="failed">failed</option>
                        <option value="retrying">retrying</option>
                        <option value="blocked">blocked</option>
                      </select>
                    </label>
                  </div>

                  <div className="graph-list" data-testid="graph-list">
                    {graphNodes.map((node) => {
                      const statusClass = normalizeStatusClass(node.status);
                      return (
                        <div key={node.id} className="graph-node" data-testid={`graph-node-${node.id}`}>
                          <div className="graph-node-head">
                            <strong>{node.name}</strong>
                            <span className={`status-pill ${statusClass}`}>{node.status}</span>
                          </div>
                          <small>
                            {node.agent} | {node.lifecycle_state} | p{node.priority}
                          </small>
                          <small>depends_on: {(node.depends_on ?? []).join(", ") || "none"}</small>
                        </div>
                      );
                    })}
                  </div>
                </>
              )}
            </div>
          )}

          {activeTab === "files" && (
            <div className="tab-content files" data-testid="files-tab">
              <div className="file-controls">
                <input data-testid="file-path-input" value={filePath} onChange={(e) => setFilePath(e.target.value)} />
                <button data-testid="file-path-open" onClick={() => void loadFiles(filePath)}>
                  Open
                </button>
              </div>

              {filesError && (
                <p className="error-text" data-testid="files-error">
                  {filesError}
                </p>
              )}

              <div className="file-list">
                {files.map((entry) => (
                  <button
                    key={`${entry.path}-${entry.name}`}
                    onClick={() => {
                      if (entry.isDir) {
                        setFilePath(entry.path);
                      } else {
                        void openFile(entry.path);
                      }
                    }}
                  >
                    {entry.isDir ? "[DIR]" : "[FILE]"} {entry.name}
                  </button>
                ))}
              </div>

              <pre className="file-content">{fileContent}</pre>
            </div>
          )}

          {activeTab === "diff" && (
            <div className="tab-content">
              <p>
                Diff stream is shown via event type <code>diff.produced</code>.
              </p>
            </div>
          )}

          {activeTab === "terminal" && (
            <div className="tab-content">
              <p>Terminal output is available in timeline summaries for agent steps.</p>
            </div>
          )}
        </section>
      </aside>
    </div>
  );
}
