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

func newSorterEngine(t *testing.T, sc SorterConfig) *Engine {
	t.Helper()
	e, err := NewEngine(EngineConfig{Sorter: &sc})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	return e
}

func defaultSorterConfig() SorterConfig {
	return SorterConfig{
		SorterID: "sorter-01",
		Exits: []SorterExit{
			{ExitID: "exit-A", Enabled: true},
			{ExitID: "exit-B", Enabled: true},
			{ExitID: "exit-default", Enabled: true},
			{ExitID: "exit-reject", Enabled: true},
		},
		DefaultExitID: "exit-default",
		RejectExitID:  "exit-reject",
	}
}

func availableState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true}
}

func unavailableState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: false}
}

func blockedState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, Blocked: true}
}

func fullState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, Full: true}
}

func errorState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, HasError: true}
}

func maintenanceState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, Maintenance: true}
}

func assignmentTo(id, exitID string, priority int) Assignment {
	return Assignment{
		ID:         id,
		Enabled:    true,
		Priority:   priority,
		TargetType: TargetTypeExit,
		TargetID:   exitID,
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpExists},
		},
	}
}

func skuAssignment(id, sku, exitID string, priority int) Assignment {
	return Assignment{
		ID:         id,
		Enabled:    true,
		Priority:   priority,
		TargetType: TargetTypeExit,
		TargetID:   exitID,
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpEquals, Values: []string{sku}},
		},
	}
}

func boxWithSKU(sku string) BoxContext {
	return BoxContext{BoxID: "BOX-001", QR: "QR-001", SKU: sku}
}

// ---------------------------------------------------------------------------
// TestResolveSorter — flujo principal
// ---------------------------------------------------------------------------

// TestResolveSorter_DirectMatch_ActionRoute: asignación matchea, exit disponible → Route.
func TestResolveSorter_DirectMatch_ActionRoute(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		TraceID:       "trace-001",
		CorrelationID: "corr-001",
		Box:           boxWithSKU("SKU-A"),
		Assignments:   AssignmentSet{Assignments: []Assignment{assignmentTo("asgn-1", "exit-A", 10)}},
		ExitStates:    []SorterExitState{availableState("exit-A")},
		EvalTime:      time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ExitID != "exit-A" {
		t.Errorf("expected ExitID exit-A, got %q", dec.ExitID)
	}
	if dec.AssignmentID != "asgn-1" {
		t.Errorf("expected AssignmentID asgn-1, got %q", dec.AssignmentID)
	}
	if dec.FallbackUsed {
		t.Error("expected FallbackUsed=false")
	}
	if dec.Rejected {
		t.Error("expected Rejected=false")
	}
	if dec.Trace.RuleApplied != RuleSorterAssignmentMatched {
		t.Errorf("expected RuleSorterAssignmentMatched, got %q", dec.Trace.RuleApplied)
	}
}

