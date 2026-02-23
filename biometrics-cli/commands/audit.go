package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"biometrics-cli/pkg/audit"
)

type AuditQueryFlags struct {
	StartTime  string
	EndTime    string
	EventTypes string
	Actors     string
	Resources  string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
	Format     string
	Output     string
}

type AuditExportFlags struct {
	StartTime string
	EndTime   string
	Format    string
	Output    string
}

func RunAuditQuery(flags *AuditQueryFlags) error {
	config := audit.DefaultAuditConfig()
	auditor, err := audit.NewAuditor(config)
	if err != nil {
		return fmt.Errorf("failed to create auditor: %w", err)
	}
	defer auditor.Stop()

	query := &audit.AuditQuery{
		Limit:     100,
		Offset:    0,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	if flags.StartTime != "" {
		t, err := parseTime(flags.StartTime)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
		query.StartTime = t
	} else {
		query.StartTime = time.Now().Add(-24 * time.Hour)
	}

	if flags.EndTime != "" {
		t, err := parseTime(flags.EndTime)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
		query.EndTime = t
	}

	if flags.EventTypes != "" {
		types := strings.Split(flags.EventTypes, ",")
		query.EventTypes = make([]audit.EventType, 0, len(types))
		for _, t := range types {
			query.EventTypes = append(query.EventTypes, audit.EventType(strings.TrimSpace(t)))
		}
	}

	if flags.Actors != "" {
		query.Actors = strings.Split(flags.Actors, ",")
		for i, a := range query.Actors {
			query.Actors[i] = strings.TrimSpace(a)
		}
	}

	if flags.Resources != "" {
		query.Resources = strings.Split(flags.Resources, ",")
		for i, r := range query.Resources {
			query.Resources[i] = strings.TrimSpace(r)
		}
	}

	if flags.Limit > 0 {
		query.Limit = flags.Limit
	}

	if flags.Offset > 0 {
		query.Offset = flags.Offset
	}

	if flags.SortBy != "" {
		query.SortBy = flags.SortBy
	}

	if flags.SortOrder != "" {
		query.SortOrder = flags.SortOrder
	}

	results, err := auditor.Query(context.Background(), query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	outputFormat := flags.Format
	if outputFormat == "" {
		outputFormat = "table"
	}

	var output []byte
	switch outputFormat {
	case "json":
		output, err = formatJSON(results)
	case "jsonl":
		output, err = formatJSONLines(results)
	case "csv":
		output, err = formatCSV(results)
	default:
		output, err = formatTable(results)
	}

	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	if flags.Output != "" {
		if err := os.WriteFile(flags.Output, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", flags.Output)
	} else {
		fmt.Println(string(output))
	}

	return nil
}

func RunAuditExport(flags *AuditExportFlags) error {
	config := audit.DefaultAuditConfig()
	auditor, err := audit.NewAuditor(config)
	if err != nil {
		return fmt.Errorf("failed to create auditor: %w", err)
	}
	defer auditor.Stop()

	var startTime, endTime time.Time

	if flags.StartTime != "" {
		startTime, err = parseTime(flags.StartTime)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
	} else {
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	}

	if flags.EndTime != "" {
		endTime, err = parseTime(flags.EndTime)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
	} else {
		endTime = time.Now()
	}

	exportFormat := audit.ExportFormatJSON
	switch flags.Format {
	case "csv":
		exportFormat = audit.ExportFormatCSV
	case "xml":
		exportFormat = audit.ExportFormatXML
	}

	data, err := auditor.Export(startTime, endTime, exportFormat)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	if flags.Output != "" {
		if err := os.WriteFile(flags.Output, data, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Exported %d bytes to %s\n", len(data), flags.Output)
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func RunAuditStats() error {
	config := audit.DefaultAuditConfig()
	auditor, err := audit.NewAuditor(config)
	if err != nil {
		return fmt.Errorf("failed to create auditor: %w", err)
	}
	defer auditor.Stop()

	stats, err := auditor.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Println("=== Audit Log Statistics ===")
	fmt.Printf("Total Events:     %d\n", stats.TotalEvents)
	fmt.Printf("Last Event Time:  %s\n", stats.LastEventTime.Format(time.RFC3339))
	fmt.Printf("Storage Size:     %d bytes\n", stats.StorageSize)
	fmt.Printf("Avg Events/Day:   %.2f\n", stats.AvgEventsPerDay)
	fmt.Println()
	fmt.Println("Events by Type:")
	for eventType, count := range stats.EventsByType {
		fmt.Printf("  %-30s %d\n", eventType, count)
	}
	fmt.Println()
	fmt.Println("Top Actors:")
	for actor, count := range stats.EventsByActor {
		fmt.Printf("  %-30s %d\n", actor, count)
	}

	return nil
}

func RunAuditCleanup(retentionDays int) error {
	config := audit.DefaultAuditConfig()
	if retentionDays > 0 {
		config.RetentionDays = retentionDays
	}

	auditor, err := audit.NewAuditor(config)
	if err != nil {
		return fmt.Errorf("failed to create auditor: %w", err)
	}
	defer auditor.Stop()

	if err := auditor.Cleanup(); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Printf("Cleaned up audit logs older than %d days\n", config.RetentionDays)
	return nil
}

func RunAuditRotate() error {
	config := audit.DefaultAuditConfig()
	auditor, err := audit.NewAuditor(config)
	if err != nil {
		return fmt.Errorf("failed to create auditor: %w", err)
	}
	defer auditor.Stop()

	if err := auditor.RotateLogs(); err != nil {
		return fmt.Errorf("rotation failed: %w", err)
	}

	fmt.Println("Audit logs rotated successfully")
	return nil
}

func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func formatJSON(results *audit.AuditQueryResult) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}

func formatJSONLines(results *audit.AuditQueryResult) ([]byte, error) {
	var lines []string
	for _, event := range results.Events {
		data, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		lines = append(lines, string(data))
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func formatCSV(results *audit.AuditQueryResult) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("id,timestamp,event_type,actor,action,resource\n")

	for _, event := range results.Events {
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s\n",
			event.ID,
			event.Timestamp.Format(time.RFC3339),
			event.EventType,
			event.Actor,
			event.Action,
			event.Resource,
		))
	}

	return []byte(sb.String()), nil
}

func formatTable(results *audit.AuditQueryResult) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Total: %d events (showing %d)\n\n", results.TotalCount, len(results.Events)))
	sb.WriteString(fmt.Sprintf("%-20s %-25s %-20s %-15s %-15s\n",
		"TIMESTAMP", "EVENT TYPE", "ACTOR", "ACTION", "RESOURCE"))
	sb.WriteString(strings.Repeat("-", 100) + "\n")

	for _, event := range results.Events {
		sb.WriteString(fmt.Sprintf("%-20s %-25s %-20s %-15s %-15s\n",
			event.Timestamp.Format("2006-01-02 15:04:05"),
			truncate(string(event.EventType), 25),
			truncate(event.Actor, 20),
			truncate(event.Action, 15),
			truncate(event.Resource, 15),
		))
	}

	if results.HasMore {
		sb.WriteString("\n(More results available - use --offset to paginate)\n")
	}

	return []byte(sb.String()), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func PrintAuditHelp() {
	fmt.Println(`
Audit Commands - Query and manage audit logs

Usage:
  biometrics audit <subcommand> [options]

Subcommands:
  query     Query audit events with filters
  export    Export audit events to file
  stats     Show audit log statistics
  cleanup   Remove old audit logs
  rotate    Rotate audit log files

Query Options:
  --start-time    Start time filter (RFC3339 or YYYY-MM-DD)
  --end-time      End time filter (RFC3339 or YYYY-MM-DD)
  --event-types   Comma-separated event types
  --actors        Comma-separated actor IDs
  --resources     Comma-separated resource names
  --limit         Maximum results (default: 100)
  --offset        Pagination offset
  --sort-by       Sort field (timestamp, event_type, actor)
  --sort-order    Sort order (asc, desc)
  --format        Output format (table, json, jsonl, csv)
  --output        Output file path

Export Options:
  --start-time    Start time (default: 7 days ago)
  --end-time      End time (default: now)
  --format        Export format (json, csv, xml)
  --output        Output file path

Examples:
  # Query last 24 hours
  biometrics audit query

  # Query specific time range
  biometrics audit query --start-time 2026-02-01 --end-time 2026-02-20

  # Query by event type
  biometrics audit query --event-types authentication.failure,authorization.denied

  # Query by actor
  biometrics audit query --actors user-123,user-456

  # Export to JSON
  biometrics audit export --format json --output audit.json

  # Show statistics
  biometrics audit stats

  # Cleanup old logs
  biometrics audit cleanup --retention-days 90`)
}
