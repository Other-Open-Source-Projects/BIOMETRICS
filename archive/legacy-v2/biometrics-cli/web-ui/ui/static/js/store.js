document.addEventListener('alpine:init', () => {
    Alpine.store('biometrics', {
        activeTab: 'dashboard',
        currentView: 'swarm',
        files: [],
        tasks: [],
        logs: [],
        chatHistory: [
            { role: 'assistant', text: 'Swarm Engine ready. How can I help you today?' }
        ],
        chatMessage: '',
        wsStatus: 'Disconnected',
        wsIndicatorClass: 'text-red-500',
        version: 'v1.0.0-enterprise',
        editorContent: 'Select a file to view...',
        activeTasksCount: 0,
        totalTasksCount: 0,
        delqhiLoop: false,
        swarmEngine: false,

        async init() {
            this.fetchFiles('.');
            this.fetchTasks();
            this.startLogStream();
            this.connectWebSocket();
        },

        async fetchFiles(path) {
            try {
                const res = await fetch(`/api/files/list?path=${path}`);
                this.files = await res.json();
            } catch (e) {
                console.error('Failed to fetch files', e);
            }
        },

        async openFile(file) {
            if (file.isDir) {
                this.fetchFiles(file.path);
            } else {
                this.activeTab = file.name;
                this.currentView = 'editor';
                try {
                    const res = await fetch(`/api/files/read?path=${file.path}`);
                    const text = await res.text();
                    this.editorContent = text;
                } catch (e) {
                    this.editorContent = 'Error loading file content.';
                }
            }
        },

        async fetchTasks() {
            try {
                const res = await fetch('/api/tasks');
                this.tasks = await res.json();
                this.activeTasksCount = this.tasks.filter(t => t.status === 'in_progress').length;
                this.totalTasksCount = this.tasks.length;
            } catch (e) {
                console.error('Failed to fetch tasks', e);
            }
        },

        startLogStream() {
            // Simulated log stream for UI demo
            setInterval(() => {
                if (this.delqhiLoop) {
                    const time = new Date().toLocaleTimeString();
                    this.logs.push({
                        id: Date.now(),
                        time: time,
                        source: 'ORCHESTRATOR',
                        message: `Processing task queue... Round ${Math.floor(Math.random() * 100)}`
                    });
                    if (this.logs.length > 50) this.logs.shift();
                }
            }, 3000);
        },

        connectWebSocket() {
            this.wsStatus = 'Connected';
            this.wsIndicatorClass = 'text-green-500';
        },

        async sendMessage() {
            if (!this.chatMessage.trim()) return;
            const text = this.chatMessage;
            this.chatHistory.push({ role: 'user', text });
            this.chatMessage = '';
            
            // Auto-scroll chat
            setTimeout(() => {
                const chat = document.getElementById('chat-stream');
                chat.scrollTop = chat.scrollHeight;
            }, 10);

            try {
                const res = await fetch('/api/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ message: text })
                });
                const data = await res.json();
                this.chatHistory.push({ role: 'assistant', text: data.reply });
            } catch (e) {
                this.chatHistory.push({ role: 'assistant', text: 'Error communicating with AI agent.' });
            }
        },

        toggleDelqhiLoop() {
            console.log('Delqhi-Loop:', this.delqhiLoop);
        },

        toggleSwarmEngine() {
            console.log('Swarm Engine:', this.swarmEngine);
        }
    });
});
