package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"biometrics-cli/internal/skillkit"
	"biometrics-cli/internal/skillops"
)

func main() {
	workspace := detectWorkspace()
	manager, err := skillkit.NewManager(skillkit.ManagerOptions{
		Workspace: workspace,
		CWD:       workspace,
		CodexHome: strings.TrimSpace(os.Getenv("CODEX_HOME")),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills manager init failed: %v\n", err)
		os.Exit(1)
	}
	manager.SetOperations(skillops.New(workspace, strings.TrimSpace(os.Getenv("CODEX_HOME"))))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()
	sub := os.Args[1]
	switch sub {
	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		experimental := fs.Bool("experimental", false, "list experimental curated skills")
		_ = fs.Parse(os.Args[2:])
		if result, err := manager.ListInstallable(ctx, *experimental); err == nil {
			printJSON(result)
			return
		}
		skills, err := manager.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "list skills failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(skills)
	case "reload":
		outcome, err := manager.Reload()
		if err != nil {
			printJSON(outcome)
			fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(outcome)
	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		name := fs.String("name", "", "curated skill name")
		repo := fs.String("repo", "", "owner/repo")
		url := fs.String("url", "", "github tree URL")
		paths := fs.String("paths", "", "comma-separated skill paths inside repo")
		ref := fs.String("ref", "", "git ref")
		dest := fs.String("dest", "", "destination skills directory")
		method := fs.String("method", "", "install method: auto|download|git")
		experimental := fs.Bool("experimental", false, "use experimental curated path when --name is set")
		_ = fs.Parse(os.Args[2:])

		req := skillkit.InstallRequest{
			Name:         strings.TrimSpace(*name),
			Repo:         strings.TrimSpace(*repo),
			URL:          strings.TrimSpace(*url),
			Ref:          strings.TrimSpace(*ref),
			Dest:         strings.TrimSpace(*dest),
			Method:       strings.TrimSpace(*method),
			Experimental: *experimental,
		}
		if raw := strings.TrimSpace(*paths); raw != "" {
			req.Paths = splitCSV(raw)
		}
		result, err := manager.Install(ctx, req)
		printJSON(result)
		if err != nil {
			os.Exit(1)
		}
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		name := fs.String("name", "", "skill name")
		path := fs.String("path", "", "output directory")
		resources := fs.String("resources", "", "comma-separated resources: scripts,references,assets")
		examples := fs.Bool("examples", false, "create example files")
		validate := fs.Bool("validate", true, "run quick validation after create")
		_ = fs.Parse(os.Args[2:])

		req := skillkit.CreateRequest{
			Name:      strings.TrimSpace(*name),
			Path:      strings.TrimSpace(*path),
			Resources: splitCSV(*resources),
			Examples:  *examples,
			Validate:  *validate,
		}
		result, err := manager.Create(ctx, req)
		printJSON(result)
		if err != nil {
			os.Exit(1)
		}
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		skillPath := fs.String("path", "", "path to skill folder")
		_ = fs.Parse(os.Args[2:])
		if strings.TrimSpace(*skillPath) == "" {
			fmt.Fprintln(os.Stderr, "--path is required")
			os.Exit(1)
		}
		ops := skillops.New(workspace, strings.TrimSpace(os.Getenv("CODEX_HOME")))
		result, err := ops.Validate(ctx, strings.TrimSpace(*skillPath))
		printJSON(result)
		if err != nil {
			os.Exit(1)
		}
	case "enable":
		reference := parseReferenceArgs(os.Args[2:])
		if reference == "" {
			fmt.Fprintln(os.Stderr, "enable requires --name or --path")
			os.Exit(1)
		}
		result, err := manager.Enable(reference)
		printJSON(result)
		if err != nil {
			os.Exit(1)
		}
	case "disable":
		reference := parseReferenceArgs(os.Args[2:])
		if reference == "" {
			fmt.Fprintln(os.Stderr, "disable requires --name or --path")
			os.Exit(1)
		}
		result, err := manager.Disable(reference)
		printJSON(result)
		if err != nil {
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func parseReferenceArgs(args []string) string {
	fs := flag.NewFlagSet("reference", flag.ExitOnError)
	name := fs.String("name", "", "skill name")
	path := fs.String("path", "", "absolute path to SKILL.md")
	_ = fs.Parse(args)
	if strings.TrimSpace(*path) != "" {
		return strings.TrimSpace(*path)
	}
	return strings.TrimSpace(*name)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func detectWorkspace() string {
	if env := strings.TrimSpace(os.Getenv("BIOMETRICS_WORKSPACE")); env != "" {
		return env
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(cwd) == "biometrics-cli" {
		return filepath.Dir(cwd)
	}
	return cwd
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func printUsage() {
	fmt.Println("usage: biometrics-skills <command> [flags]")
	fmt.Println("commands: list, reload, install, create, validate, enable, disable")
}
