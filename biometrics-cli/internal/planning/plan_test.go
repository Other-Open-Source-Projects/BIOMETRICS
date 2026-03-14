package planning

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestBuildPlanDeterministic(t *testing.T) {
	t.Setenv("BIOMETRICS_MAX_WORK_PACKAGES", "")
	goal := "api changes, ui changes, tests, docs"
	planA := BuildPlan(goal)
	planB := BuildPlan(goal)

	if len(planA.WorkPackages) != len(planB.WorkPackages) {
		t.Fatalf("work package count mismatch: %d != %d", len(planA.WorkPackages), len(planB.WorkPackages))
	}
	for i := range planA.WorkPackages {
		if !reflect.DeepEqual(planA.WorkPackages[i], planB.WorkPackages[i]) {
			t.Fatalf("non-deterministic package at index %d: %+v vs %+v", i, planA.WorkPackages[i], planB.WorkPackages[i])
		}
	}
}

func TestBuildPlanNoCapByDefault(t *testing.T) {
	t.Setenv("BIOMETRICS_MAX_WORK_PACKAGES", "")
	parts := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		parts = append(parts, fmt.Sprintf("part-%02d", i))
	}
	goal := strings.Join(parts, ", ")

	plan := BuildPlan(goal)
	if got := len(plan.WorkPackages); got != 60 {
		t.Fatalf("expected 60 work packages, got %d", got)
	}
}

func TestBuildPlanCapsWhenEnvSet(t *testing.T) {
	t.Setenv("BIOMETRICS_MAX_WORK_PACKAGES", "50")
	parts := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		parts = append(parts, fmt.Sprintf("part-%02d", i))
	}
	goal := strings.Join(parts, ", ")

	plan := BuildPlan(goal)
	if got := len(plan.WorkPackages); got != 50 {
		t.Fatalf("expected 50 work packages, got %d", got)
	}
}
