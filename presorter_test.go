package sorter

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers locales
// ---------------------------------------------------------------------------

func newPresorterEngine(t *testing.T, pc PresorterConfig) *Engine {
	t.Helper()
	e, err := NewEngine(EngineConfig{Presorter: &pc})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	return e
}

func defaultPresorterConfig() PresorterConfig {
	return PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true, Weight: 5},
			{ParkID: "park-B", Enabled: true, Weight: 10},
			{ParkID: "park-C", Enabled: true, Weight: 3},
			{ParkID: "park-default", Enabled: true, HasErrorControl: true, Weight: 1},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park-default",
	}
}

func presorterBox() BoxContext {
	return BoxContext{BoxID: "BOX-001", QR: "QR-001", SKU: "SKU-01"}
}

func parkAvailable(parkID string, load int) ParkState {
	return ParkState{ParkID: parkID, Available: true, CurrentLoad: load}
}

func parkBlocked(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: true, Blocked: true}
}

func parkFull(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: true, Full: true}
}

func parkError(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: true, HasError: true}
}

func parkMaintenance(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: true, Maintenance: true}
}

func parkUnavailable(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: false}
}

// ---------------------------------------------------------------------------
// Flujo principal
// ---------------------------------------------------------------------------

func TestResolvePresorter_ParkAvailable_ActionRoute(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())
	req := PresorterRequest{
		TraceID: "trace-001",
		Box:     presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 5),
			parkAvailable("park-B", 3),
			parkAvailable("park-C", 7),
			parkAvailable("park-default", 1),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ParkID == "" {
		t.Error("expected non-empty ParkID")
	}
}

// ---------------------------------------------------------------------------
// Balance strategies
// ---------------------------------------------------------------------------

func TestResolvePresorter_LeastLoaded_ChoosesLowest(t *testing.T) {
	cfg := defaultPresorterConfig()
	cfg.BalanceStrategy = BalanceLeastLoaded
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 10),
			parkAvailable("park-B", 2),
			parkAvailable("park-C", 8),
			parkAvailable("park-default", 15),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-B" {
		t.Errorf("expected park-B (load=2), got %q", dec.ParkID)
	}
}

func TestResolvePresorter_RoundRobin_RotatesBetweenParks(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceRoundRobin,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 0),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec1, _ := e.ResolvePresorter(context.Background(), req)
	dec2, _ := e.ResolvePresorter(context.Background(), req)

	if dec1.ParkID == dec2.ParkID {
		t.Errorf("expected different parks on consecutive calls, got %q both times", dec1.ParkID)
	}
}

func TestResolvePresorter_Weighted_ChoosesHighestWeight(t *testing.T) {
	cfg := defaultPresorterConfig()
	cfg.BalanceStrategy = BalanceWeighted
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 0),
			parkAvailable("park-B", 0),
			parkAvailable("park-C", 0),
			parkAvailable("park-default", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-B" {
		t.Errorf("expected park-B (weight=10), got %q", dec.ParkID)
	}
}

// ---------------------------------------------------------------------------
// Disponibilidad de parks
// ---------------------------------------------------------------------------

func TestResolvePresorter_ParkFull_NotEligible(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park-B",
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkFull("park-A"),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID == "park-A" {
		t.Error("park-A (full) should not be chosen")
	}
	if dec.ParkID != "park-B" {
		t.Errorf("expected park-B, got %q", dec.ParkID)
	}
}

func TestResolvePresorter_ParkBlocked_NotEligible(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkBlocked("park-A"),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	if dec.ParkID == "park-A" {
		t.Error("park-A (blocked) should not be chosen")
	}
}

func TestResolvePresorter_ParkWithError_NotEligible(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkError("park-A"),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	if dec.ParkID == "park-A" {
		t.Error("park-A (error) should not be chosen")
	}
}

func TestResolvePresorter_ParkInMaintenance_NotEligible(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkMaintenance("park-A"),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	if dec.ParkID == "park-A" {
		t.Error("park-A (maintenance) should not be chosen")
	}
}

func TestResolvePresorter_ParkDisabled_NotEligible(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: false},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 0),
			parkAvailable("park-B", 0),
		},
		EvalTime: time.Now(),
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	if dec.ParkID == "park-A" {
		t.Error("park-A (disabled) should not be chosen")
	}
}

