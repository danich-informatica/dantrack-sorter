package sorter

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func globalBalancePresorterConfig() PresorterConfig {
	return PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park_1", Enabled: true, Priority: 10},
			{ParkID: "park_2", Enabled: true, Priority: 10},
			{ParkID: "park_3", Enabled: true, Priority: 10},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park_1",
	}
}

func gbBox() BoxContext {
	return BoxContext{BoxID: "box_1", QR: "qr_1", SKU: "SKU_A"}
}

// ---------------------------------------------------------------------------
// Types / Validation
// ---------------------------------------------------------------------------

func TestParkState_AccumulatedLoad_FieldExists(t *testing.T) {
	ps := ParkState{ParkID: "p1", Available: true, AccumulatedLoad: 500}
	if ps.AccumulatedLoad != 500 {
		t.Fatalf("expected 500, got %d", ps.AccumulatedLoad)
	}
}

func TestValidation_PresorterConfig_WithAccumulatedLoad_StillValid(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	if err := ValidatePresorterConfig(cfg); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Global Balance: AccumulatedLoad
// ---------------------------------------------------------------------------

func TestPresorter_GlobalBalance_PicksLowerAccumulated(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
			{ParkID: "park_2", Available: true, CurrentLoad: 10, AccumulatedLoad: 80},
			{ParkID: "park_3", Available: true, CurrentLoad: 10, AccumulatedLoad: 120},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_2" {
		t.Fatalf("expected park_2 (lowest accumulated=80), got %s", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterGlobalBalance {
		t.Fatalf("expected RulePresorterGlobalBalance, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_GlobalBalance_TieAccumulated_UsesCurrentLoad(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 30, AccumulatedLoad: 100},
			{ParkID: "park_2", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
			{ParkID: "park_3", Available: true, CurrentLoad: 50, AccumulatedLoad: 100},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Tie in accumulated → pick lower CurrentLoad
	if dec.ParkID != "park_2" {
		t.Fatalf("expected park_2 (tie accumulated, lowest current=10), got %s", dec.ParkID)
	}
}

func TestPresorter_GlobalBalance_TieBoth_RespectsOrder(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
			{ParkID: "park_2", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
			{ParkID: "park_3", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// All tied → first in config order (park_1, index=0)
	if dec.ParkID != "park_1" {
		t.Fatalf("expected park_1 (stable order), got %s", dec.ParkID)
	}
}

func TestPresorter_GlobalBalance_AllZeroAccumulated_UsesLeastLoaded(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 50, AccumulatedLoad: 0},
			{ParkID: "park_2", Available: true, CurrentLoad: 20, AccumulatedLoad: 0},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// No accumulated data → legacy least_loaded by CurrentLoad
	if dec.ParkID != "park_2" {
		t.Fatalf("expected park_2 (least current=20, no accumulated), got %s", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterLeastLoaded {
		t.Fatalf("expected RulePresorterLeastLoaded (legacy), got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_GlobalBalance_OneHasAccumulated_OtherZero_PicksZero(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// park_1 has accumulated=100, park_2 has accumulated=0
	// 0 is treated as real value → park_2 wins
	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 5, AccumulatedLoad: 100},
			{ParkID: "park_2", Available: true, CurrentLoad: 50, AccumulatedLoad: 0},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Global balance triggered (park_1 has AccumulatedLoad > 0).
	// park_2 and park_3 have AccumulatedLoad=0 (treated as real).
	// Tie between park_2 and park_3 on accumulated=0 → use CurrentLoad.
	// park_3 CurrentLoad=30 < park_2 CurrentLoad=50.
	if dec.ParkID != "park_3" {
		t.Fatalf("expected park_3 (accumulated=0, lowest current=30), got %s", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterGlobalBalance {
		t.Fatalf("expected RulePresorterGlobalBalance, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_GlobalBalance_FullBlockedExcluded(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: false, Full: true, CurrentLoad: 100, AccumulatedLoad: 50},
			{ParkID: "park_2", Available: false, Blocked: true, CurrentLoad: 0, AccumulatedLoad: 30},
			{ParkID: "park_3", Available: true, CurrentLoad: 40, AccumulatedLoad: 200},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Only park_3 is available
	if dec.ParkID != "park_3" {
		t.Fatalf("expected park_3 (only available), got %s", dec.ParkID)
	}
}

func TestPresorter_GlobalBalance_ErrorParkIDHasPriority(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park_error", Enabled: true, HasErrorControl: true, Priority: 10},
			{ParkID: "park_1", Enabled: true, Priority: 5},
			{ParkID: "park_2", Enabled: true, Priority: 5},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park_1",
		ErrorParkID:     "park_error",
	}
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// Error box with AccumulatedLoad data → ErrorParkID still has priority
	req := PresorterRequest{
		Box: BoxContext{BoxID: "b1", QR: "q1", ReadStatus: "no_read"},
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: true, CurrentLoad: 200, AccumulatedLoad: 999},
			{ParkID: "park_1", Available: true, CurrentLoad: 0, AccumulatedLoad: 0},
			{ParkID: "park_2", Available: true, CurrentLoad: 0, AccumulatedLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error (ErrorParkID priority), got %s", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterErrorFixedPark {
		t.Fatalf("expected RulePresorterErrorFixedPark, got %s", dec.Trace.RuleApplied)
	}
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

func TestPresorter_GlobalBalance_TraceRuleApplied(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		TraceID:       "t-gb-1",
		CorrelationID: "c-gb-1",
		Box:           gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 10, AccumulatedLoad: 200},
			{ParkID: "park_2", Available: true, CurrentLoad: 5, AccumulatedLoad: 50},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 150},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Trace.RuleApplied != RulePresorterGlobalBalance {
		t.Fatalf("expected RulePresorterGlobalBalance, got %s", dec.Trace.RuleApplied)
	}
	if dec.Trace.TraceID != "t-gb-1" {
		t.Fatalf("TraceID not preserved: %s", dec.Trace.TraceID)
	}
	if dec.Trace.CorrelationID != "c-gb-1" {
		t.Fatalf("CorrelationID not preserved: %s", dec.Trace.CorrelationID)
	}
}

func TestPresorter_GlobalBalance_TraceReasonMentionsAccumulated(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 10, AccumulatedLoad: 200},
			{ParkID: "park_2", Available: true, CurrentLoad: 5, AccumulatedLoad: 50},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 150},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(dec.Trace.Reason, "accumulated") {
		t.Fatalf("Reason should mention accumulated, got %q", dec.Trace.Reason)
	}
}

func TestPresorter_GlobalBalance_LegacyReasonMentionsCurrent(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 50, AccumulatedLoad: 0},
			{ParkID: "park_2", Available: true, CurrentLoad: 20, AccumulatedLoad: 0},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(dec.Trace.Reason, "current=") {
		t.Fatalf("Reason should mention current, got %q", dec.Trace.Reason)
	}
}

func TestPresorter_GlobalBalance_CandidateEvaluationsPopulated(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: gbBox(),
		ParkStates: []ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 10, AccumulatedLoad: 100},
			{ParkID: "park_2", Available: true, CurrentLoad: 5, AccumulatedLoad: 50},
			{ParkID: "park_3", Available: true, CurrentLoad: 30, AccumulatedLoad: 150},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(dec.Trace.CandidateEvaluations) < 3 {
		t.Fatalf("expected at least 3 candidate evaluations, got %d", len(dec.Trace.CandidateEvaluations))
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestPresorter_GlobalBalance_ConcurrentNoRace(t *testing.T) {
	cfg := globalBalancePresorterConfig()
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := PresorterRequest{
				Box: gbBox(),
				ParkStates: []ParkState{
					{ParkID: "park_1", Available: true, CurrentLoad: i, AccumulatedLoad: i * 10},
					{ParkID: "park_2", Available: true, CurrentLoad: i + 1, AccumulatedLoad: (i + 1) * 10},
					{ParkID: "park_3", Available: true, CurrentLoad: i + 2, AccumulatedLoad: (i + 2) * 10},
				},
				EvalTime: time.Now(),
			}
			dec, err := eng.ResolvePresorter(context.Background(), req)
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, err)
			}
			if dec.Action != ActionRoute {
				t.Errorf("goroutine %d: expected ActionRoute, got %s", i, dec.Action)
			}
		}(i)
	}
	wg.Wait()
}
