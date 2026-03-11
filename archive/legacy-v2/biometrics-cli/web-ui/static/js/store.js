// @ts-check
document.addEventListener('alpine:init', () => {
    Alpine.data('biometricsStore', () => ({
        projectId: 'biometrics',
        currentView: 'explorer', // explorer, swarm, docker, config
        activeTab: 'dashboard', // dashboard, or filename
        files: [],
        currentPath: '.',
        editorContent: '',
        
        delqhiLoop: false,
        swarmEngine: true,
        activeTasksCount: 0,
        totalTasksCount: 0,
        version: 'v1.0.0',
        wsStatus: 'Connecting...',
        wsIndicatorClass: 'bg-gray-500',
        
        tasks: [],
        logs: [],
        projects: [],
        maxLogs: 100,
        
        // Chat
        chatMessage: '',
        chatHistory: [
            { role: 'assistant', text: 'Ready for Enterprise Orchestration. How can I assist you today?' }
        ],

        async init() {
            await this.fetchProjects();
            await this.fetchConfig();
            await this.fetchStatus();
            await this.fetchTasks();
            await this.fetchFiles();
            
            this.connectWebSocket();

            setInterval(() => this.fetchStatus(), 5000);
            setInterval(() => this.fetchTasks(), 10000);
        },

        async fetchProjects() {
            try {
                const res = await fetch('/api/projects');
                if (res.ok) this.projects = await res.json();
            } catch (e) {}
        },

        async fetchFiles(path = '.') {
            try {
                const res = await fetch(`/api/files/list?path=${encodeURIComponent(path)}`);
                if (res.ok) {
                    this.files = await res.json();
                    this.currentPath = path;
                }
            } catch (e) {}
        },

        async openFile(file) {
            if (file.isDir) {
                await this.fetchFiles(file.path);
            } else {
                try {
                    const res = await fetch(`/api/files/read?path=${encodeURIComponent(file.path)}`);
                    if (res.ok) {
                        this.editorContent = await res.text();
                        this.activeTab = file.name;
                        this.currentView = 'editor';
                    }
                } catch (e) {}
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
            } catch (e) {}
        },

        async fetchStatus() {
            try {
                const res = await fetch('/api/status');
                if (res.ok) {
                    const status = await res.json();
                    this.activeTasksCount = status.active_tasks || 0;
                    this.totalTasksCount = status.total_tasks || 0;
                }
            } catch (e) {}
        },

        async fetchTasks() {
            try {
                const res = await fetch('/api/tasks/list');
                if (res.ok) this.tasks = await res.json();
            } catch (e) {}
        },

        connectWebSocket() {
            const loc = window.location;
            const wsUri = (loc.protocol === "https:" ? "wss:" : "ws:") + "//" + loc.host + "/ws";
            const ws = new WebSocket(wsUri);

            ws.onopen = () => {
                this.wsStatus = 'Link Active';
                this.wsIndicatorClass = 'bg-neon shadow-[0_0_8px_#04B575]';
                this.addLog('SYSTEM', 'BIOMETRICS Link established.', 'success');
            };

            ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === 'update') {
                        this.addLog('ORCHESTRATOR', data.message, 'info');
                        this.fetchTasks();
                        this.fetchStatus();
                    } else if (data.type === 'log') {
                        this.addLog(data.agent || 'AGENT', data.message, data.level || 'info');
                    }
                } catch(e) {}
            };

            ws.onclose = () => {
                this.wsStatus = 'Offline';
                this.wsIndicatorClass = 'bg-alert shadow-[0_0_8px_#F56565]';
                setTimeout(() => this.connectWebSocket(), 3000);
            };
        },

        addLog(source, message, type = 'info') {
            const time = new Date().toLocaleTimeString('en-US', {hour12: false, hour: '2-digit', minute:'2-digit', second:'2-digit'});
            this.logs.push({ id: Math.random(), time, source, message, type });
            if (this.logs.length > this.maxLogs) this.logs.shift();
            this.$nextTick(() => {
                const el = document.getElementById('log-stream');
                if(el) el.scrollTop = el.scrollHeight;
            });
        },

        async sendMessage() {
            if (!this.chatMessage.trim()) return;
            const msg = this.chatMessage;
            this.chatHistory.push({ role: 'user', text: msg });
            this.chatMessage = '';
            
            // Mock AI response for now or call real chat API if exists
            this.addLog('USER', `Chat: ${msg}`, 'info');
            setTimeout(() => {
                this.chatHistory.push({ role: 'assistant', text: 'Processing command through BIOMETRICS swarm...' });
            }, 500);
        },

        async toggleDelqhiLoop() {
            const enabled = !this.delqhiLoop;
            await fetch('/api/config/toggle-delqhi-loop', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enabled })
            });
            this.delqhiLoop = enabled;
        },

        async toggleSwarmEngine() {
            const enabled = !this.swarmEngine;
            await fetch('/api/config/toggle-swarm-engine', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ enabled })
            });
            this.swarmEngine = enabled;
        }
    }));
});
