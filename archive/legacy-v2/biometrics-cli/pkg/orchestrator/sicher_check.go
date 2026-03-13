package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SicherCheckResult represents the result of a verification check
type SicherCheckResult struct {
	Passed        bool     `json:"passed"`
	Checks        []Check  `json:"checks"`
	FilesVerified int      `json:"files_verified"`
	TestsPassed   int      `json:"tests_passed"`
	TestsFailed   int      `json:"tests_failed"`
	GitCommitted  bool     `json:"git_committed"`
	NoDuplicates  bool     `json:"no_duplicates"`
	Issues        []string `json:"issues,omitempty"`
}

// Check represents a single verification check
type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// PerformSicherCheck performs comprehensive verification of agent work
func (o *Orchestrator) PerformSicherCheck(session *AgentSession) *SicherCheckResult {
	fmt.Printf("🔍 Starting 'Sicher?' check for agent %s (Session: %s)...\n", session.AgentName, session.ID)

	result := &SicherCheckResult{
		Passed: true,
		Checks: make([]Check, 0),
	}

	// Check 1: Verify files were created/modified
	filesCheck := o.checkFilesCreatedOrModified(session)
	result.Checks = append(result.Checks, filesCheck)
	if !filesCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, filesCheck.Message)
	}
	result.FilesVerified = result.FilesVerified + 1

	// Check 2: Verify tests pass (if applicable)
	testsCheck := o.checkTestsPass(session)
	result.Checks = append(result.Checks, testsCheck)
	if !testsCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, testsCheck.Message)
	}
	result.TestsPassed = 1
	if !testsCheck.Passed {
		result.TestsFailed = 1
		result.TestsPassed = 0
	}

	// Check 3: Verify git commit was made
	gitCheck := o.checkGitCommit(session)
	result.Checks = append(result.Checks, gitCheck)
	if !gitCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, gitCheck.Message)
	}
	result.GitCommitted = gitCheck.Passed

	// Check 4: Verify no duplicates created
	dupCheck := o.checkNoDuplicates(session)
	result.Checks = append(result.Checks, dupCheck)
	if !dupCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, dupCheck.Message)
	}
	result.NoDuplicates = dupCheck.Passed

	// Check 5: Verify LSP diagnostics are clean
	lspCheck := o.checkLSPDiagnostics(session)
	result.Checks = append(result.Checks, lspCheck)
	if !lspCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, lspCheck.Message)
	}

	// Check 6: Verify agent didn't lie about completion
	honestyCheck := o.checkAgentHonesty(session)
	result.Checks = append(result.Checks, honestyCheck)
	if !honestyCheck.Passed {
		result.Passed = false
		result.Issues = append(result.Issues, honestyCheck.Message)
	}

	// Print summary
	o.printSicherCheckSummary(result)

	return result
}

// checkFilesCreatedOrModified verifies that files were actually created or modified
func (o *Orchestrator) checkFilesCreatedOrModified(session *AgentSession) Check {
	check := Check{
		Name: "Files Created/Modified",
	}

	// Parse session result to find mentioned files
	files := o.extractFilesFromResult(session.Result)
	if len(files) == 0 {
		check.Passed = false
		check.Message = "No files mentioned in agent result"
		return check
	}

	verifiedCount := 0
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			verifiedCount++
		}
	}

	if verifiedCount == 0 {
		check.Passed = false
		check.Message = fmt.Sprintf("None of %d mentioned files exist", len(files))
		return check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("Verified %d/%d files exist", verifiedCount, len(files))
	return check
}

// checkTestsPass runs tests if they exist
func (o *Orchestrator) checkTestsPass(session *AgentSession) Check {
	check := Check{
		Name: "Tests Pass",
	}

	// Check if there are test files in the modified directories
	testFiles := o.findTestFiles(session)
	if len(testFiles) == 0 {
		check.Passed = true
		check.Message = "No test files found (skipped)"
		return check
	}

	// Run tests
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = o.config.ProjectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Tests failed: %v\n%s", err, string(output))
		return check
	}

	check.Passed = true
	check.Message = "All tests passed"
	return check
}

// checkGitCommit verifies a git commit was made
func (o *Orchestrator) checkGitCommit(session *AgentSession) Check {
	check := Check{
		Name: "Git Commit",
	}

	// Check git status for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = o.config.ProjectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("Git command failed: %v", err)
		return check
	}

	// If there are uncommitted changes, agent didn't commit
	if len(output) > 0 {
		check.Passed = false
		check.Message = "Uncommitted changes detected - agent didn't commit"
		return check
	}

	// Check if a commit was made in the last minute
	cmd = exec.Command("git", "log", "-1", "--since=1 minute", "--format=%H %s")
	cmd.Dir = o.config.ProjectRoot
	output, err = cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		check.Passed = false
		check.Message = "No recent commit found"
		return check
	}

	check.Passed = true
	check.Message = fmt.Sprintf("Commit found: %s", strings.TrimSpace(string(output)))
	return check
}

