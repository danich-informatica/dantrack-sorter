package sorter

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers for fixed error park tests
// ---------------------------------------------------------------------------

func errorParkPresorterConfig(errorParkID string) PresorterConfig {
	return PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park_error", Enabled: true, HasErrorControl: true, Priority: 10},
			{ParkID: "park_normal_1", Enabled: true, HasErrorControl: false, Priority: 5},
			{ParkID: "park_normal_2", Enabled: true, HasErrorControl: false, Priority: 5},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park_normal_1",
		ErrorParkID:     errorParkID,
	}
}

func errorBox(readStatus string) BoxContext {
	return BoxContext{BoxID: "box_err", QR: "qr_err", ReadStatus: readStatus}
}

func normalBox() BoxContext {
	return BoxContext{BoxID: "box_ok", QR: "qr_ok", SKU: "SKU_A"}
}

// ---------------------------------------------------------------------------
// Validation tests
// ---------------------------------------------------------------------------

func TestValidation_PresorterConfig_ErrorParkID_Existing_Valid(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	if err := ValidatePresorterConfig(cfg); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidation_PresorterConfig_ErrorParkID_NonExistent_Invalid(t *testing.T) {
	cfg := errorParkPresorterConfig("park_does_not_exist")
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for non-existent ErrorParkID")
	}
	if !containsStr(err.Error(), "ErrorParkID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidation_PresorterConfig_ErrorParkID_Empty_Valid(t *testing.T) {
	cfg := errorParkPresorterConfig("")
	if err := ValidatePresorterConfig(cfg); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Routing: error boxes to fixed error park
// ---------------------------------------------------------------------------

func TestPresorter_ErrorFixedPark_NoRead(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: true, CurrentLoad: 5},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_2", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error, got %s", dec.ParkID)
	}
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
	if dec.Trace.RuleApplied != RulePresorterErrorFixedPark {
		t.Fatalf("expected RulePresorterErrorFixedPark, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_ErrorFixedPark_ErrorStatus(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("error"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: true, CurrentLoad: 5},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error, got %s", dec.ParkID)
	}
}

func TestPresorter_ErrorFixedPark_QualityBad(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box:        BoxContext{BoxID: "box_1", QR: "qr_1", QualityStatus: "bad"},
		ParkStates: []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error, got %s", dec.ParkID)
	}
}

func TestPresorter_ErrorFixedPark_ErrorFlags(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box:        BoxContext{BoxID: "box_1", QR: "qr_1", ErrorFlags: []string{"damage_detected"}},
		ParkStates: []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error, got %s", dec.ParkID)
	}
}

func TestPresorter_ErrorFixedPark_NoReadEmptyQR(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	// QR empty but BoxID present, ReadStatus = no_read
	req := PresorterRequest{
		Box:        BoxContext{BoxID: "box_1", ReadStatus: "no_read"},
		ParkStates: []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error, got %s", dec.ParkID)
	}
}

// ---------------------------------------------------------------------------
// No fallback for error boxes
// ---------------------------------------------------------------------------

func TestPresorter_ErrorFixedPark_Full_ActionPass(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, Full: true, CurrentLoad: 100},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_2", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not return Go error, got %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
	if dec.Trace.RuleApplied != RulePresorterPassErrorParkUnavailable {
		t.Fatalf("expected RulePresorterPassErrorParkUnavailable, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_ErrorFixedPark_Blocked_ActionPass(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, Blocked: true},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not return Go error, got %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
}

func TestPresorter_ErrorFixedPark_Maintenance_ActionPass(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("error"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, Maintenance: true},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not return Go error, got %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
}

func TestPresorter_ErrorFixedPark_HasError_ActionPass(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, HasError: true},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionPass must not return Go error, got %v", err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
}

func TestPresorter_ErrorFixedPark_Unavailable_DoesNotUseOtherErrorControlPark(t *testing.T) {
	// park_error is unavailable, park_normal_1 has HasErrorControl but should NOT be used
	cfg := PresorterConfig{
		Parks: []ParkConfig{
			{ParkID: "park_error", Enabled: true, HasErrorControl: true, Priority: 10},
			{ParkID: "park_other_ec", Enabled: true, HasErrorControl: true, Priority: 5},
			{ParkID: "park_normal_1", Enabled: true, HasErrorControl: false, Priority: 5},
		},
		BalanceStrategy: BalanceLeastLoaded,
		DefaultParkID:   "park_normal_1",
		ErrorParkID:     "park_error",
	}
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, Full: true},
			{ParkID: "park_other_ec", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass (no fallback to other EC park), got %s", dec.Action)
	}
	if dec.ParkID == "park_other_ec" {
		t.Fatal("must NOT fallback to another error control park")
	}
}

