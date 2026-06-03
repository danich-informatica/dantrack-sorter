package sorter

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers for balance tests
// ---------------------------------------------------------------------------

func balanceSorterConfig(strategy BalanceStrategy) SorterConfig {
	return SorterConfig{
		SorterID: "sorter_balance",
		Exits: []SorterExit{
			{ExitID: "exit_1", Enabled: true},
			{ExitID: "exit_2", Enabled: true},
			{ExitID: "exit_3", Enabled: true},
			{ExitID: "exit_default", Enabled: true},
		},
		DefaultExitID:   "exit_default",
		BalanceStrategy: strategy,
	}
}

func balanceAssignment(id string, targets []string) Assignment {
	return Assignment{
		ID:         id,
		Enabled:    true,
		Priority:   10,
		TargetType: TargetTypeExit,
		TargetIDs:  targets,
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpEquals, Values: []string{"SKU_A"}},
		},
	}
}

func balanceBox() BoxContext {
	return BoxContext{BoxID: "box_1", QR: "qr_1", SKU: "SKU_A"}
}

// ---------------------------------------------------------------------------
// Config / Validation tests
// ---------------------------------------------------------------------------

func TestValidation_SorterConfig_BalanceStrategyEmpty_Valid(t *testing.T) {
	cfg := balanceSorterConfig("")
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidation_SorterConfig_BalanceLeastLoaded_Valid(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidation_SorterConfig_BalanceRoundRobin_InvalidForSorter(t *testing.T) {
	cfg := balanceSorterConfig(BalanceRoundRobin)
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported strategy")
	}
	if !containsStr(err.Error(), "sorter only supports") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidation_SorterConfig_BalanceWeighted_InvalidForSorter(t *testing.T) {
	cfg := balanceSorterConfig(BalanceWeighted)
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported strategy")
	}
}

func TestValidation_SorterConfig_UnknownStrategy_InvalidForSorter(t *testing.T) {
	cfg := balanceSorterConfig("random_strategy")
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

// ---------------------------------------------------------------------------
// Balance: least_loaded picks lowest load
// ---------------------------------------------------------------------------

func TestResolveSorter_Balance_PicksLowestLoad(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 50},
			{ExitID: "exit_2", Available: true, CurrentLoad: 20},
			{ExitID: "exit_3", Available: true, CurrentLoad: 10},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_2" {
		t.Fatalf("expected exit_2 (load=20), got %s", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
	if dec.Trace.RuleApplied != RuleSorterBalancedMultiTarget {
		t.Fatalf("expected RuleSorterBalancedMultiTarget, got %s", dec.Trace.RuleApplied)
	}
}

func TestResolveSorter_Balance_TieBreakByOrder(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 30},
			{ExitID: "exit_2", Available: true, CurrentLoad: 30},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Tie → first in TargetIDs order wins
	if dec.ExitID != "exit_1" {
		t.Fatalf("expected exit_1 (first in TargetIDs), got %s", dec.ExitID)
	}
}

func TestResolveSorter_Balance_NoStateAssumeLoadZero(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 10},
			// exit_2 has no state → assume available (policy) with load=0
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_2" {
		t.Fatalf("expected exit_2 (no state=load 0), got %s", dec.ExitID)
	}
}

func TestResolveSorter_Balance_SingleTargetStillWorks(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	a := Assignment{
		ID:         "a1",
		Enabled:    true,
		Priority:   10,
		TargetType: TargetTypeExit,
		TargetID:   "exit_1", // legacy single target
		Conditions: []MatchCondition{{Field: "sku", Operator: OpEquals, Values: []string{"SKU_A"}}},
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 100},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_1" {
		t.Fatalf("expected exit_1 (single target), got %s", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
}

func TestResolveSorter_NoBalanceStrategy_UsesFirstAvailable(t *testing.T) {
	cfg := balanceSorterConfig("") // no strategy = first-available
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: true, CurrentLoad: 5},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// first-available should pick exit_1 (first in list), NOT exit_2 (lower load)
	if dec.ExitID != "exit_1" {
		t.Fatalf("expected exit_1 (first-available), got %s", dec.ExitID)
	}
}

// ---------------------------------------------------------------------------
// ActionPass: all targets Full
// ---------------------------------------------------------------------------

func TestResolveSorter_Balance_AllTargetsFull_ReturnsActionPass(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: false, Full: true, CurrentLoad: 80},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not return Go error, got %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
	if dec.Trace.RuleApplied != RuleSorterPassAllTargetsFull {
		t.Fatalf("expected RuleSorterPassAllTargetsFull, got %s", dec.Trace.RuleApplied)
	}
	// Must NOT fallback to default exit
	if dec.ExitID == "exit_default" {
		t.Fatal("should not use DefaultExitID when all targets are explicitly full")
	}
	if dec.ExitID != "" {
		t.Fatalf("expected empty ExitID for ActionPass, got %s", dec.ExitID)
	}
}