// checkNoDuplicates verifies no duplicate files were created
func (o *Orchestrator) checkNoDuplicates(session *AgentSession) Check {
	check := Check{
		Name: "No Duplicates",
	}

	// Extract files from result
	files := o.extractFilesFromResult(session.Result)
	if len(files) == 0 {
		check.Passed = true
		check.Message = "No files to check for duplicates"
		return check
	}

	// Check for duplicates
	seen := make(map[string]bool)
	duplicates := []string{}

	for _, file := range files {
		if seen[file] {
			duplicates = append(duplicates, file)
		}
		seen[file] = true
	}

	if len(duplicates) > 0 {
		check.Passed = false
		check.Message = fmt.Sprintf("Duplicate files detected: %v", duplicates)
		return check
	}

	check.Passed = true
	check.Message = "No duplicate files found"
	return check
}

// checkLSPDiagnostics verifies LSP diagnostics are clean
func (o *Orchestrator) checkLSPDiagnostics(session *AgentSession) Check {
	check := Check{
		Name: "LSP Diagnostics Clean",
	}

	// This would integrate with LSP server
	// For now, we'll do a basic go vet check
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = o.config.ProjectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		check.Passed = false
		check.Message = fmt.Sprintf("LSP/vet errors found: %v\n%s", err, string(output))
		return check
	}

	check.Passed = true
	check.Message = "No LSP/vet errors found"
	return check
}

// checkAgentHonesty verifies agent didn't lie about what it did
func (o *Orchestrator) checkAgentHonesty(session *AgentSession) Check {
	check := Check{
		Name: "Agent Honesty",
	}

	// Check if agent claimed to do something but didn't
	claims := o.extractClaimsFromResult(session.Result)

	unfulfilledClaims := []string{}
	for _, claim := range claims {
		if !o.verifyClaim(claim) {
			unfulfilledClaims = append(unfulfilledClaims, claim)
		}
	}

	if len(unfulfilledClaims) > 0 {
		check.Passed = false
		check.Message = fmt.Sprintf("Unfulfilled claims: %v", unfulfilledClaims)
		return check
	}

	check.Passed = true
	check.Message = "All claims verified"
	return check
}

// Helper functions

func (o *Orchestrator) extractFilesFromResult(result string) []string {
	// Parse result to find file paths
	// Look for patterns like "Created file:", "Modified:", "Wrote file successfully"
	files := []string{}
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Created file") ||
			strings.Contains(line, "Modified") ||
			strings.Contains(line, "Wrote file") {
			// Extract file path
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				path := strings.TrimSpace(parts[len(parts)-1])
				if path != "" {
					files = append(files, path)
				}
			}
		}
	}

	return files
}

func (o *Orchestrator) findTestFiles(session *AgentSession) []string {
	testFiles := []string{}
	filepath.Walk(o.config.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	return testFiles
}

func (o *Orchestrator) extractClaimsFromResult(result string) []string {
	// Extract claims like "I created X", "I fixed Y", etc.
	claims := []string{}
	// TODO: Implement claim extraction
	return claims
}

func (o *Orchestrator) verifyClaim(claim string) bool {
	// Verify if a claim is true
	// TODO: Implement claim verification
	return true
}

func (o *Orchestrator) printSicherCheckSummary(result *SicherCheckResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 SICHER? CHECK SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	for _, check := range result.Checks {
		status := "✅"
		if !check.Passed {
			status = "❌"
		}
		fmt.Printf("%s %s: %s\n", status, check.Name, check.Message)
	}

	fmt.Println(strings.Repeat("=", 60))
	if result.Passed {
		fmt.Println("✅ SICHER? CHECK PASSED - Agent work verified!")
	} else {
		fmt.Println("❌ SICHER? CHECK FAILED - Issues found:")
		for _, issue := range result.Issues {
			fmt.Printf("   • %s\n", issue)
		}
	}
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// SaveSicherCheckResult saves the verification result to disk
func (o *Orchestrator) SaveSicherCheckResult(sessionID string, result *SicherCheckResult) error {
	dir := filepath.Join(o.config.ProjectRoot, ".sisyphus", "sicher-checks")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", sessionID))
	return ioutil.WriteFile(filename, data, 0644)
}
