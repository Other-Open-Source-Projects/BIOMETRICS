// @ts-check
/**
 * @typedef {Object} GlobalState
 * @property {string} projectId
 * @property {boolean} delqhiLoop
 * @property {boolean} swarmEngine
 * @property {number} activeTasks
 * @property {number} totalTasks
 * @property {string} version
 * @property {string} wsStatus
 * @property {Array<Object>} tasks
 * @property {Array<Object>} logs
 * @property {Array<Object>} projects
 * @property {number} maxLogs
 */

document.addEventListener('alpine:init', () => {
    Alpine.data('biometricsStore', () => ({
        projectId: 'biometrics',
        delqhiLoop: false,
        swarmEngine: true,
        activeTasks: 0,
        totalTasks: 0,
        version: 'Loading...',
        wsStatus: 'Connecting...',
        wsIndicatorClass: 'bg-gray-500',
        tasks: [],
        logs: [],
        projects: [],
        maxLogs: 100, // Virtualized limit

        /** @type {WebSocket|null} */
        ws: null,

        async init() {
            await this.fetchProjects();
            await this.fetchConfig();
            await this.fetchStatus();
            await this.fetchTasks();
            
            this.connectWebSocket();

            // Setup polling for fallbacks
            setInterval(() => this.fetchStatus(), 5000);
            setInterval(() => this.fetchTasks(), 10000);
        },

        async fetchProjects() {
            try {
                const res = await fetch('/api/projects');
                if (res.ok) this.projects = await res.json();
            } catch (e) {
                console.error('Failed to fetch projects', e);
            }
        },

        async fetchConfig() {
            try {
                const res = await fetch('/api/config');
                if (res.ok) {
                    const cfg = await res.json();
                    this.delqhiLoop = !!cfg.delqhi_loop;
                    this.swarmEngine = !!cfg.swarm_engine;
                }
            } catch (e) {
                console.error('Failed to fetch config', e);
            }
        },

        async fetchStatus() {
            try {
                const res = await fetch('/api/status');
                if (res.ok) {
                    const status = await res.json();
                    this.activeTasks = status.active_tasks || 0;
                    this.totalTasks = status.total_tasks || 0;
                    this.version = status.version || 'v1.0.0';
                }
            } catch (e) {
                // Ignore
            }
        },

        async fetchTasks() {
            try {
                const res = await fetch('/api/tasks/list');
                if (res.ok) {
                    this.tasks = await res.json();
                }
            } catch (e) {
                // Ignore
            }
        },

        connectWebSocket() {
            const loc = window.location;
            const wsUri = (loc.protocol === "https:" ? "wss:" : "ws:") + "//" + loc.host + "/ws";
            this.ws = new WebSocket(wsUri);

            this.ws.onopen = () => {
                this.wsStatus = 'Swarm Link Active';
                this.wsIndicatorClass = 'bg-neon pulse-green';
                this.addLog('SYSTEM', 'WebSocket connection established. Swarm protocol ready.', 'success');
                // State Sync (Mandate 0.37)
                this.fetchTasks();
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === 'update') {
                        this.addLog('ORCHESTRATOR', data.message, 'info');
                        this.fetchTasks();
                    } else if (data.type === 'log') {
                        this.addLog(data.agent || 'AGENT', data.message, data.level || 'info');
                    }
                } catch(e) {
                    this.addLog('RAW', event.data, 'info');
                }
            };

            this.ws.onclose = () => {
                this.wsStatus = 'Link Offline';
                this.wsIndicatorClass = 'bg-alert';
                this.addLog('SYSTEM', 'Connection lost. Re-establishing link...', 'error');
                setTimeout(() => this.connectWebSocket(), 3000);
            };
        },

        addLog(source, message, type = 'info') {
            const time = new Date().toLocaleTimeString('en-US', {hour12: false, hour: '2-digit', minute:'2-digit', second:'2-digit'});
            
            // Add TraceID visualization randomly for effect or real if provided
            const traceId = 'tr_' + Math.random().toString(36).substr(2, 6);

            this.logs.push({
                id: Date.now() + Math.random(),
                time,
                source,
                message,
                type,
                traceId
            });

            // Keep array bounded for virtualized rendering concept
            if (this.logs.length > this.maxLogs) {
                this.logs.shift();
            }

            // Auto-scroll
            this.$nextTick(() => {
                const el = document.getElementById('terminal-logs');
                if(el) el.scrollTop = el.scrollHeight;
            });
        },

        async toggleDelqhiLoop(e) {
            const enabled = e.target.checked;
            try {
                await fetch('/api/config/toggle-delqhi-loop', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled })
                });
                this.delqhiLoop = enabled;
                this.addLog('SYSTEM', `Delqhi-Loop Mode ${enabled ? 'ENABLED' : 'DISABLED'} (Mandate 0.36)`, enabled ? 'success' : 'warn');
            } catch (err) {
                e.target.checked = !enabled; // Revert
                this.addLog('SYSTEM', 'Failed to toggle Delqhi-Loop', 'error');
            }
        },

        async toggleSwarmEngine(e) {
            const enabled = e.target.checked;
            try {
                await fetch('/api/config/toggle-swarm-engine', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled })
                });
                this.swarmEngine = enabled;
                this.addLog('SYSTEM', `Swarm Engine ${enabled ? 'ENABLED' : 'DISABLED'} (Mandate 0.11)`, enabled ? 'success' : 'warn');
            } catch (err) {
                e.target.checked = !enabled; // Revert
                this.addLog('SYSTEM', 'Failed to toggle Swarm Engine', 'error');
            }
        },

        async dispatchTask() {
            const title = document.getElementById('task-title').value;
            const description = document.getElementById('task-description').value;
            const agent = document.getElementById('task-agent').value;

            this.addLog('USER', `Dispatching Task: ${title} to ${agent} [${this.projectId}]`, 'success');

            try {
                const res = await fetch('/api/tasks/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title, description, agent, project: this.projectId })
                });

                if(!res.ok) throw new Error('API Offline - Task dispatch failed');
                
                const task = await res.json();
                this.addLog('ORCHESTRATOR', `Task registered: ${task.id}`, 'info');
                
                await fetch('/api/tasks/execute', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ task_id: task.id })
                });

                document.getElementById('task-form').reset();
                this.fetchTasks();
                this.fetchStatus();
            } catch (err) {
                this.addLog('SYSTEM', err.message, 'error');
            }
        },

        getColorClass(type) {
            if (type === 'error') return 'text-alert';
            if (type === 'success') return 'text-neon';
            if (type === 'warn') return 'text-yellow-400';
            return 'text-gray-300';
        },

        getSourceColorClass(source) {
            if(source === 'SYSTEM') return 'text-purple-400';
            if(source === 'ORCHESTRATOR') return 'text-blue-400';
            if(source === 'USER') return 'text-neon';
            if(source === 'AGENT' || source.toLowerCase().includes('sisyphus')) return 'text-orange-400';
            return 'text-gray-400';
        },

        getTaskStatusClass(status) {
            if(status === 'running' || status === 'in_progress') return 'border-yellow-500 bg-yellow-500 bg-opacity-10';
            if(status === 'completed' || status === 'done') return 'border-neon bg-gray-800 bg-opacity-50';
            if(status === 'failed') return 'border-alert bg-alert bg-opacity-10';
            return 'border-gray-600 bg-gray-800 bg-opacity-50';
        },

        getTaskIconClass(status) {
            if(status === 'running' || status === 'in_progress') return 'fa-circle-notch fa-spin text-yellow-500';
            if(status === 'completed' || status === 'done') return 'fa-check text-neon';
            if(status === 'failed') return 'fa-triangle-exclamation text-alert';
            return 'fa-clock text-gray-400';
        },

        clearLogs() {
            this.logs = [];
        }
    }));
});
