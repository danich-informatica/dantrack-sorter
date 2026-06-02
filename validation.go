package sorter

import (
	"fmt"
	"slices"
)

// validBalanceStrategies contiene las estrategias de balanceo aceptadas.
var validBalanceStrategies = []BalanceStrategy{
	BalanceLeastLoaded,
	BalanceRoundRobin,
	BalanceWeighted,
}

// validMatchOperators contiene los operadores de condición aceptados.
var validMatchOperators = []MatchOperator{
	OpEquals,
	OpNotEquals,
	OpIn,
	OpNotIn,
	OpExists,
	OpNotExists,
}

// validTargetTypes contiene los tipos de destino aceptados.
var validTargetTypes = []TargetType{
	TargetTypeExit,
	TargetTypePark,
}

// validAmbiguityPolicies contiene las políticas de ambigüedad aceptadas.
var validAmbiguityPolicies = []AmbiguityPolicy{
	AmbiguityPolicyError,
	AmbiguityPolicyFirstWins,
}

// ---------------------------------------------------------------------------
// ValidateConfig
// ---------------------------------------------------------------------------

// ValidateConfig valida la configuración completa del engine.
// Devuelve ErrInvalidConfig si la config no permite operar.
func ValidateConfig(cfg EngineConfig) error {
	if cfg.Presorter == nil && cfg.Sorter == nil {
		return fmt.Errorf("%w: at least one of Presorter or Sorter must be configured", ErrInvalidConfig)
	}
	if cfg.Presorter != nil {
		if err := ValidatePresorterConfig(*cfg.Presorter); err != nil {
			return err
		}
	}
	if cfg.Sorter != nil {
		if err := ValidateSorterConfig(*cfg.Sorter); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ValidatePresorterConfig
// ---------------------------------------------------------------------------

// ValidatePresorterConfig valida la configuración del presorter.
func ValidatePresorterConfig(cfg PresorterConfig) error {
	if len(cfg.Parks) == 0 {
		return fmt.Errorf("%w: presorter requires at least one park", ErrInvalidConfig)
	}

	seen := make(map[string]struct{}, len(cfg.Parks))
	for i, p := range cfg.Parks {
		if p.ParkID == "" {
			return fmt.Errorf("%w: park at index %d has empty ParkID", ErrMissingRequiredField, i)
		}
		if _, dup := seen[p.ParkID]; dup {
			return fmt.Errorf("%w: duplicate ParkID %q in presorter config", ErrInvalidConfig, p.ParkID)
		}
		seen[p.ParkID] = struct{}{}
	}

	if cfg.BalanceStrategy != "" && !slices.Contains(validBalanceStrategies, cfg.BalanceStrategy) {
		return fmt.Errorf("%w: unknown BalanceStrategy %q", ErrInvalidConfig, cfg.BalanceStrategy)
	}

	if cfg.DefaultParkID != "" {
		if _, ok := seen[cfg.DefaultParkID]; !ok {
			return fmt.Errorf("%w: DefaultParkID %q does not reference an existing park", ErrInvalidConfig, cfg.DefaultParkID)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// ValidateSorterConfig
// ---------------------------------------------------------------------------

// ValidateSorterConfig valida la configuración del sorter.
func ValidateSorterConfig(cfg SorterConfig) error {
	if cfg.SorterID == "" {
		return fmt.Errorf("%w: SorterID is required", ErrMissingRequiredField)
	}
	if len(cfg.Exits) == 0 {
		return fmt.Errorf("%w: sorter %q requires at least one exit", ErrInvalidConfig, cfg.SorterID)
	}

	seen := make(map[string]struct{}, len(cfg.Exits))
	for i, e := range cfg.Exits {
		if e.ExitID == "" {
			return fmt.Errorf("%w: exit at index %d in sorter %q has empty ExitID", ErrMissingRequiredField, i, cfg.SorterID)
		}
		if _, dup := seen[e.ExitID]; dup {
			return fmt.Errorf("%w: duplicate ExitID %q in sorter %q", ErrInvalidConfig, e.ExitID, cfg.SorterID)
		}
		seen[e.ExitID] = struct{}{}
	}

	if cfg.DefaultExitID != "" {
		if _, ok := seen[cfg.DefaultExitID]; !ok {
			return fmt.Errorf("%w: DefaultExitID %q does not reference an existing exit", ErrInvalidConfig, cfg.DefaultExitID)
		}
	}

	if cfg.RejectExitID != "" {
		if _, ok := seen[cfg.RejectExitID]; !ok {
			return fmt.Errorf("%w: RejectExitID %q does not reference an existing exit", ErrInvalidConfig, cfg.RejectExitID)
		}
	}

	if cfg.AmbiguityPolicy != "" && !slices.Contains(validAmbiguityPolicies, cfg.AmbiguityPolicy) {
		return fmt.Errorf("%w: unknown AmbiguityPolicy %q", ErrInvalidConfig, cfg.AmbiguityPolicy)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Validación de BoxContext
// ---------------------------------------------------------------------------

// validateBoxContext valida los campos mínimos obligatorios de un BoxContext.
// BoxID y QR son los campos mínimos requeridos para identificar una caja.
func validateBoxContext(box BoxContext) error {
	if box.BoxID == "" && box.QR == "" {
		return fmt.Errorf("%w: BoxContext requires at least BoxID or QR", ErrMissingRequiredField)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ValidatePresorterRequest
// ---------------------------------------------------------------------------

// ValidatePresorterRequest valida un PresorterRequest antes de procesarlo.
func ValidatePresorterRequest(req PresorterRequest) error {
	if err := validateBoxContext(req.Box); err != nil {
		return fmt.Errorf("%w", err)
	}

	seen := make(map[string]struct{}, len(req.ParkStates))
	for i, ps := range req.ParkStates {
		if ps.ParkID == "" {
			return fmt.Errorf("%w: ParkState at index %d has empty ParkID", ErrMissingRequiredField, i)
		}
		if _, dup := seen[ps.ParkID]; dup {
			return fmt.Errorf("%w: duplicate ParkID %q in ParkStates", ErrInvalidRequest, ps.ParkID)
		}
		seen[ps.ParkID] = struct{}{}
	}

	return nil
}

// ---------------------------------------------------------------------------
// ValidateSorterRequest
// ---------------------------------------------------------------------------

// ValidateSorterRequest valida un SorterRequest antes de procesarlo.
func ValidateSorterRequest(req SorterRequest) error {
	if err := validateBoxContext(req.Box); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := validateAssignmentSet(req.Assignments); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(req.ExitStates))
	for i, es := range req.ExitStates {
		if es.ExitID == "" {
			return fmt.Errorf("%w: ExitState at index %d has empty ExitID", ErrMissingRequiredField, i)
		}
		if _, dup := seen[es.ExitID]; dup {
			return fmt.Errorf("%w: duplicate ExitID %q in ExitStates", ErrInvalidRequest, es.ExitID)
		}
		seen[es.ExitID] = struct{}{}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Validación de AssignmentSet
// ---------------------------------------------------------------------------

// validateAssignmentSet valida el conjunto de asignaciones si no viene vacío.
func validateAssignmentSet(set AssignmentSet) error {
	for i, a := range set.Assignments {
		if err := validateAssignment(a, i); err != nil {
			return err
		}
	}
	return nil
}

// validateAssignment valida una asignación individual.
func validateAssignment(a Assignment, idx int) error {
	if a.ID == "" {
		return fmt.Errorf("%w: assignment at index %d has empty ID", ErrMissingRequiredField, idx)
	}
	if a.Priority < 0 {
		return fmt.Errorf("%w: assignment %q has negative Priority", ErrInvalidRequest, a.ID)
	}
	if !slices.Contains(validTargetTypes, a.TargetType) {
		return fmt.Errorf("%w: assignment %q has unknown TargetType %q", ErrInvalidRequest, a.ID, a.TargetType)
	}
	if a.TargetID == "" {
		return fmt.Errorf("%w: assignment %q has empty TargetID", ErrMissingRequiredField, a.ID)
	}
	if !a.ValidFrom.IsZero() && !a.ValidTo.IsZero() && a.ValidTo.Before(a.ValidFrom) {
		return fmt.Errorf("%w: assignment %q has ValidTo before ValidFrom", ErrInvalidRequest, a.ID)
	}
	for j, c := range a.Conditions {
		if err := validateMatchCondition(c, a.ID, j); err != nil {
			return err
		}
	}
	return nil
}

// validateMatchCondition valida una condición de matching.
func validateMatchCondition(c MatchCondition, assignmentID string, idx int) error {
	if c.Field == "" {
		return fmt.Errorf("%w: condition at index %d in assignment %q has empty Field", ErrMissingRequiredField, idx, assignmentID)
	}
	if !slices.Contains(validMatchOperators, c.Operator) {
		return fmt.Errorf("%w: condition at index %d in assignment %q has unknown operator %q", ErrInvalidRequest, idx, assignmentID, c.Operator)
	}
	// exists/not_exists no requieren valores
	if c.Operator != OpExists && c.Operator != OpNotExists && len(c.Values) == 0 {
		return fmt.Errorf("%w: condition at index %d in assignment %q requires at least one value for operator %q", ErrInvalidRequest, idx, assignmentID, c.Operator)
	}
	return nil
}
