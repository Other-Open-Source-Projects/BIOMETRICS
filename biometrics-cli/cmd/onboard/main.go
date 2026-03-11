package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"biometrics-cli/internal/onboarding"
)

func main() {
	var (
		resume         bool
		yes            bool
		nonInteractive bool
		doctor         bool
		workspace      string
	)

	flag.BoolVar(&resume, "resume", false, "resume from existing onboarding state")
	flag.BoolVar(&yes, "yes", false, "auto-confirm privileged installer prompts")
	flag.BoolVar(&nonInteractive, "non-interactive", false, "fail instead of prompting for confirmation")
	flag.BoolVar(&doctor, "doctor", false, "run non-mutating health checks only")
	flag.StringVar(&workspace, "workspace", "", "explicit BIOMETRICS workspace path")
	flag.Parse()

	runner, err := onboarding.NewRunner(onboarding.Options{
		Workspace:      workspace,
		Resume:         resume,
		Yes:            yes,
		NonInteractive: nonInteractive,
		Doctor:         doctor,
		Out:            os.Stdout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[onboard] init failed: %v\n", err)
		os.Exit(1)
	}

	if err := runner.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "[onboard] failed: %v\n", err)
		os.Exit(1)
	}
}
