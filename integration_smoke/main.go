// Command integration_smoke simulates an external orchestrator consuming
// dantrack-sorter to validate the library can be integrated correctly.
//
// This is NOT a production orchestrator. It uses hardcoded data to simulate
// the flow: load config → receive events → resolve decisions → validate results.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func main() {
	fmt.Println("=== DANTRACK-SORTER INTEGRATION SMOKE TEST ===")
	fmt.Println()

	// --- 1. Load config (simulates dantrack-db) ---
	sorterCfg := &sorter.SorterConfig{
		SorterID: "sorter-plant-01",
		Exits: []sorter.SorterExit{
			{ExitID: "exit_fuji", Enabled: true, Priority: 1, Capacity: 50},
			{ExitID: "exit_gala", Enabled: true, Priority: 2, Capacity: 50},
			{ExitID: "exit_default", Enabled: true, Priority: 0, Capacity: 100},
			{ExitID: "exit_reject", Enabled: true, Priority: 0, Capacity: 20},
		},
		DefaultExitID:   "exit_default",
		RejectExitID:    "exit_reject",
		AmbiguityPolicy: sorter.AmbiguityPolicyFirstWins,
	}

	presorterCfg := &sorter.PresorterConfig{
		Parks: []sorter.ParkConfig{
			{ParkID: "park_A", Enabled: true, HasErrorControl: false, MaxLoad: 100, Weight: 5},
			{ParkID: "park_B", Enabled: true, HasErrorControl: false, MaxLoad: 100, Weight: 3},
			{ParkID: "park_error", Enabled: true, HasErrorControl: true, MaxLoad: 50, Weight: 1},
		},
		BalanceStrategy: sorter.BalanceLeastLoaded,
		DefaultParkID:   "park_A",
	}

	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Sorter:    sorterCfg,
		Presorter: presorterCfg,
	})
	if err != nil {
		log.Fatalf("FAIL: NewEngine: %v", err)
	}
	fmt.Println("[OK] Engine created with Sorter + Presorter config")

	// --- 2. Simulate assignments (from DB) ---
	assignments := sorter.AssignmentSet{
		Assignments: []sorter.Assignment{
			{
				ID: "assign_fuji_80", Enabled: true, Priority: 10,
				Conditions: []sorter.MatchCondition{
					{Field: "variety", Operator: sorter.OpEquals, Values: []string{"Fuji"}},
					{Field: "caliber", Operator: sorter.OpIn, Values: []string{"80", "85"}},
				},
				TargetType: sorter.TargetTypeExit, TargetID: "exit_fuji",
			},
			{
				ID: "assign_gala", Enabled: true, Priority: 5,
				Conditions: []sorter.MatchCondition{
					{Field: "variety", Operator: sorter.OpEquals, Values: []string{"Gala"}},
				},
				TargetType: sorter.TargetTypeExit, TargetID: "exit_gala",
			},
			{
				ID: "assign_premium_tag", Enabled: true, Priority: 15,
				Conditions: []sorter.MatchCondition{
					{Field: "tags.grade", Operator: sorter.OpEquals, Values: []string{"premium"}},
				},
				TargetType: sorter.TargetTypeExit, TargetID: "exit_fuji",
			},
		},
	}
	fmt.Printf("[OK] %d assignments loaded\n", len(assignments.Assignments))

	// --- 3. Simulate exit states (from PLC/hardware) ---
	exitStatesAllOK := []sorter.SorterExitState{
		{ExitID: "exit_fuji", Available: true, CurrentLoad: 20},
		{ExitID: "exit_gala", Available: true, CurrentLoad: 15},
		{ExitID: "exit_default", Available: true, CurrentLoad: 5},
		{ExitID: "exit_reject", Available: true, CurrentLoad: 0},
	}

	exitStatesBlockedFuji := []sorter.SorterExitState{
		{ExitID: "exit_fuji", Available: false, Blocked: true},
		{ExitID: "exit_gala", Available: true, CurrentLoad: 15},
		{ExitID: "exit_default", Available: true, CurrentLoad: 5},
		{ExitID: "exit_reject", Available: true, CurrentLoad: 0},
	}

	// --- 4. Simulate park states (from PLC/hardware) ---
	parkStates := []sorter.ParkState{
		{ParkID: "park_A", Available: true, CurrentLoad: 45},
		{ParkID: "park_B", Available: true, CurrentLoad: 12},
		{ParkID: "park_error", Available: true, CurrentLoad: 3},
	}

	evalTime := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	ctx := context.Background()

	fmt.Println()
	fmt.Println("--- TEST 1: Sorter — SKU match routes to expected exit ---")
	testSorterMatch(ctx, engine, assignments, exitStatesAllOK, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 2: Sorter — No match goes to reject exit ---")
	testSorterNoMatch(ctx, engine, exitStatesAllOK, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 3: Sorter — Blocked exit uses fallback ---")
	testSorterBlockedFallback(ctx, engine, assignments, exitStatesBlockedFuji, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 4: Presorter — no_read box goes to error control park ---")
	testPresorterErrorControl(ctx, engine, parkStates, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 5: Presorter — least_loaded chooses lowest load ---")
	testPresorterLeastLoaded(ctx, engine, parkStates, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 6: TraceID and CorrelationID preserved ---")
	testTracePreserved(ctx, engine, assignments, exitStatesAllOK, parkStates, evalTime)

	fmt.Println()
	fmt.Println("--- TEST 7: CandidateEvaluations populated ---")
	testCandidateEvaluations(ctx, engine, assignments, exitStatesAllOK, parkStates, evalTime)

	fmt.Println()
	fmt.Println("=== ALL SMOKE TESTS PASSED ===")
}

// --- Test implementations ---

func testSorterMatch(ctx context.Context, engine *sorter.Engine, assignments sorter.AssignmentSet, exitStates []sorter.SorterExitState, evalTime time.Time) {
	box := sorter.BoxContext{
		BoxID:   "BOX-FUJI-001",
		QR:      "QR-F001",
		Variety: "Fuji",
		Caliber: "80",
	}

	dec, err := engine.ResolveSorter(ctx, sorter.SorterRequest{
		TraceID:     "smoke-trace-001",
		Box:         box,
		Assignments: assignments,
		ExitStates:  exitStates,
		EvalTime:    evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolveSorter: %v", err)
	}

	printSorterDecision(dec)

	// Validations
	assertEqual("Action", string(dec.Action), string(sorter.ActionRoute))
	assertEqual("ExitID", dec.ExitID, "exit_fuji")
	assertEqual("AssignmentID", dec.AssignmentID, "assign_fuji_80")
	assertEqual("RuleApplied", string(dec.Trace.RuleApplied), string(sorter.RuleSorterAssignmentMatched))
	assertFalse("FallbackUsed", dec.FallbackUsed)
	assertFalse("Rejected", dec.Rejected)

	fmt.Println("[OK] TEST 1 PASSED")
}

func testSorterNoMatch(ctx context.Context, engine *sorter.Engine, exitStates []sorter.SorterExitState, evalTime time.Time) {
	box := sorter.BoxContext{
		BoxID:   "BOX-UNKNOWN-001",
		QR:      "QR-U001",
		Variety: "Honeycrisp",
		Caliber: "70",
	}

	// No assignments match this box
	dec, err := engine.ResolveSorter(ctx, sorter.SorterRequest{
		TraceID:     "smoke-trace-002",
		Box:         box,
		Assignments: sorter.AssignmentSet{Assignments: []sorter.Assignment{{ID: "no_match", Enabled: true, Priority: 1, Conditions: []sorter.MatchCondition{{Field: "variety", Operator: sorter.OpEquals, Values: []string{"NonExistent"}}}, TargetType: sorter.TargetTypeExit, TargetID: "exit_fuji"}}},
		ExitStates:  exitStates,
		EvalTime:    evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolveSorter: %v", err)
	}

	printSorterDecision(dec)

	// No match → should use RejectExitID
	assertEqual("Action", string(dec.Action), string(sorter.ActionReject))
	assertEqual("ExitID", dec.ExitID, "exit_reject")
	assertEqual("RuleApplied", string(dec.Trace.RuleApplied), string(sorter.RuleSorterRejectNoMatch))

	fmt.Println("[OK] TEST 2 PASSED")
}

func testSorterBlockedFallback(ctx context.Context, engine *sorter.Engine, assignments sorter.AssignmentSet, exitStates []sorter.SorterExitState, evalTime time.Time) {
	box := sorter.BoxContext{
		BoxID:   "BOX-FUJI-002",
		QR:      "QR-F002",
		Variety: "Fuji",
		Caliber: "85",
	}

	dec, err := engine.ResolveSorter(ctx, sorter.SorterRequest{
		TraceID:     "smoke-trace-003",
		Box:         box,
		Assignments: assignments,
		ExitStates:  exitStates, // exit_fuji is blocked
		EvalTime:    evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolveSorter: %v", err)
	}

	printSorterDecision(dec)

	assertEqual("Action", string(dec.Action), string(sorter.ActionRoute))
	assertEqual("ExitID", dec.ExitID, "exit_default")
	assertTrue("FallbackUsed", dec.FallbackUsed)
	assertEqual("RuleApplied", string(dec.Trace.RuleApplied), string(sorter.RuleSorterFallbackDefaultExit))

	fmt.Println("[OK] TEST 3 PASSED")
}

func testPresorterErrorControl(ctx context.Context, engine *sorter.Engine, parkStates []sorter.ParkState, evalTime time.Time) {
	box := sorter.BoxContext{
		BoxID:      "BOX-NOREAD-001",
		QR:         "QR-NR001",
		ReadStatus: "no_read",
	}

	dec, err := engine.ResolvePresorter(ctx, sorter.PresorterRequest{
		TraceID:    "smoke-trace-004",
		Box:        box,
		ParkStates: parkStates,
		EvalTime:   evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolvePresorter: %v", err)
	}

	printPresorterDecision(dec)

	assertEqual("Action", string(dec.Action), string(sorter.ActionRoute))
	assertEqual("ParkID", dec.ParkID, "park_error")
	assertTrue("ErrorControlRequired", dec.ErrorControlRequired)
	assertTrue("ErrorControlAvailable", dec.ErrorControlAvailable)
	assertFalse("FallbackUsed", dec.FallbackUsed)

	fmt.Println("[OK] TEST 4 PASSED")
}

func testPresorterLeastLoaded(ctx context.Context, engine *sorter.Engine, parkStates []sorter.ParkState, evalTime time.Time) {
	box := sorter.BoxContext{
		BoxID:         "BOX-NORMAL-001",
		QR:            "QR-N001",
		ReadStatus:    "ok",
		QualityStatus: "approved",
	}

	dec, err := engine.ResolvePresorter(ctx, sorter.PresorterRequest{
		TraceID:    "smoke-trace-005",
		Box:        box,
		ParkStates: parkStates, // park_B has load=12, park_error=3 but error control only
		EvalTime:   evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolvePresorter: %v", err)
	}

	printPresorterDecision(dec)

	assertEqual("Action", string(dec.Action), string(sorter.ActionRoute))
	// park_error has lowest load (3) but all parks are candidates for normal box.
	// Least loaded picks park_error (load=3) since it's lowest.
	assertEqual("ParkID", dec.ParkID, "park_error")
	assertEqual("RuleApplied", string(dec.Trace.RuleApplied), string(sorter.RulePresorterLeastLoaded))
	assertFalse("ErrorControlRequired", dec.ErrorControlRequired)

	fmt.Println("[OK] TEST 5 PASSED")
}

func testTracePreserved(ctx context.Context, engine *sorter.Engine, assignments sorter.AssignmentSet, exitStates []sorter.SorterExitState, parkStates []sorter.ParkState, evalTime time.Time) {
	traceID := "trace-integration-abc-123"
	correlationID := "corr-session-xyz-789"

	box := sorter.BoxContext{BoxID: "BOX-TRACE-001", QR: "QR-T001", Variety: "Fuji", Caliber: "80"}

	// Sorter
	sorterDec, err := engine.ResolveSorter(ctx, sorter.SorterRequest{
		TraceID:       traceID,
		CorrelationID: correlationID,
		Box:           box,
		Assignments:   assignments,
		ExitStates:    exitStates,
		EvalTime:      evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolveSorter trace: %v", err)
	}
	assertEqual("Sorter.TraceID", sorterDec.Trace.TraceID, traceID)
	assertEqual("Sorter.CorrelationID", sorterDec.Trace.CorrelationID, correlationID)

	// Presorter
	presorterDec, err := engine.ResolvePresorter(ctx, sorter.PresorterRequest{
		TraceID:       traceID,
		CorrelationID: correlationID,
		Box:           box,
		ParkStates:    parkStates,
		EvalTime:      evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolvePresorter trace: %v", err)
	}
	assertEqual("Presorter.TraceID", presorterDec.Trace.TraceID, traceID)
	assertEqual("Presorter.CorrelationID", presorterDec.Trace.CorrelationID, correlationID)

	fmt.Printf("  TraceID:       %s\n", traceID)
	fmt.Printf("  CorrelationID: %s\n", correlationID)
	fmt.Println("[OK] TEST 6 PASSED")
}

func testCandidateEvaluations(ctx context.Context, engine *sorter.Engine, assignments sorter.AssignmentSet, exitStates []sorter.SorterExitState, parkStates []sorter.ParkState, evalTime time.Time) {
	box := sorter.BoxContext{BoxID: "BOX-EVAL-001", QR: "QR-E001", Variety: "Fuji", Caliber: "80"}

	// Sorter
	sorterDec, err := engine.ResolveSorter(ctx, sorter.SorterRequest{
		TraceID:     "smoke-trace-007",
		Box:         box,
		Assignments: assignments,
		ExitStates:  exitStates,
		EvalTime:    evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolveSorter candidates: %v", err)
	}
	if len(sorterDec.Trace.CandidateEvaluations) == 0 {
		log.Fatalf("FAIL: Sorter CandidateEvaluations is empty")
	}
	fmt.Printf("  Sorter CandidateEvaluations: %d entries\n", len(sorterDec.Trace.CandidateEvaluations))

	// Presorter
	presorterDec, err := engine.ResolvePresorter(ctx, sorter.PresorterRequest{
		TraceID:    "smoke-trace-008",
		Box:        box,
		ParkStates: parkStates,
		EvalTime:   evalTime,
	})
	if err != nil {
		log.Fatalf("FAIL: ResolvePresorter candidates: %v", err)
	}
	if len(presorterDec.Trace.CandidateEvaluations) == 0 {
		log.Fatalf("FAIL: Presorter CandidateEvaluations is empty")
	}
	fmt.Printf("  Presorter CandidateEvaluations: %d entries\n", len(presorterDec.Trace.CandidateEvaluations))

	fmt.Println("[OK] TEST 7 PASSED")
}

// --- Helpers ---

func printSorterDecision(dec sorter.SorterDecision) {
	fmt.Printf("  Action:       %s\n", dec.Action)
	fmt.Printf("  ExitID:       %s\n", dec.ExitID)
	fmt.Printf("  AssignmentID: %s\n", dec.AssignmentID)
	fmt.Printf("  RuleApplied:  %s\n", dec.Trace.RuleApplied)
	fmt.Printf("  Reason:       %s\n", dec.Trace.Reason)
	fmt.Printf("  FallbackUsed: %v\n", dec.FallbackUsed)
	fmt.Printf("  Rejected:     %v\n", dec.Rejected)
}

func printPresorterDecision(dec sorter.PresorterDecision) {
	fmt.Printf("  Action:               %s\n", dec.Action)
	fmt.Printf("  ParkID:               %s\n", dec.ParkID)
	fmt.Printf("  BalanceStrategy:      %s\n", dec.BalanceStrategy)
	fmt.Printf("  RuleApplied:          %s\n", dec.Trace.RuleApplied)
	fmt.Printf("  Reason:               %s\n", dec.Trace.Reason)
	fmt.Printf("  FallbackUsed:         %v\n", dec.FallbackUsed)
	fmt.Printf("  ErrorControlRequired: %v\n", dec.ErrorControlRequired)
}

func assertEqual(name, got, want string) {
	if got != want {
		log.Fatalf("FAIL: %s = %q, want %q", name, got, want)
	}
}

func assertTrue(name string, got bool) {
	if !got {
		log.Fatalf("FAIL: %s = false, want true", name)
	}
}

func assertFalse(name string, got bool) {
	if got {
		log.Fatalf("FAIL: %s = true, want false", name)
	}
}
