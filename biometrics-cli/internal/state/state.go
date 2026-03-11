package state

import (
	"biometrics-cli/internal/paths"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AppState struct {
	mu           sync.Mutex
	ActivePlan   string
	PlanName     string
	CurrentAgent string
	ActiveModel  string
	ModelStatus  map[string]string
	Logs         []string
	DB           *sql.DB
	ChaosEnabled bool
	ChaosActive  bool
}

var GlobalState = &AppState{
	ModelStatus:  make(map[string]string),
	Logs:         make([]string, 0),
	ChaosEnabled: true,
}

func (s *AppState) InitDB() {
	dbPath := paths.SisyphusDBPath("logs.db")
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return
	}
	query := "CREATE TABLE IF NOT EXISTS logs (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp TEXT, level TEXT, agent TEXT, plan TEXT, message TEXT)"
	_, _ = db.Exec(query)
	s.DB = db
}

func (s *AppState) Log(level, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ts := time.Now().Format("15:04:05")
	s.Logs = append(s.Logs, fmt.Sprintf("[%s] %s: %s", ts, level, msg))
	if len(s.Logs) > 10 {
		s.Logs = s.Logs[1:]
	}
	if s.DB != nil {
		_, _ = s.DB.Exec("INSERT INTO logs (timestamp, level, agent, plan, message) VALUES (?, ?, ?, ?, ?)",
			time.Now().Format(time.RFC3339), level, s.CurrentAgent, s.PlanName, msg)
	}
}

func (s *AppState) SetChaos(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ChaosActive = active
}

func (s *AppState) GetChaos() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ChaosActive
}