func TestPresorter_ErrorFixedPark_Unavailable_DoesNotUseDefaultPark(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: false, Full: true},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_2", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Action != ActionPass {
		t.Fatalf("expected ActionPass, got %s", dec.Action)
	}
	if dec.ParkID == "park_normal_1" {
		t.Fatal("must NOT use DefaultParkID when ErrorParkID is full")
	}
}

// ---------------------------------------------------------------------------
// Legacy compatibility: ErrorParkID empty → old behavior
// ---------------------------------------------------------------------------

func TestPresorter_ErrorParkIDEmpty_LegacyBehavior_PrefersHasErrorControl(t *testing.T) {
	cfg := errorParkPresorterConfig("") // no fixed error park
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: errorBox("no_read"),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: true, CurrentLoad: 50},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_2", Available: true, CurrentLoad: 0},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy: prefers HasErrorControl parks
	if dec.ParkID != "park_error" {
		t.Fatalf("expected park_error (HasErrorControl), got %s", dec.ParkID)
	}
	if dec.Trace.RuleApplied != RulePresorterErrorControlPreferred {
		t.Fatalf("expected RulePresorterErrorControlPreferred, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_NormalBox_DoesNotUseErrorParkID(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box: normalBox(),
		ParkStates: []ParkState{
			{ParkID: "park_error", Available: true, CurrentLoad: 0},
			{ParkID: "park_normal_1", Available: true, CurrentLoad: 10},
			{ParkID: "park_normal_2", Available: true, CurrentLoad: 5},
		},
		EvalTime: time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Normal box should balance among all parks (park_error has load 0, would be chosen by least_loaded)
	// This is fine — ErrorParkID logic only triggers for error boxes
	if dec.Action != ActionRoute {
		t.Fatalf("expected ActionRoute, got %s", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

func TestPresorter_ErrorFixedPark_Trace_RuleApplied(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		TraceID:       "trace-123",
		CorrelationID: "corr-456",
		Box:           errorBox("no_read"),
		ParkStates:    []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:      time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Trace.RuleApplied != RulePresorterErrorFixedPark {
		t.Fatalf("expected RulePresorterErrorFixedPark, got %s", dec.Trace.RuleApplied)
	}
	if dec.Trace.TraceID != "trace-123" {
		t.Fatalf("expected TraceID trace-123, got %s", dec.Trace.TraceID)
	}
	if dec.Trace.CorrelationID != "corr-456" {
		t.Fatalf("expected CorrelationID corr-456, got %s", dec.Trace.CorrelationID)
	}
}

func TestPresorter_ErrorFixedPark_Unavailable_Trace_RuleApplied(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box:        errorBox("no_read"),
		ParkStates: []ParkState{{ParkID: "park_error", Available: false, Full: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Trace.RuleApplied != RulePresorterPassErrorParkUnavailable {
		t.Fatalf("expected RulePresorterPassErrorParkUnavailable, got %s", dec.Trace.RuleApplied)
	}
}

func TestPresorter_ErrorFixedPark_CandidateEvaluations(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box:        errorBox("no_read"),
		ParkStates: []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, ce := range dec.Trace.CandidateEvaluations {
		if ce.TargetID == "park_error" {
			found = true
			if !ce.Eligible {
				t.Fatal("park_error should be eligible")
			}
		}
	}
	if !found {
		t.Fatal("CandidateEvaluations must contain park_error")
	}
}

func TestPresorter_ErrorFixedPark_ErrorControlFields(t *testing.T) {
	cfg := errorParkPresorterConfig("park_error")
	eng, err := NewEngine(EngineConfig{Presorter: &cfg})
	if err != nil {
		t.Fatal(err)
	}

	req := PresorterRequest{
		Box:        errorBox("no_read"),
		ParkStates: []ParkState{{ParkID: "park_error", Available: true}, {ParkID: "park_normal_1", Available: true}, {ParkID: "park_normal_2", Available: true}},
		EvalTime:   time.Now(),
	}

	dec, err := eng.ResolvePresorter(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.ErrorControlRequired {
		t.Fatal("expected ErrorControlRequired=true")
	}
	if !dec.ErrorControlAvailable {
		t.Fatal("expected ErrorControlAvailable=true (park has HasErrorControl)")
	}
}
