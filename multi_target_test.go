package sorter

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// assignmentTargetIDs helper tests
// ---------------------------------------------------------------------------

func TestAssignmentTargetIDs_TargetIDOnly(t *testing.T) {
	a := Assignment{TargetID: "exit_1"}
	got := assignmentTargetIDs(a)
	if len(got) != 1 || got[0] != "exit_1" {
		t.Fatalf("expected [exit_1], got %v", got)
	}
}

func TestAssignmentTargetIDs_TargetIDsOnly(t *testing.T) {
	a := Assignment{TargetIDs: []string{"exit_1", "exit_3"}}
	got := assignmentTargetIDs(a)
	if len(got) != 2 || got[0] != "exit_1" || got[1] != "exit_3" {
		t.Fatalf("expected [exit_1 exit_3], got %v", got)
	}
}

func TestAssignmentTargetIDs_TargetIDsOverridesTargetID(t *testing.T) {
	a := Assignment{TargetID: "exit_old", TargetIDs: []string{"exit_1", "exit_3"}}
	got := assignmentTargetIDs(a)
	if len(got) != 2 || got[0] != "exit_1" || got[1] != "exit_3" {
		t.Fatalf("TargetIDs should override TargetID, got %v", got)
	}
}

func TestAssignmentTargetIDs_BothEmpty(t *testing.T) {
	a := Assignment{}
	got := assignmentTargetIDs(a)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestAssignmentTargetIDs_TargetIDsWithEmptyStrings(t *testing.T) {
	a := Assignment{TargetIDs: []string{"", "exit_1", "", "exit_3", ""}}
	got := assignmentTargetIDs(a)
	if len(got) != 2 || got[0] != "exit_1" || got[1] != "exit_3" {
		t.Fatalf("expected [exit_1 exit_3], got %v", got)
	}
}

func TestAssignmentTargetIDs_TargetIDsAllEmpty_FallsBackToTargetID(t *testing.T) {
	a := Assignment{TargetID: "exit_legacy", TargetIDs: []string{"", ""}}
	got := assignmentTargetIDs(a)
	if len(got) != 1 || got[0] != "exit_legacy" {
		t.Fatalf("expected fallback to TargetID, got %v", got)
	}
}

func TestAssignmentTargetIDs_Deduplicates(t *testing.T) {
	a := Assignment{TargetIDs: []string{"exit_1", "exit_1", "exit_3"}}
	got := assignmentTargetIDs(a)
	if len(got) != 2 || got[0] != "exit_1" || got[1] != "exit_3" {
		t.Fatalf("expected deduplication [exit_1 exit_3], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Validation tests for multi-target
// ---------------------------------------------------------------------------

func TestValidation_AssignmentWithOnlyTargetID(t *testing.T) {
	a := Assignment{ID: "a1", Enabled: true, Priority: 1, TargetType: TargetTypeExit, TargetID: "exit_1"}
	if err := validateAssignment(a, 0); err != nil {
		t.Fatalf("should be valid: %v", err)
	}
}

func TestValidation_AssignmentWithOnlyTargetIDs(t *testing.T) {
	a := Assignment{ID: "a1", Enabled: true, Priority: 1, TargetType: TargetTypeExit, TargetIDs: []string{"exit_1", "exit_3"}}
	if err := validateAssignment(a, 0); err != nil {
		t.Fatalf("should be valid: %v", err)
	}
}

func TestValidation_AssignmentWithMultipleTargetIDs(t *testing.T) {
	a := Assignment{ID: "a1", Enabled: true, Priority: 1, TargetType: TargetTypeExit, TargetIDs: []string{"exit_1", "exit_2", "exit_3"}}
	if err := validateAssignment(a, 0); err != nil {
		t.Fatalf("should be valid: %v", err)
	}
}

func TestValidation_AssignmentNoTargetIDNorTargetIDs_Fails(t *testing.T) {
	a := Assignment{ID: "a1", Enabled: true, Priority: 1, TargetType: TargetTypeExit}
	err := validateAssignment(a, 0)
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestValidation_AssignmentTargetIDsAllEmpty_Fails(t *testing.T) {
	a := Assignment{ID: "a1", Enabled: true, Priority: 1, TargetType: TargetTypeExit, TargetIDs: []string{"", ""}}
	err := validateAssignment(a, 0)
	if err == nil {
		t.Fatal("expected error for all-empty TargetIDs")
	}
}

// ---------------------------------------------------------------------------
// EvaluateAssignments with TargetIDs
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_MultiTargetMatch(t *testing.T) {
	box := BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"}
	set := AssignmentSet{
		Assignments: []Assignment{
			{
				ID: "multi1", Enabled: true, Priority: 10,
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}},
				TargetType: TargetTypeExit,
				TargetIDs:  []string{"exit_1", "exit_3"},
			},
		},
	}

	results := EvaluateAssignments(box, set, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Matched {
		t.Fatal("expected match")
	}
	// The assignment preserves TargetIDs
	if len(results[0].Assignment.TargetIDs) != 2 {
		t.Fatalf("expected TargetIDs preserved, got %v", results[0].Assignment.TargetIDs)
	}
}

func TestEvaluateAssignments_LegacyTargetIDStillWorks(t *testing.T) {
	box := BoxContext{BoxID: "B1", QR: "Q1", Variety: "Gala"}
	set := AssignmentSet{
		Assignments: []Assignment{
			{
				ID: "legacy1", Enabled: true, Priority: 5,
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Gala"}}},
				TargetType: TargetTypeExit,
				TargetID:   "exit_2",
			},
		},
	}

	results := EvaluateAssignments(box, set, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if !results[0].Matched {
		t.Fatal("expected match with legacy TargetID")
	}
	if results[0].Assignment.TargetID != "exit_2" {
		t.Fatalf("expected TargetID preserved, got %q", results[0].Assignment.TargetID)
	}
}

// ---------------------------------------------------------------------------
// ResolveSorter with multi-target
// ---------------------------------------------------------------------------

func newMultiTargetEngine(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine(EngineConfig{
		Sorter: &SorterConfig{
			SorterID: "sorter-mt",
			Exits: []SorterExit{
				{ExitID: "exit_1", Enabled: true},
				{ExitID: "exit_2", Enabled: true},
				{ExitID: "exit_3", Enabled: true},
				{ExitID: "exit_default", Enabled: true},
				{ExitID: "exit_reject", Enabled: true},
			},
			DefaultExitID:   "exit_default",
			RejectExitID:    "exit_reject",
			AmbiguityPolicy: AmbiguityPolicyFirstWins,
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return engine
}

func TestResolveSorter_LegacyTargetID(t *testing.T) {
	engine := newMultiTargetEngine(t)
	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "t1",
		Box:     BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"},
		Assignments: AssignmentSet{Assignments: []Assignment{
			{ID: "a1", Enabled: true, Priority: 10, TargetType: TargetTypeExit, TargetID: "exit_1",
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}}},
		}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true},
			{ExitID: "exit_2", Available: true},
			{ExitID: "exit_3", Available: true},
			{ExitID: "exit_default", Available: true},
			{ExitID: "exit_reject", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute || dec.ExitID != "exit_1" {
		t.Fatalf("expected route to exit_1, got %s/%s", dec.Action, dec.ExitID)
	}
}

func TestResolveSorter_MultiTarget_FirstAvailable(t *testing.T) {
	engine := newMultiTargetEngine(t)
	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "t2",
		Box:     BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"},
		Assignments: AssignmentSet{Assignments: []Assignment{
			{ID: "multi1", Enabled: true, Priority: 10, TargetType: TargetTypeExit,
				TargetIDs:  []string{"exit_1", "exit_3"},
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}}},
		}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true},
			{ExitID: "exit_2", Available: true},
			{ExitID: "exit_3", Available: true},
			{ExitID: "exit_default", Available: true},
			{ExitID: "exit_reject", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute || dec.ExitID != "exit_1" {
		t.Fatalf("expected route to first available (exit_1), got %s/%s", dec.Action, dec.ExitID)
	}
	if dec.AssignmentID != "multi1" {
		t.Fatalf("expected assignmentID multi1, got %q", dec.AssignmentID)
	}
}

func TestResolveSorter_MultiTarget_SkipsBlockedUsesNext(t *testing.T) {
	engine := newMultiTargetEngine(t)
	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "t3",
		Box:     BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"},
		Assignments: AssignmentSet{Assignments: []Assignment{
			{ID: "multi2", Enabled: true, Priority: 10, TargetType: TargetTypeExit,
				TargetIDs:  []string{"exit_1", "exit_3"},
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}}},
		}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Blocked: true}, // blocked
			{ExitID: "exit_2", Available: true},
			{ExitID: "exit_3", Available: true}, // next available
			{ExitID: "exit_default", Available: true},
			{ExitID: "exit_reject", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute || dec.ExitID != "exit_3" {
		t.Fatalf("expected route to exit_3 (skip blocked exit_1), got %s/%s", dec.Action, dec.ExitID)
	}
}

func TestResolveSorter_MultiTarget_AllBlockedFallsBackToDefault(t *testing.T) {
	engine := newMultiTargetEngine(t)
	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "t4",
		Box:     BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"},
		Assignments: AssignmentSet{Assignments: []Assignment{
			{ID: "multi3", Enabled: true, Priority: 10, TargetType: TargetTypeExit,
				TargetIDs:  []string{"exit_1", "exit_3"},
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}}},
		}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Blocked: true},
			{ExitID: "exit_2", Available: true},
			{ExitID: "exit_3", Available: false, Full: true},
			{ExitID: "exit_default", Available: true},
			{ExitID: "exit_reject", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All assigned targets blocked → falls back to default exit
	if dec.Action != ActionRoute || dec.ExitID != "exit_default" {
		t.Fatalf("expected fallback to exit_default, got %s/%s", dec.Action, dec.ExitID)
	}
	if !dec.FallbackUsed {
		t.Fatal("expected FallbackUsed=true")
	}
}

func TestResolveSorter_MultiTarget_CandidateEvaluationsShowsAllTargets(t *testing.T) {
	engine := newMultiTargetEngine(t)
	dec, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "t5",
		Box:     BoxContext{BoxID: "B1", QR: "Q1", Variety: "Fuji"},
		Assignments: AssignmentSet{Assignments: []Assignment{
			{ID: "multi4", Enabled: true, Priority: 10, TargetType: TargetTypeExit,
				TargetIDs:  []string{"exit_1", "exit_3"},
				Conditions: []MatchCondition{{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}}}},
		}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Blocked: true},
			{ExitID: "exit_2", Available: true},
			{ExitID: "exit_3", Available: true},
			{ExitID: "exit_default", Available: true},
			{ExitID: "exit_reject", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have candidate evaluations for exit_1 (rejected) and exit_3 (chosen)
	foundExit1 := false
	foundExit3 := false
	for _, ce := range dec.Trace.CandidateEvaluations {
		if ce.TargetID == "exit_1" && !ce.Eligible {
			foundExit1 = true
		}
		if ce.TargetID == "exit_3" && ce.Eligible {
			foundExit3 = true
		}
	}
	if !foundExit1 {
		t.Fatal("expected candidate evaluation for exit_1 (rejected)")
	}
	if !foundExit3 {
		t.Fatal("expected candidate evaluation for exit_3 (eligible)")
	}
}

// ---------------------------------------------------------------------------
// ActionPass constant test
// ---------------------------------------------------------------------------

func TestActionPass_Exists(t *testing.T) {
	if ActionPass != "pass" {
		t.Fatalf("expected ActionPass = \"pass\", got %q", ActionPass)
	}
}

func TestActionPass_DifferentFromReject(t *testing.T) {
	if ActionPass == ActionReject {
		t.Fatal("ActionPass must be different from ActionReject")
	}
}
