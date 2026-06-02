package sorter

import (
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func validPark(id string) ParkConfig {
	return ParkConfig{ParkID: id, Enabled: true}
}

func validExit(id string) SorterExit {
	return SorterExit{ExitID: id, Enabled: true}
}

func validBox() BoxContext {
	return BoxContext{BoxID: "BOX-001", QR: "QR-001"}
}

func validPresorterConfig() PresorterConfig {
	return PresorterConfig{
		Parks:           []ParkConfig{validPark("park-a"), validPark("park-b")},
		BalanceStrategy: BalanceLeastLoaded,
	}
}

func validSorterConfig() SorterConfig {
	return SorterConfig{
		SorterID: "sorter-01",
		Exits:    []SorterExit{validExit("exit-1"), validExit("exit-2")},
	}
}

func validAssignment(id string) Assignment {
	return Assignment{
		ID:         id,
		Enabled:    true,
		Priority:   1,
		TargetType: TargetTypeExit,
		TargetID:   "exit-1",
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpEquals, Values: []string{"SKU-01"}},
		},
	}
}

// ---------------------------------------------------------------------------
// ValidateConfig
// ---------------------------------------------------------------------------

func TestValidateConfig_ValidPresorterOnly(t *testing.T) {
	cfg := EngineConfig{Presorter: ptr(validPresorterConfig())}
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_ValidSorterOnly(t *testing.T) {
	cfg := EngineConfig{Sorter: ptr(validSorterConfig())}
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_BothConfigured(t *testing.T) {
	cfg := EngineConfig{
		Presorter: ptr(validPresorterConfig()),
		Sorter:    ptr(validSorterConfig()),
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_NeitherConfigured(t *testing.T) {
	cfg := EngineConfig{}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty EngineConfig, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidatePresorterConfig
// ---------------------------------------------------------------------------

func TestValidatePresorterConfig_Valid(t *testing.T) {
	cfg := validPresorterConfig()
	if err := ValidatePresorterConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidatePresorterConfig_NoParks(t *testing.T) {
	cfg := PresorterConfig{Parks: nil}
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty parks, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidatePresorterConfig_EmptyParkID(t *testing.T) {
	cfg := PresorterConfig{Parks: []ParkConfig{{ParkID: ""}}}
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty ParkID, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidatePresorterConfig_DuplicateParkID(t *testing.T) {
	cfg := PresorterConfig{
		Parks: []ParkConfig{validPark("park-a"), validPark("park-a")},
	}
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate ParkID, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidatePresorterConfig_UnknownBalanceStrategy(t *testing.T) {
	cfg := PresorterConfig{
		Parks:           []ParkConfig{validPark("park-a")},
		BalanceStrategy: "unknown_strategy",
	}
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown BalanceStrategy, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidatePresorterConfig_DefaultParkIDNotFound(t *testing.T) {
	cfg := PresorterConfig{
		Parks:         []ParkConfig{validPark("park-a")},
		DefaultParkID: "park-nonexistent",
	}
	err := ValidatePresorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown DefaultParkID, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidatePresorterConfig_DefaultParkIDValid(t *testing.T) {
	cfg := PresorterConfig{
		Parks:         []ParkConfig{validPark("park-a"), validPark("park-b")},
		DefaultParkID: "park-b",
	}
	if err := ValidatePresorterConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateSorterConfig
// ---------------------------------------------------------------------------

func TestValidateSorterConfig_Valid(t *testing.T) {
	cfg := validSorterConfig()
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSorterConfig_EmptySorterID(t *testing.T) {
	cfg := SorterConfig{Exits: []SorterExit{validExit("exit-1")}}
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty SorterID, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidateSorterConfig_NoExits(t *testing.T) {
	cfg := SorterConfig{SorterID: "sorter-01", Exits: nil}
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty exits, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidateSorterConfig_EmptyExitID(t *testing.T) {
	cfg := SorterConfig{
		SorterID: "sorter-01",
		Exits:    []SorterExit{{ExitID: ""}},
	}
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty ExitID, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidateSorterConfig_DuplicateExitID(t *testing.T) {
	cfg := SorterConfig{
		SorterID: "sorter-01",
		Exits:    []SorterExit{validExit("exit-1"), validExit("exit-1")},
	}
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate ExitID, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidateSorterConfig_DefaultExitIDNotFound(t *testing.T) {
	cfg := SorterConfig{
		SorterID:      "sorter-01",
		Exits:         []SorterExit{validExit("exit-1")},
		DefaultExitID: "exit-nonexistent",
	}
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown DefaultExitID, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidatePresorterRequest
// ---------------------------------------------------------------------------

func TestValidatePresorterRequest_Valid(t *testing.T) {
	req := PresorterRequest{
		TraceID:  "trace-001",
		Box:      validBox(),
		EvalTime: time.Now(),
		ParkStates: []ParkState{
			{ParkID: "park-a", Available: true},
		},
	}
	if err := ValidatePresorterRequest(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidatePresorterRequest_EmptyBoxIDAndQR(t *testing.T) {
	req := PresorterRequest{
		Box:      BoxContext{BoxID: "", QR: ""},
		EvalTime: time.Now(),
	}
	err := ValidatePresorterRequest(req)
	if err == nil {
		t.Fatal("expected error for empty BoxID and QR, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidatePresorterRequest_OnlyBoxID(t *testing.T) {
	req := PresorterRequest{
		Box: BoxContext{BoxID: "BOX-001"},
	}
	if err := ValidatePresorterRequest(req); err != nil {
		t.Errorf("expected no error when BoxID present, got: %v", err)
	}
}

func TestValidatePresorterRequest_OnlyQR(t *testing.T) {
	req := PresorterRequest{
		Box: BoxContext{QR: "QR-001"},
	}
	if err := ValidatePresorterRequest(req); err != nil {
		t.Errorf("expected no error when QR present, got: %v", err)
	}
}

func TestValidatePresorterRequest_DuplicateParkState(t *testing.T) {
	req := PresorterRequest{
		Box: validBox(),
		ParkStates: []ParkState{
			{ParkID: "park-a"},
			{ParkID: "park-a"},
		},
	}
	err := ValidatePresorterRequest(req)
	if err == nil {
		t.Fatal("expected error for duplicate ParkState, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateSorterRequest
// ---------------------------------------------------------------------------

func TestValidateSorterRequest_Valid(t *testing.T) {
	req := SorterRequest{
		TraceID:  "trace-002",
		Box:      validBox(),
		EvalTime: time.Now(),
		Assignments: AssignmentSet{
			Assignments: []Assignment{validAssignment("asgn-01")},
		},
		ExitStates: []SorterExitState{
			{ExitID: "exit-1", Available: true},
		},
	}
	if err := ValidateSorterRequest(req); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSorterRequest_EmptyBoxIDAndQR(t *testing.T) {
	req := SorterRequest{
		Box: BoxContext{},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for empty BoxID and QR, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidateSorterRequest_EmptyAssignments(t *testing.T) {
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{},
	}
	if err := ValidateSorterRequest(req); err != nil {
		t.Errorf("expected no error for empty AssignmentSet, got: %v", err)
	}
}

func TestValidateSorterRequest_DuplicateExitState(t *testing.T) {
	req := SorterRequest{
		Box: validBox(),
		ExitStates: []SorterExitState{
			{ExitID: "exit-1"},
			{ExitID: "exit-1"},
		},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for duplicate ExitState, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Assignment validation
// ---------------------------------------------------------------------------

func TestValidateAssignment_InvalidOperator(t *testing.T) {
	a := validAssignment("asgn-01")
	a.Conditions = []MatchCondition{
		{Field: "sku", Operator: "unknown_op", Values: []string{"val"}},
	}
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for unknown operator, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidateAssignment_InvalidTemporalRange(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	a := validAssignment("asgn-02")
	a.ValidFrom = now.Add(2 * time.Hour)
	a.ValidTo = now.Add(1 * time.Hour) // ValidTo antes de ValidFrom
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for invalid temporal range, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestValidateAssignment_EmptyID(t *testing.T) {
	a := validAssignment("")
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for empty assignment ID, got nil")
	}
	if !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrMissingRequiredField, got: %v", err)
	}
}

func TestValidateAssignment_ExistsOperatorNoValues(t *testing.T) {
	a := validAssignment("asgn-03")
	a.Conditions = []MatchCondition{
		{Field: "sku", Operator: OpExists, Values: nil}, // exists no requiere valores
	}
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
	}
	if err := ValidateSorterRequest(req); err != nil {
		t.Errorf("expected no error for exists operator without values, got: %v", err)
	}
}

func TestValidateAssignment_EqualsOperatorNoValues(t *testing.T) {
	a := validAssignment("asgn-04")
	a.Conditions = []MatchCondition{
		{Field: "sku", Operator: OpEquals, Values: nil}, // equals requiere valores
	}
	req := SorterRequest{
		Box:         validBox(),
		Assignments: AssignmentSet{Assignments: []Assignment{a}},
	}
	err := ValidateSorterRequest(req)
	if err == nil {
		t.Fatal("expected error for equals without values, got nil")
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("expected ErrInvalidRequest, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Error sentinel distinguishability
// ---------------------------------------------------------------------------

func TestErrors_AreDistinguishable(t *testing.T) {
	errs := []error{
		ErrNoAssignmentMatch,
		ErrNoAvailableExit,
		ErrNoAvailablePark,
		ErrInvalidConfig,
		ErrInvalidRequest,
		ErrAmbiguousAssignment,
		ErrMissingRequiredField,
		ErrContextCanceled,
	}
	for i, a := range errs {
		for j, b := range errs {
			if i != j && errors.Is(a, b) {
				t.Errorf("error %v should not match %v via errors.Is", a, b)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

// ---------------------------------------------------------------------------
// ValidateSorterConfig — AmbiguityPolicy and RejectExitID (Iteration 04)
// ---------------------------------------------------------------------------

func TestValidateSorterConfig_ValidAmbiguityPolicyError(t *testing.T) {
	cfg := validSorterConfig()
	cfg.AmbiguityPolicy = AmbiguityPolicyError
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Errorf("expected no error for valid AmbiguityPolicy, got: %v", err)
	}
}

func TestValidateSorterConfig_ValidAmbiguityPolicyFirstWins(t *testing.T) {
	cfg := validSorterConfig()
	cfg.AmbiguityPolicy = AmbiguityPolicyFirstWins
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Errorf("expected no error for first_wins AmbiguityPolicy, got: %v", err)
	}
}

func TestValidateSorterConfig_UnknownAmbiguityPolicy(t *testing.T) {
	cfg := validSorterConfig()
	cfg.AmbiguityPolicy = "unknown_policy"
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown AmbiguityPolicy, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestValidateSorterConfig_EmptyAmbiguityPolicyIsValid(t *testing.T) {
	cfg := validSorterConfig()
	cfg.AmbiguityPolicy = "" // empty = default error policy
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Errorf("expected no error for empty AmbiguityPolicy (default), got: %v", err)
	}
}

func TestValidateSorterConfig_RejectExitIDValid(t *testing.T) {
	cfg := validSorterConfig()
	cfg.RejectExitID = "exit-1"
	if err := ValidateSorterConfig(cfg); err != nil {
		t.Errorf("expected no error for valid RejectExitID, got: %v", err)
	}
}

func TestValidateSorterConfig_RejectExitIDNotFound(t *testing.T) {
	cfg := validSorterConfig()
	cfg.RejectExitID = "exit-nonexistent"
	err := ValidateSorterConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown RejectExitID, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}