// TestResolveSorter_HigherPriorityMatchChosen: dos asignaciones; la de mayor prioridad se usa.
func TestResolveSorter_HigherPriorityMatchChosen(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-low", "exit-B", 5),
			assignmentTo("asgn-high", "exit-A", 20),
		}},
		ExitStates: []SorterExitState{
			availableState("exit-A"),
			availableState("exit-B"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ExitID != "exit-A" {
		t.Errorf("expected high-priority exit exit-A, got %q", dec.ExitID)
	}
	if dec.AssignmentID != "asgn-high" {
		t.Errorf("expected asgn-high, got %q", dec.AssignmentID)
	}
}

// TestResolveSorter_BlockedExitFallsToNextMatch: primer exit bloqueado → segundo match usado.
func TestResolveSorter_BlockedExitFallsToNextMatch(t *testing.T) {
	sc := defaultSorterConfig()
	sc.DefaultExitID = "" // quitar default para claridad
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 20),
			assignmentTo("asgn-2", "exit-B", 10),
		}},
		ExitStates: []SorterExitState{
			blockedState("exit-A"),
			availableState("exit-B"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ExitID != "exit-B" {
		t.Errorf("expected fallback to exit-B, got %q", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
}

// TestResolveSorter_AllMatchedExitsUnavailable_UsesDefaultExit: todos los matches bloqueados
// → se usa DefaultExitID con FallbackUsed=true.
func TestResolveSorter_AllMatchedExitsUnavailable_UsesDefaultExit(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{
			blockedState("exit-A"),
			availableState("exit-default"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ExitID != "exit-default" {
		t.Errorf("expected exit-default, got %q", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if !dec.FallbackUsed {
		t.Error("expected FallbackUsed=true")
	}
	if dec.Trace.RuleApplied != RuleSorterFallbackDefaultExit {
		t.Errorf("expected RuleSorterFallbackDefaultExit, got %q", dec.Trace.RuleApplied)
	}
}

// TestResolveSorter_AllMatchedExitsUnavailable_NoDefault_RejectsNoAvailableExit: sin default
// → ActionReject con RuleSorterRejectNoAvailableExit.
func TestResolveSorter_AllMatchedExitsUnavailable_NoDefault_RejectsNoAvailableExit(t *testing.T) {
	sc := defaultSorterConfig()
	sc.DefaultExitID = ""
	sc.RejectExitID = ""
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{
			blockedState("exit-A"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject, got %q", dec.Action)
	}
	if !dec.Rejected {
		t.Error("expected Rejected=true")
	}
	if dec.Trace.RuleApplied != RuleSorterRejectNoAvailableExit {
		t.Errorf("expected RuleSorterRejectNoAvailableExit, got %q", dec.Trace.RuleApplied)
	}
}

// TestResolveSorter_NoMatch_RejectExitAvailable_ActionReject: ninguna asignación matchea,
// RejectExitID disponible → ActionReject con RuleSorterRejectNoMatch.
func TestResolveSorter_NoMatch_RejectExitAvailable_ActionReject(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-UNKNOWN"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			skuAssignment("asgn-1", "SKU-OTHER", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{
			availableState("exit-reject"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject, got %q", dec.Action)
	}
	if dec.ExitID != "exit-reject" {
		t.Errorf("expected exit-reject, got %q", dec.ExitID)
	}
	if !dec.Rejected {
		t.Error("expected Rejected=true")
	}
	if dec.FallbackUsed {
		t.Error("expected FallbackUsed=false for reject exit")
	}
	if dec.Trace.RuleApplied != RuleSorterRejectNoMatch {
		t.Errorf("expected RuleSorterRejectNoMatch, got %q", dec.Trace.RuleApplied)
	}
}

// TestResolveSorter_NoMatch_NoRejectExit_DefaultAvailable_ActionRoute: sin match,
// sin RejectExitID disponible, DefaultExitID disponible → ActionRoute FallbackUsed=true.
func TestResolveSorter_NoMatch_NoRejectExit_DefaultAvailable_ActionRoute(t *testing.T) {
	sc := defaultSorterConfig()
	sc.RejectExitID = ""
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-UNKNOWN"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			skuAssignment("asgn-1", "SKU-OTHER", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{
			availableState("exit-default"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	if dec.ExitID != "exit-default" {
		t.Errorf("expected exit-default, got %q", dec.ExitID)
	}
	if !dec.FallbackUsed {
		t.Error("expected FallbackUsed=true")
	}
	if dec.Trace.RuleApplied != RuleSorterRejectNoMatch {
		t.Errorf("expected RuleSorterRejectNoMatch, got %q", dec.Trace.RuleApplied)
	}
}

// TestResolveSorter_NoMatch_NoExits_ActionReject: sin match y sin exits disponibles
// → ActionReject con RuleSorterRejectNoAvailableExit.
func TestResolveSorter_NoMatch_NoExits_ActionReject(t *testing.T) {
	sc := defaultSorterConfig()
	sc.DefaultExitID = ""
	sc.RejectExitID = ""
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box:         boxWithSKU("SKU-UNKNOWN"),
		Assignments: AssignmentSet{Assignments: []Assignment{skuAssignment("asgn-1", "SKU-OTHER", "exit-A", 10)}},
		EvalTime:    time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject, got %q", dec.Action)
	}
	if dec.Trace.RuleApplied != RuleSorterRejectNoAvailableExit {
		t.Errorf("expected RuleSorterRejectNoAvailableExit, got %q", dec.Trace.RuleApplied)
	}
}

// ---------------------------------------------------------------------------
// Disponibilidad de exits
// ---------------------------------------------------------------------------

func TestResolveSorter_ExitFull_NotAvailable(t *testing.T) {
	testExitStateNotAvailable(t, fullState("exit-A"))
}

func TestResolveSorter_ExitHasError_NotAvailable(t *testing.T) {
	testExitStateNotAvailable(t, errorState("exit-A"))
}

func TestResolveSorter_ExitInMaintenance_NotAvailable(t *testing.T) {
	testExitStateNotAvailable(t, maintenanceState("exit-A"))
}

func TestResolveSorter_ExitAvailableFalse_NotAvailable(t *testing.T) {
	testExitStateNotAvailable(t, unavailableState("exit-A"))
}

// testExitStateNotAvailable es un helper para verificar que un estado operativo específico
// hace que el exit no sea elegible y la decisión caiga en el DefaultExitID.
func testExitStateNotAvailable(t *testing.T, state SorterExitState) {
	t.Helper()
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{
			state,
			availableState("exit-default"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// exit-A no disponible → DefaultExitID usado
	if dec.ExitID == "exit-A" {
		t.Errorf("exit-A should not be used when state is %+v", state)
	}
	if dec.ExitID != "exit-default" {
		t.Errorf("expected fallback to exit-default, got %q", dec.ExitID)
	}
}

// TestResolveSorter_MissingExitState_AssumedAvailable: sin estado para el exit
// → se asume disponible (política documentada).
func TestResolveSorter_MissingExitState_AssumedAvailable(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
		}},
		ExitStates: []SorterExitState{}, // sin estado para exit-A
		EvalTime:   time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ExitID != "exit-A" {
		t.Errorf("expected exit-A (assumed available without state), got %q", dec.ExitID)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Ambigüedad
// ---------------------------------------------------------------------------

// TestResolveSorter_Ambiguity_PolicyError: dos asignaciones de igual prioridad
// apuntan a diferentes exits + policy=error → ErrAmbiguousAssignment.
func TestResolveSorter_Ambiguity_PolicyError(t *testing.T) {
	sc := defaultSorterConfig()
	sc.AmbiguityPolicy = AmbiguityPolicyError
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
			assignmentTo("asgn-2", "exit-B", 10), // misma prioridad, exit diferente
		}},
		ExitStates: []SorterExitState{
			availableState("exit-A"),
			availableState("exit-B"),
		},
		EvalTime: time.Now(),
	}

	_, err := e.ResolveSorter(context.Background(), req)
	if err == nil {
		t.Fatal("expected ErrAmbiguousAssignment, got nil")
	}
	if !errors.Is(err, ErrAmbiguousAssignment) {
		t.Errorf("expected ErrAmbiguousAssignment, got: %v", err)
	}
}

// TestResolveSorter_Ambiguity_PolicyFirstWins: igual prioridad, exits distintos,
// policy=first_wins → usa el primer match → ActionRoute.
func TestResolveSorter_Ambiguity_PolicyFirstWins(t *testing.T) {
	sc := defaultSorterConfig()
	sc.AmbiguityPolicy = AmbiguityPolicyFirstWins
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
			assignmentTo("asgn-2", "exit-B", 10),
		}},
		ExitStates: []SorterExitState{
			availableState("exit-A"),
			availableState("exit-B"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with first_wins policy: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
	// El primer match en orden estable debe ser usado.
	if dec.ExitID == "" {
		t.Error("expected a non-empty ExitID")
	}
}

// TestResolveSorter_Ambiguity_SameExit_NoError: dos asignaciones de igual prioridad
// apuntan al MISMO exit → no es ambigüedad real → no error.
func TestResolveSorter_Ambiguity_SameExit_NoError(t *testing.T) {
	sc := defaultSorterConfig()
	sc.AmbiguityPolicy = AmbiguityPolicyError
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
			assignmentTo("asgn-2", "exit-A", 10), // mismo exit → no ambigüedad
		}},
		ExitStates: []SorterExitState{availableState("exit-A")},
		EvalTime:   time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error when same exit, equal priority: %v", err)
	}
	if dec.Action != ActionRoute {
		t.Errorf("expected ActionRoute, got %q", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Errores Go
// ---------------------------------------------------------------------------

// TestResolveSorter_ContextCanceled_ReturnsErrContextCanceled.
func TestResolveSorter_ContextCanceled_ReturnsErrContextCanceled(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.ResolveSorter(ctx, SorterRequest{Box: validBox(), EvalTime: time.Now()})
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("expected ErrContextCanceled, got: %v", err)
	}
}

// TestResolveSorter_InvalidRequest_ReturnsErrInvalidRequest.
func TestResolveSorter_InvalidRequest_ReturnsErrInvalidRequest(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	// Box sin BoxID ni QR es inválido.
	req := SorterRequest{Box: BoxContext{}, EvalTime: time.Now()}
	_, err := e.ResolveSorter(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid request, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) && !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrInvalidRequest or ErrMissingRequiredField, got: %v", err)
	}
}

// TestResolveSorter_SorterNotConfigured_ReturnsErrInvalidConfig.
func TestResolveSorter_SorterNotConfigured_ReturnsErrInvalidConfig(t *testing.T) {
	e := &Engine{} // sin sorterCfg

	req := SorterRequest{Box: validBox(), EvalTime: time.Now()}
	_, err := e.ResolveSorter(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when sorter not configured, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

// TestResolveSorter_ActionRejectIsNotGoError: un ActionReject normal no produce error Go.
func TestResolveSorter_ActionRejectIsNotGoError(t *testing.T) {
	sc := defaultSorterConfig()
	sc.DefaultExitID = ""
	sc.RejectExitID = ""
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box:         boxWithSKU("SKU-UNKNOWN"),
		Assignments: AssignmentSet{Assignments: []Assignment{skuAssignment("a", "OTHER", "exit-A", 10)}},
		EvalTime:    time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("ActionReject must not produce Go error, got: %v", err)
	}
	if dec.Action != ActionReject {
		t.Errorf("expected ActionReject, got %q", dec.Action)
	}
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

// TestResolveSorter_TraceIDAndCorrelationIDCopied: IDs del orquestador se propagan a la traza.
func TestResolveSorter_TraceIDAndCorrelationIDCopied(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		TraceID:       "trace-xyz",
		CorrelationID: "corr-abc",
		Box:           boxWithSKU("SKU-A"),
		Assignments:   AssignmentSet{Assignments: []Assignment{assignmentTo("a", "exit-A", 1)}},
		ExitStates:    []SorterExitState{availableState("exit-A")},
		EvalTime:      time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
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

// TestResolveSorter_CandidateEvaluationsPopulated: la traza incluye evaluaciones de candidatos.
func TestResolveSorter_CandidateEvaluationsPopulated(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-A", 10),
			skuAssignment("asgn-2", "NOMATCH", "exit-B", 5),
		}},
		ExitStates: []SorterExitState{availableState("exit-A")},
		EvalTime:   time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dec.Trace.CandidateEvaluations) == 0 {
		t.Error("expected CandidateEvaluations to be populated")
	}
}

// TestResolveSorter_SorterIDPopulated: el SorterID de la config se copia en la decisión.
func TestResolveSorter_SorterIDPopulated(t *testing.T) {
	sc := defaultSorterConfig()
	sc.SorterID = "my-sorter"
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box:         boxWithSKU("SKU-UNKNOWN"),
		Assignments: AssignmentSet{},
		EvalTime:    time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.SorterID != "my-sorter" {
		t.Errorf("expected SorterID my-sorter, got %q", dec.SorterID)
	}
}

// TestResolveSorter_EvalTimeZeroUsesNow: EvalTime cero → se usa time.Now() en la decisión.
func TestResolveSorter_EvalTimeZeroUsesNow(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	before := time.Now()
	req := SorterRequest{
		Box:      boxWithSKU("SKU-A"),
		EvalTime: time.Time{}, // cero
	}
	dec, _ := e.ResolveSorter(context.Background(), req)
	after := time.Now()

	if dec.EvalTime.IsZero() {
		t.Error("expected EvalTime to be set when request EvalTime is zero")
	}
	if dec.EvalTime.Before(before) || dec.EvalTime.After(after) {
		t.Errorf("expected EvalTime between %v and %v, got %v", before, after, dec.EvalTime)
	}
}

// TestResolveSorter_EvalTimePropagated: EvalTime no-cero se preserva en la decisión.
func TestResolveSorter_EvalTimePropagated(t *testing.T) {
	sc := defaultSorterConfig()
	e := newSorterEngine(t, sc)

	fixed := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	req := SorterRequest{
		Box:      boxWithSKU("SKU-A"),
		EvalTime: fixed,
	}
	dec, _ := e.ResolveSorter(context.Background(), req)

	if !dec.EvalTime.Equal(fixed) {
		t.Errorf("expected EvalTime %v, got %v", fixed, dec.EvalTime)
	}
}

// TestResolveSorter_DisabledExitInConfig_NotAvailable: exit en config pero con Enabled=false
// → no debe ser elegido.
func TestResolveSorter_DisabledExitInConfig_NotAvailable(t *testing.T) {
	sc := SorterConfig{
		SorterID: "sorter-01",
		Exits: []SorterExit{
			{ExitID: "exit-disabled", Enabled: false},
			{ExitID: "exit-default", Enabled: true},
		},
		DefaultExitID: "exit-default",
	}
	e := newSorterEngine(t, sc)

	req := SorterRequest{
		Box: boxWithSKU("SKU-A"),
		Assignments: AssignmentSet{Assignments: []Assignment{
			assignmentTo("asgn-1", "exit-disabled", 10),
		}},
		ExitStates: []SorterExitState{
			availableState("exit-disabled"), // el estado dice available pero la config lo deshabilita
			availableState("exit-default"),
		},
		EvalTime: time.Now(),
	}

	dec, err := e.ResolveSorter(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.ExitID == "exit-disabled" {
		t.Error("disabled exit should not be chosen even if state says available")
	}
	if dec.ExitID != "exit-default" {
		t.Errorf("expected fallback to exit-default, got %q", dec.ExitID)
	}
}
