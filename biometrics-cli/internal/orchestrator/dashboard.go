package orchestrator

import (
	"biometrics-cli/internal/state"
	"fmt"
	"time"
)

func DisplayDashboard() {
	for {
		fmt.Print("\033[H\033[2J")
		fmt.Println("==============================================================")
		fmt.Println("         BIOMETRICS ENTERPRISE ORCHESTRATOR DASHBOARD         ")
		fmt.Println("==============================================================")
		fmt.Printf("STATUS:     RUNNING (24/7 MODE)\n")
		fmt.Printf("METRICS:    :59002/metrics\n")

		chaosStatus := "DISABLED"
		if state.GlobalState.ChaosEnabled {
			chaosStatus = "ACTIVE (CHAOS MONKEY)"
		}
		fmt.Printf("CHAOS:      %s\n", chaosStatus)
		fmt.Printf("PLAN:       %s\n", state.GlobalState.PlanName)
		fmt.Printf("AGENT:      %s\n", state.GlobalState.CurrentAgent)
		fmt.Printf("MODEL:      %s\n", state.GlobalState.ActiveModel)
		fmt.Println("--------------------------------------------------------------")
		fmt.Println("RECENT LOGS:")
		for _, l := range state.GlobalState.Logs {
			fmt.Println("  " + l)
		}
		fmt.Println("==============================================================")
		time.Sleep(2 * time.Second)
	}
}