func TestResolvePresorter_MissingParkState_AssumedAvailable(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box:        presorterBox(),
		ParkStates: []ParkState{}, // sin estado para park-A
		EvalTime:   time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-A" {
		t.Errorf("expected park-A (assumed available without state), got %q", dec.ParkID)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Fallback y reject
// ---------------------------------------------------------------------------

func TestResolvePresorter_DefaultParkID_UsedAsFallback(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-default", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park-default",
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkBlocked("park-A"),
			parkAvailable("park-default", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// park-A is blocked; park-default is available via normal candidates
	// Since park-default is in Parks and available, it should be in candidates.
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
}

func TestResolvePresorter_AllUnavailable_DefaultParkFallback(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
			{ParkID: "park-default", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park-default",
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkBlocked("park-A"),
			parkBlocked("park-B"),
			parkBlocked("park-default"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All parks blocked including default → ActionReject.
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject when all parks including default blocked, got %q", dec.Action)
	}
}

func TestResolvePresorter_NoValidPark_ActionReject(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkBlocked("park-A"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject, got %q", dec.Action)
	}
	if dec.Trace.RuleApplied != RulePresorterRejectNoAvailablePark {
		t.Errorf("expected RulePresorterRejectNoAvailablePark, got %q", dec.Trace.RuleApplied)
	}
}

func TestResolvePresorter_ActionReject_NoGoError(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box:        presorterBox(),
		ParkStates: []ParkState{parkFull("park-A")},
		EvalTime:   time.Now(),
	}

	_, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionReject must not produce Go error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Errores Go
// ---------------------------------------------------------------------------

func TestResolvePresorter_ContextCanceled(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.ResolvePresorter(ctx, PresorterRequest{Box: presorterBox(), EvalTime: time.Now()})
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("expected ErrContextCanceled, got: %v", err)
	}
}

func TestResolvePresorter_InvalidRequest(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())

	req := PresorterRequest{Box: BoxContext{}, EvalTime: time.Now()} // sin BoxID ni QR
	_, err := e.ResolvePresorter(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid request, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestResolvePresorter_NotConfigured(t *testing.T) {
	// Engine sin presorter config.
	e := &Engine{}

	req := PresorterRequest{Box: presorterBox(), EvalTime: time.Now()}
	_, err := e.ResolvePresorter(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when presorter not configured, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

func TestResolvePresorter_TraceIDAndCorrelationIDCopied(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())

	req := PresorterRequest{
		TraceID:       "trace-xyz",
		CorrelationID: "corr-abc",
		Box:           presorterBox(),
		ParkStates:    []ParkState{parkAvailable("park-A", 0)},
		EvalTime:      time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Trace.TraceID != "trace-xyz" {
		t.Errorf("expected TraceID trace-xyz, got %q", dec.Trace.TraceID)
	}
	if dec.Trace.CorrelationID != "corr-abc" {
		t.Errorf("expected CorrelationID corr-abc, got %q", dec.Trace.CorrelationID)
	}
}

func TestResolvePresorter_CandidateEvaluationsPopulated(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkAvailable("park-A", 5),
			parkBlocked("park-B"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dec.Trace.CandidateEvaluations) == 0 {
		t.Error("expected CandidateEvaluations to be populated")
	}

	// Verificar que park-A está elegible y park-B no.
	foundA := false
	foundB := false
	for _, ce := range dec.Trace.CandidateEvaluations {
		if ce.TargetID == "park-A" {
			foundA = true
			if !ce.Eligible {
				t.Error("park-A should be eligible")
			}
		}
		if ce.TargetID == "park-B" {
			foundB = true
			if ce.Eligible {
				t.Error("park-B should not be eligible (blocked)")
			}
		}
	}
	if !foundA {
		t.Error("park-A not found in CandidateEvaluations")
	}
	if !foundB {
		t.Error("park-B not found in CandidateEvaluations")
	}
}

// ---------------------------------------------------------------------------
// Error control
// ---------------------------------------------------------------------------

func TestResolvePresorter_ErrorFlags_PrefersErrorControlPark(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-normal", Enabled: true, HasErrorControl: false},
			{ParkID: "park-ec", Enabled: true, HasErrorControl: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	box := presorterBox()
	box.ErrorFlags = []string{"damaged_label"}

	req := PresorterRequest{
		Box: box,
		ParkStates: []ParkState{
			parkAvailable("park-normal", 0),
			parkAvailable("park-ec", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-ec" {
		t.Errorf("expected park-ec (has error control), got %q", dec.ParkID)
	}
	if !dec.ErrorControlRequired {
		t.Error("expected ErrorControlRequired=true")
	}
	if !dec.ErrorControlAvailable {
		t.Error("expected ErrorControlAvailable=true")
	}
}

func TestResolvePresorter_NoRead_PrefersErrorControlPark(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-normal", Enabled: true, HasErrorControl: false},
			{ParkID: "park-ec", Enabled: true, HasErrorControl: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	box := presorterBox()
	box.ReadStatus = "no_read"

	req := PresorterRequest{
		Box: box,
		ParkStates: []ParkState{
			parkAvailable("park-normal", 0),
			parkAvailable("park-ec", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-ec" {
		t.Errorf("expected park-ec for no_read box, got %q", dec.ParkID)
	}
	if !dec.ErrorControlRequired {
		t.Error("expected ErrorControlRequired=true")
	}
}

func TestResolvePresorter_QualityBad_PrefersErrorControlPark(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-normal", Enabled: true, HasErrorControl: false},
			{ParkID: "park-ec", Enabled: true, HasErrorControl: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	box := presorterBox()
	box.QualityStatus = "bad"

	req := PresorterRequest{
		Box: box,
		ParkStates: []ParkState{
			parkAvailable("park-normal", 0),
			parkAvailable("park-ec", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-ec" {
		t.Errorf("expected park-ec for bad quality box, got %q", dec.ParkID)
	}
}

func TestResolvePresorter_ErrorControlRequired_NoneAvailable_UsesFallback(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-normal", Enabled: true, HasErrorControl: false},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	box := presorterBox()
	box.ErrorFlags = []string{"scan_error"}

	req := PresorterRequest{
		Box: box,
		ParkStates: []ParkState{
			parkAvailable("park-normal", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ParkID != "park-normal" {
		t.Errorf("expected park-normal as fallback, got %q", dec.ParkID)
	}
	if !dec.ErrorControlRequired {
		t.Error("expected ErrorControlRequired=true")
	}
	if dec.ErrorControlAvailable {
		t.Error("expected ErrorControlAvailable=false")
	}
	if !dec.FallbackUsed {
		t.Error("expected FallbackUsed=true when no error control park available")
	}
}

func TestResolvePresorter_ErrorControlRequired_Available_Correct(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-normal", Enabled: true, HasErrorControl: false},
			{ParkID: "park-ec", Enabled: true, HasErrorControl: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
	}
	e := newPresorterEngine(t, cfg)

	box := presorterBox()
	box.ReadStatus = "ERROR" // case-insensitive

	req := PresorterRequest{
		Box: box,
		ParkStates: []ParkState{
			parkAvailable("park-normal", 0),
			parkAvailable("park-ec", 0),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.ErrorControlRequired {
		t.Error("expected ErrorControlRequired=true")
	}
	if !dec.ErrorControlAvailable {
		t.Error("expected ErrorControlAvailable=true")
	}
	if dec.FallbackUsed {
		t.Error("expected FallbackUsed=false when error control park is available")
	}
}

// ---------------------------------------------------------------------------
// EvalTime
// ---------------------------------------------------------------------------

func TestResolvePresorter_EvalTimeFromRequest(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())
	fixed := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	req := PresorterRequest{
		Box:        presorterBox(),
		ParkStates: []ParkState{parkAvailable("park-A", 0)},
		EvalTime:   fixed,
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	if !dec.EvalTime.Equal(fixed) {
		t.Errorf("expected EvalTime %v, got %v", fixed, dec.EvalTime)
	}
}

func TestResolvePresorter_EvalTimeZeroUsesNow(t *testing.T) {
	e := newPresorterEngine(t, defaultPresorterConfig())

	before := time.Now()
	req := PresorterRequest{
		Box:        presorterBox(),
		ParkStates: []ParkState{parkAvailable("park-A", 0)},
		EvalTime:   time.Time{}, // zero
	}

	dec, _ := e.ResolvePresorter(context.Background(), req)
	after := time.Now()

	if dec.EvalTime.IsZero() {
		t.Error("expected EvalTime to be set when request EvalTime is zero")
	}
	if dec.EvalTime.Before(before) || dec.EvalTime.After(after) {
		t.Errorf("expected EvalTime between %v and %v, got %v", before, after, dec.EvalTime)
	}
}

// ---------------------------------------------------------------------------
// DefaultParkID fallback con park disponible
// ---------------------------------------------------------------------------

func TestResolvePresorter_DefaultParkFallback_WhenAllOthersUnavailable(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park-A", Enabled: true},
			{ParkID: "park-B", Enabled: true},
			{ParkID: "park-default", Enabled: true},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park-default",
	}
	e := newPresorterEngine(t, cfg)

	req := PresorterRequest{
		Box: presorterBox(),
		ParkStates: []ParkState{
			parkBlocked("park-A"),
			parkBlocked("park-B"),
			// park-default sin estado → asumido disponible
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// park-default is in Parks and is assumed available (no state) → it's a candidate
	// So it should be routed normally (not via fallback path).
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ParkID != "park-default" {
		t.Errorf("expected park-default, got %q", dec.ParkID)
	}
}
