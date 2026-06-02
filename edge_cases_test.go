package sorter

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Edge case tests for v0.1.0-rc1 release hardening.
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_NilTagsMapDoesNotPanic(t *testing.T) {
	box := BoxContext{
		BoxID: "BOX-001",
		Tags:  nil, // explícitamente nil
	}
	set := AssignmentSet{
		Assignments: []Assignment{
			{
				ID:       "a1",
				Enabled:  true,
				Priority: 1,
				Conditions: []MatchCondition{
					{Field: "tags.lote", Operator: OpEquals, Values: []string{"L1"}},
				},
				TargetType: TargetTypeExit,
				TargetID:   "exit_A",
			},
		},
	}

	results := EvaluateAssignments(box, set, time.Now())
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Matched {
		t.Error("expected no match when Tags is nil")
	}
}

func TestResolvePresorter_EmptyBalanceStrategy_DefaultsToLeastLoaded(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: "", // vacío → least_loaded
		DefaultParkID:   "park-A",
	}
	engine, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec, err := engine.ResolvePresorter(context.Background(), PresorterRequest{
		Box: BoxContext{BoxID: "BOX-001"},
		ParkStates: []ParkState{
			{ParkID: "park-A", Available: true, CurrentLoad: 50},
			{ParkID: "park-B", Available: true, CurrentLoad: 10},
		},
		EvalTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-B" {
		t.Errorf("expected park-B (least loaded), got %q", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterLeastLoaded {
		t.Errorf("expected rule presorter_least_loaded, got %q", dec.Trace.RuleApplied)
	}
}

func TestResolveSorter_NilExitStates_AssumedAvailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Sorter: &SorterConfig{
			SorterID: "sorter-01",
			Exits:    []SorterExit{{ExitID: "exit_A", Enabled: true}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		Box: BoxContext{BoxID: "BOX-001"},
		Assignments: AssignmentSet{
			Assignments: []Assignment{
				{
					ID: "a1", Enabled: true, Priority: 10,
					Conditions: []MatchCondition{{Field: "box_id", Operator: OpExists}},
					TargetType: TargetTypeExit, TargetID: "exit_A",
				},
			},
		},
		ExitStates: nil, // sin estados → asumido disponible
		EvalTime:   time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ExitID != "exit_A" {
		t.Errorf("expected exit_A, got %q", dec.ExitID)
	}
}

func TestResolvePresorter_NilParkStates_AssumedAvailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Presorter: &PresorterConfig{
			Parks:           []ParkConfig{{ParkID: "park-A", Enabled: true}},
			BalanceStrategy: BalanceLeastLoaded,
			DefaultParkID:   "park-A",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec, err := engine.ResolvePresorter(context.Background(), PresorterRequest{
		Box:        BoxContext{BoxID: "BOX-001"},
		ParkStates: nil, // sin estados → asumido disponible
		EvalTime:   time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ParkID != "park-A" {
		t.Errorf("expected park-A, got %q", dec.ParkID)
	}
}

func TestResolveSorter_AssignmentTargetsExitNotInConfig(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Sorter: &SorterConfig{
			SorterID:      "sorter-01",
			Exits:         []SorterExit{{ExitID: "exit_A", Enabled: true}},
			DefaultExitID: "exit_A",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		Box: BoxContext{BoxID: "BOX-001"},
		Assignments: AssignmentSet{
			Assignments: []Assignment{
				{
					ID: "a1", Enabled: true, Priority: 10,
					Conditions: []MatchCondition{{Field: "box_id", Operator: OpExists}},
					TargetType: TargetTypeExit, TargetID: "exit_UNKNOWN",
				},
			},
		},
		ExitStates: []SorterExitState{{ExitID: "exit_A", Available: true}},
		EvalTime:   time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Assignment matched but exit not in config → should fallback to default.
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute (fallback), got %q", dec.Action)
	}
	if dec.ExitID != "exit_A" {
		t.Errorf("expected fallback to exit_A, got %q", dec.ExitID)
	}
	if !dec.FallbackUsed {
		t.Error("expected FallbackUsed=true")
	}
}

func TestResolveSorter_EmptyAssignments_NoMatch(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Sorter: &SorterConfig{
			SorterID:      "sorter-01",
			Exits:         []SorterExit{{ExitID: "exit_A", Enabled: true}},
			DefaultExitID: "exit_A",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		Box:         BoxContext{BoxID: "BOX-001"},
		Assignments: AssignmentSet{Assignments: nil}, // vacío
		ExitStates:  []SorterExitState{{ExitID: "exit_A", Available: true}},
		EvalTime:    time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No assignments → no match → fallback to default or reject.
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute (default fallback), got %q", dec.Action)
	}
}

func TestEvaluateAssignments_NilAssignmentSlice_ReturnsEmpty(t *testing.T) {
	box := BoxContext{BoxID: "BOX-001"}
	set := AssignmentSet{Assignments: nil}

	results := EvaluateAssignments(box, set, time.Now())
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil assignments, got %d", len(results))
	}
}

func TestResolvePresorter_UnknownParkInStates_Ignored(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Presorter: &PresorterConfig{
			Parks:           []ParkConfig{{ParkID: "park-A", Enabled: true}},
			BalanceStrategy: BalanceLeastLoaded,
			DefaultParkID:   "park-A",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Provide state for a park not in config — should be silently ignored.
	dec, err := engine.ResolvePresorter(context.Background(), PresorterRequest{
		Box: BoxContext{BoxID: "BOX-001"},
		ParkStates: []ParkState{
			{ParkID: "park-A", Available: true, CurrentLoad: 5},
			{ParkID: "park-UNKNOWN", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-A" {
		t.Errorf("expected park-A, got %q", dec.ParkID)
	}
}