func TestResolveSorter_Balance_ActionPass_NoGoError(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 50},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not produce Go error: %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
}

func TestResolveSorter_Balance_NoMatch_DoesNotUseActionPass(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// SKU doesn't match any assignment
	box := BoxContext{BoxID: "box_1", QR: "qr_1", SKU: "NO_MATCH"}
	a := balanceAssignment("a1", []string{"exit_1", "exit_2"})

	req := SorterRequest{
		Box:         box,
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: false, Full: true, CurrentLoad: 80},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// No match → should use default/reject, NOT ActionPass
	if dec.Action == ActionPass {
		t.Fatal("ActionPass must not be used when there is no match")
	}
}

// ---------------------------------------------------------------------------
// Mixed states
// ---------------------------------------------------------------------------

func TestResolveSorter_Balance_OneFullOneAvailable_PicksAvailable(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: true, CurrentLoad: 30},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_2" {
		t.Fatalf("expected exit_2 (available), got %s", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
}

func TestResolveSorter_Balance_OneBlockedOneAvailable_PicksAvailable(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Blocked: true, CurrentLoad: 0},
			{ExitID: "exit_2", Available: true, CurrentLoad: 40},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_2" {
		t.Fatalf("expected exit_2 (available), got %s", dec.ExitID)
	}
}

func TestResolveSorter_Balance_AllBlocked_FallsToDefault(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Blocked: true},
			{ExitID: "exit_2", Available: false, Maintenance: true},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_default" {
		t.Fatalf("expected exit_default (fallback), got %s", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
	if !dec.FallbackUsed {
		t.Fatal("expected FallbackUsed=true")
	}
}

func TestResolveSorter_Balance_TargetNotInConfig_PlusAvailable_PicksAvailable(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_nonexistent", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_2", Available: true, CurrentLoad: 15},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ExitID != "exit_2" {
		t.Fatalf("expected exit_2 (available), got %s", dec.ExitID)
	}
}

func TestResolveSorter_Balance_TargetNotInConfig_PlusFull_ActionPass(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// exit_nonexistent not in config, exit_1 is full.
	// The only configured target is full → ActionPass.
	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_nonexistent", "exit_1"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass (configured targets full), got %s", dec.Action)
	}
}

func TestResolveSorter_Balance_OneFullOneBlocked_FallsToDefault(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// exit_1 Full, exit_2 Blocked. Not ALL configured targets are full → fallback.
	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: false, Blocked: true},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Mixed: one full + one blocked. Not purely capacity issue → fallback.
	if dec.ExitID != "exit_default" {
		t.Fatalf("expected exit_default (fallback), got %s", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute (via fallback), got %s", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

func TestResolveSorter_Balance_TraceShowsAllTargets(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2", "exit_3"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 50},
			{ExitID: "exit_2", Available: true, CurrentLoad: 20},
			{ExitID: "exit_3", Available: true, CurrentLoad: 35},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Must have candidate evaluations for all 3 targets
	targetsSeen := map[string]bool{}
	for _, ce := range dec.Trace.CandidateEvaluations {
		if ce.AssignmentID == "a1" && ce.TargetType == TargetTypeExit {
			targetsSeen[ce.TargetID] = true
		}
	}
	for _, tid := range []string{"exit_1", "exit_2", "exit_3"} {
		if !targetsSeen[tid] {
			t.Errorf("trace missing target %s", tid)
		}
	}
}

func TestResolveSorter_Balance_TraceReasonShowsLoad(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: true, CurrentLoad: 50},
			{ExitID: "exit_2", Available: true, CurrentLoad: 10},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(dec.Trace.Reason, "load=") {
		t.Fatalf("trace.Reason should mention load, got %q", dec.Trace.Reason)
	}
}

func TestResolveSorter_Balance_ActionPass_TraceShowsReason(t *testing.T) {
	cfg := balanceSorterConfig(BalanceLeastLoaded)
	eng, err := NewEngine(EngineConfig{Sorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := SorterRequest{
		Box:         balanceBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{balanceAssignment("a1", []string{"exit_1", "exit_2"})}},
		ExitStates: []SorterExitState{
			{ExitID: "exit_1", Available: false, Full: true, CurrentLoad: 100},
			{ExitID: "exit_2", Available: false, Full: true, CurrentLoad: 90},
			{ExitID: "exit_default", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(dec.Trace.Reason, "full") {
		t.Fatalf("trace.Reason should mention full, got %q", dec.Trace.Reason)
	}
	if !containsStr(dec.Trace.DiagnosticMessage, "ActionPass") {
		t.Fatalf("trace.DiagnosticMessage should mention ActionPass, got %q", dec.Trace.DiagnosticMessage)
	}
}

// containsStr checks if s contains substr (utility for tests).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
