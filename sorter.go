package sorter

import (
	"context"
	"fmt"
	"time"
)

// ResolveSorter decide a qué salida del sorter debe enviarse la caja descrita en req.
//
// Semántica de retorno:
//   - error Go solo para: contexto cancelado, request inválido, sorter no configurado, ambigüedad irresoluble.
//   - ActionReject NO produce error Go; es una decisión válida con trazabilidad completa.
//
// Flujo de resolución (por prioridad decreciente):
//  1. Evalúa todas las asignaciones del request.
//  2. Si no hay match → resolveNoMatch.
//  3. Si hay ambigüedad en top-priority → checkAmbiguity (error o first_wins según policy).
//  4. Itera matched en orden de prioridad; usa el primer exit disponible.
//  5. Si todos los exits matched están no disponibles → intenta DefaultExitID.
//  6. Si tampoco → ActionReject con RuleSorterRejectNoAvailableExit.
func (e *Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error) {
	// 1. Contexto cancelado.
	if ctx.Err() != nil {
		return SorterDecision{}, fmt.Errorf("%w", ErrContextCanceled)
	}

	// 2. Validar request.
	if err := ValidateSorterRequest(req); err != nil {
		return SorterDecision{}, err
	}

	// 3. Sorter debe estar configurado.
	if e.sorterCfg == nil {
		return SorterDecision{}, fmt.Errorf("%w: sorter not configured in engine", ErrInvalidConfig)
	}

	// 4. Resolver evalTime una sola vez; cero → now.
	evalTime := req.EvalTime
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	// 5. Índice de estados de exits para lookup O(1).
	exitStateIdx := buildExitStateIndex(req.ExitStates)

	// 6. Traza base con IDs del orquestador.
	trace := DecisionTrace{
		TraceID:       req.TraceID,
		CorrelationID: req.CorrelationID,
	}

	// 7. Evaluar asignaciones (incluye matches y no-matches para trazabilidad completa).
	allMatches := EvaluateAssignments(req.Box, req.Assignments, evalTime)

	// 8. Separar asignaciones que hicieron match.
	var matched []AssignmentMatch
	for _, m := range allMatches {
		if m.Matched {
			matched = append(matched, m)
		}
	}

	// 9. Sin matches de asignación.
	if len(matched) == 0 {
		return e.resolveNoMatch(trace, exitStateIdx, allMatches, evalTime)
	}

	// 10. Verificar ambigüedad: top-priority matches que apuntan a diferentes TargetIDs.
	topPriority := matched[0].Assignment.Priority
	var topMatches []AssignmentMatch
	for _, m := range matched {
		if m.Assignment.Priority == topPriority {
			topMatches = append(topMatches, m)
		}
	}
	if len(topMatches) > 1 {
		if err := e.checkAmbiguity(topMatches); err != nil {
			return SorterDecision{}, err
		}
	}

	// 11. Construir candidatos base desde todas las evaluaciones de asignaciones.
	candidates := assignmentMatchesToCandidates(allMatches)

	// 12. Intentar cada match en orden de prioridad descendente.
	// Strategy fork: first-available (legacy) vs least_loaded (balanced).
	if e.sorterCfg.BalanceStrategy == BalanceLeastLoaded {
		return e.resolveBalancedMultiTarget(matched, candidates, trace, exitStateIdx, evalTime)
	}

	// Default: first-available. Iterate all effective target IDs for each matched assignment.
	for _, m := range matched {
		targetIDs := assignmentTargetIDs(m.Assignment)

		for _, exitID := range targetIDs {
			if _, inConfig := e.exitIndex[exitID]; !inConfig {
				candidates = append(candidates, CandidateEvaluation{
					TargetType:     TargetTypeExit,
					TargetID:       exitID,
					Eligible:       false,
					RejectedReason: "exit not in sorter config",
					AssignmentID:   m.Assignment.ID,
				})
				continue
			}

			if isExitAvailable(exitID, e.exitIndex, exitStateIdx) {
				candidates = append(candidates, CandidateEvaluation{
					TargetType:   TargetTypeExit,
					TargetID:     exitID,
					Eligible:     true,
					Rule:         RuleSorterAssignmentMatched,
					AssignmentID: m.Assignment.ID,
				})
				trace.RuleApplied = RuleSorterAssignmentMatched
				trace.Reason = "assignment matched and exit available"
				trace.CandidateEvaluations = candidates

				return SorterDecision{
					SorterID:     e.sorterCfg.SorterID,
					ExitID:       exitID,
					Action:       ActionRoute,
					AssignmentID: m.Assignment.ID,
					FallbackUsed: false,
					Rejected:     false,
					Trace:        trace,
					EvalTime:     evalTime,
				}, nil
			}

			// Exit exists but not available; try next target.
			candidates = append(candidates, CandidateEvaluation{
				TargetType:     TargetTypeExit,
				TargetID:       exitID,
				Eligible:       false,
				RejectedReason: "exit unavailable",
				AssignmentID:   m.Assignment.ID,
			})
		}
	}

	// 13. Todos los exits de asignaciones no disponibles; intentar DefaultExitID como fallback.
	if e.sorterCfg.DefaultExitID != "" && isExitAvailable(e.sorterCfg.DefaultExitID, e.exitIndex, exitStateIdx) {
		candidates = append(candidates, CandidateEvaluation{
			TargetType: TargetTypeExit,
			TargetID:   e.sorterCfg.DefaultExitID,
			Eligible:   true,
			Rule:       RuleSorterFallbackDefaultExit,
		})
		trace.RuleApplied = RuleSorterFallbackDefaultExit
		trace.Reason = "all matched exits unavailable; routed to default exit"
		trace.CandidateEvaluations = candidates

		return SorterDecision{
			SorterID:     e.sorterCfg.SorterID,
			ExitID:       e.sorterCfg.DefaultExitID,
			Action:       ActionRoute,
			FallbackUsed: true,
			Rejected:     false,
			Trace:        trace,
			EvalTime:     evalTime,
		}, nil
	}

	// 14. No hay exit disponible en absoluto.
	trace.RuleApplied = RuleSorterRejectNoAvailableExit
	trace.Reason = "no available exit for any matched assignment"
	trace.DiagnosticMessage = "all exits are blocked, full, in maintenance, or not configured"
	trace.CandidateEvaluations = candidates

	return SorterDecision{
		SorterID:     e.sorterCfg.SorterID,
		Action:       ActionReject,
		FallbackUsed: false,
		Rejected:     true,
		Trace:        trace,
		EvalTime:     evalTime,
	}, nil
}

// resolveNoMatch maneja el caso en que ninguna asignación hizo match con la caja.
// Orden de preferencia: RejectExitID → DefaultExitID → ActionReject sin exit.
func (e *Engine) resolveNoMatch(
	trace DecisionTrace,
	exitStateIdx map[string]SorterExitState,
	allMatches []AssignmentMatch,
	evalTime time.Time,
) (SorterDecision, error) {
	candidates := assignmentMatchesToCandidates(allMatches)

	// Opción 1: RejectExitID configurado y disponible.
	if e.sorterCfg.RejectExitID != "" && isExitAvailable(e.sorterCfg.RejectExitID, e.exitIndex, exitStateIdx) {
		candidates = append(candidates, CandidateEvaluation{
			TargetType: TargetTypeExit,
			TargetID:   e.sorterCfg.RejectExitID,
			Eligible:   true,
			Rule:       RuleSorterRejectNoMatch,
		})
		trace.RuleApplied = RuleSorterRejectNoMatch
		trace.Reason = "no assignment matched; routed to reject exit"
		trace.CandidateEvaluations = candidates

		return SorterDecision{
			SorterID:     e.sorterCfg.SorterID,
			ExitID:       e.sorterCfg.RejectExitID,
			Action:       ActionReject,
			FallbackUsed: false,
			Rejected:     true,
			Trace:        trace,
			EvalTime:     evalTime,
		}, nil
	}

	// Opción 2: DefaultExitID configurado y disponible (ruta de fallback sin match).
	if e.sorterCfg.DefaultExitID != "" && isExitAvailable(e.sorterCfg.DefaultExitID, e.exitIndex, exitStateIdx) {
		candidates = append(candidates, CandidateEvaluation{
			TargetType: TargetTypeExit,
			TargetID:   e.sorterCfg.DefaultExitID,
			Eligible:   true,
			Rule:       RuleSorterRejectNoMatch,
		})
		trace.RuleApplied = RuleSorterRejectNoMatch
		trace.Reason = "no assignment matched; routed to default exit"
		trace.DiagnosticMessage = "no matching assignment found; default exit used as fallback"
		trace.CandidateEvaluations = candidates

		return SorterDecision{
			SorterID:     e.sorterCfg.SorterID,
			ExitID:       e.sorterCfg.DefaultExitID,
			Action:       ActionRoute,
			FallbackUsed: true,
			Rejected:     false,
			Trace:        trace,
			EvalTime:     evalTime,
		}, nil
	}

	// Opción 3: No hay exit de rechazo ni default disponible.
	trace.RuleApplied = RuleSorterRejectNoAvailableExit
	trace.Reason = "no assignment matched and no reject or default exit available"
	trace.CandidateEvaluations = candidates

	return SorterDecision{
		SorterID:     e.sorterCfg.SorterID,
		Action:       ActionReject,
		FallbackUsed: false,
		Rejected:     true,
		Trace:        trace,
		EvalTime:     evalTime,
	}, nil
}

// checkAmbiguity verifies if topMatches (equal priority) point to different targets.
// With multi-target, compares the first effective target of each assignment.
// If there is real ambiguity, applies the AmbiguityPolicy:
//   - empty or "error" → returns ErrAmbiguousAssignment
//   - "first_wins" → no error (caller uses first match)
func (e *Engine) checkAmbiguity(topMatches []AssignmentMatch) error {
	firstTargets := assignmentTargetIDs(topMatches[0].Assignment)
	firstTarget := ""
	if len(firstTargets) > 0 {
		firstTarget = firstTargets[0]
	}

	ambiguous := false
	for _, m := range topMatches[1:] {
		targets := assignmentTargetIDs(m.Assignment)
		t := ""
		if len(targets) > 0 {
			t = targets[0]
		}
		if t != firstTarget {
			ambiguous = true
			break
		}
	}
	if !ambiguous {
		return nil
	}

	policy := e.sorterCfg.AmbiguityPolicy
	if policy == "" || policy == AmbiguityPolicyError {
		return fmt.Errorf(
			"%w: %d assignments with equal priority %d point to different exits",
			ErrAmbiguousAssignment,
			len(topMatches),
			topMatches[0].Assignment.Priority,
		)
	}
	// AmbiguityPolicyFirstWins: no error; el caller usará el primer match (topMatches[0]).
	return nil
}

// ---------------------------------------------------------------------------
// resolveBalancedMultiTarget — least_loaded strategy
// ---------------------------------------------------------------------------

// resolveBalancedMultiTarget implements BalanceLeastLoaded for multi-target assignments.
// For each matched assignment (in priority order), it collects all effective targets and:
//   - If at least one target is available: picks the one with lowest CurrentLoad (tie-break by TargetIDs order).
//   - If all targets are Full: returns ActionPass with RuleSorterPassAllTargetsFull.
//   - If all targets are unavailable (blocked/error/maintenance/not in config) but not all Full:
//     falls through to fallback/default/reject.
func (e *Engine) resolveBalancedMultiTarget(
	matched []AssignmentMatch,
	candidates []CandidateEvaluation,
	trace DecisionTrace,
	exitStateIdx map[string]SorterExitState,
	evalTime time.Time,
) (SorterDecision, error) {
	for _, m := range matched {
		targetIDs := assignmentTargetIDs(m.Assignment)

		var available []string
		allFull := true
		anyFull := false

		for _, exitID := range targetIDs {
			if _, inConfig := e.exitIndex[exitID]; !inConfig {
				allFull = false
				candidates = append(candidates, CandidateEvaluation{
					TargetType:     TargetTypeExit,
					TargetID:       exitID,
					Eligible:       false,
					RejectedReason: "exit not in sorter config",
					AssignmentID:   m.Assignment.ID,
				})
				continue
			}

			state, hasState := exitStateIdx[exitID]
			if hasState && state.Full {
				anyFull = true
				candidates = append(candidates, CandidateEvaluation{
					TargetType:     TargetTypeExit,
					TargetID:       exitID,
					Eligible:       false,
					RejectedReason: fmt.Sprintf("exit full (load=%d)", state.CurrentLoad),
					AssignmentID:   m.Assignment.ID,
				})
				continue
			}

			if isExitAvailable(exitID, e.exitIndex, exitStateIdx) {
				allFull = false
				load := 0
				if hasState {
					load = state.CurrentLoad
				}
				available = append(available, exitID)
				candidates = append(candidates, CandidateEvaluation{
					TargetType:     TargetTypeExit,
					TargetID:       exitID,
					Eligible:       true,
					RejectedReason: fmt.Sprintf("load=%d", load),
					Rule:           RuleSorterBalancedMultiTarget,
					AssignmentID:   m.Assignment.ID,
				})
			} else {
				allFull = false
				reason := "exit unavailable"
				if hasState {
					reason = exitUnavailableReason(state)
				}
				candidates = append(candidates, CandidateEvaluation{
					TargetType:     TargetTypeExit,
					TargetID:       exitID,
					Eligible:       false,
					RejectedReason: reason,
					AssignmentID:   m.Assignment.ID,
				})
			}
		}

		// Case: at least one available → pick least loaded.
		if len(available) > 0 {
			chosen := pickLeastLoaded(available, exitStateIdx)
			chosenLoad := 0
			if st, ok := exitStateIdx[chosen]; ok {
				chosenLoad = st.CurrentLoad
			}

			trace.RuleApplied = RuleSorterBalancedMultiTarget
			trace.Reason = fmt.Sprintf("balanced multi-target: chose %s (load=%d)", chosen, chosenLoad)
			trace.CandidateEvaluations = candidates

			return SorterDecision{
				SorterID:     e.sorterCfg.SorterID,
				ExitID:       chosen,
				Action:       ActionRoute,
				AssignmentID: m.Assignment.ID,
				FallbackUsed: false,
				Rejected:     false,
				Trace:        trace,
				EvalTime:     evalTime,
			}, nil
		}

		// Case: all effective targets in config and all are Full → ActionPass.
		if allFull && anyFull {
			trace.RuleApplied = RuleSorterPassAllTargetsFull
			trace.Reason = "all targets full; no capacity available"
			trace.DiagnosticMessage = "ActionPass: logical destination exists but all exits are at capacity"
			trace.CandidateEvaluations = candidates

			return SorterDecision{
				SorterID:     e.sorterCfg.SorterID,
				Action:       ActionPass,
				AssignmentID: m.Assignment.ID,
				FallbackUsed: false,
				Rejected:     false,
				Trace:        trace,
				EvalTime:     evalTime,
			}, nil
		}

		// Mixed: some full, some blocked/unavailable/not-in-config, none available.
		// If the only reason none are available is Full (all configured targets are full),
		// produce ActionPass. Otherwise fall through to fallback.
		if anyFull && allConfiguredTargetsFull(targetIDs, e.exitIndex, exitStateIdx) {
			trace.RuleApplied = RuleSorterPassAllTargetsFull
			trace.Reason = "all configured targets full; unconfigured targets ignored"
			trace.DiagnosticMessage = "ActionPass: configured targets at capacity; unconfigured targets not considered operational"
			trace.CandidateEvaluations = candidates

			return SorterDecision{
				SorterID:     e.sorterCfg.SorterID,
				Action:       ActionPass,
				AssignmentID: m.Assignment.ID,
				FallbackUsed: false,
				Rejected:     false,
				Trace:        trace,
				EvalTime:     evalTime,
			}, nil
		}

		// All targets unavailable for non-capacity reasons; continue to next match or fallback.
	}

	// All matched assignments exhausted; try DefaultExitID as fallback.
	if e.sorterCfg.DefaultExitID != "" && isExitAvailable(e.sorterCfg.DefaultExitID, e.exitIndex, exitStateIdx) {
		candidates = append(candidates, CandidateEvaluation{
			TargetType: TargetTypeExit,
			TargetID:   e.sorterCfg.DefaultExitID,
			Eligible:   true,
			Rule:       RuleSorterFallbackDefaultExit,
		})
		trace.RuleApplied = RuleSorterFallbackDefaultExit
		trace.Reason = "all matched exits unavailable; routed to default exit"
		trace.CandidateEvaluations = candidates

		return SorterDecision{
			SorterID:     e.sorterCfg.SorterID,
			ExitID:       e.sorterCfg.DefaultExitID,
			Action:       ActionRoute,
			FallbackUsed: true,
			Rejected:     false,
			Trace:        trace,
			EvalTime:     evalTime,
		}, nil
	}

	// No exit available at all.
	trace.RuleApplied = RuleSorterRejectNoAvailableExit
	trace.Reason = "no available exit for any matched assignment"
	trace.DiagnosticMessage = "all exits are blocked, full, in maintenance, or not configured"
	trace.CandidateEvaluations = candidates

	return SorterDecision{
		SorterID:     e.sorterCfg.SorterID,
		Action:       ActionReject,
		FallbackUsed: false,
		Rejected:     true,
		Trace:        trace,
		EvalTime:     evalTime,
	}, nil
}

// pickLeastLoaded selects the exit with the lowest CurrentLoad from available exits.
// Tie-break: first in the available slice (preserves TargetIDs order).
// If no state exists for an exit, CurrentLoad is treated as 0.
func pickLeastLoaded(available []string, stateIdx map[string]SorterExitState) string {
	best := available[0]
	bestLoad := exitLoad(best, stateIdx)

	for _, id := range available[1:] {
		load := exitLoad(id, stateIdx)
		if load < bestLoad {
			best = id
			bestLoad = load
		}
	}
	return best
}

// exitLoad returns the CurrentLoad of an exit. Returns 0 if no state is present.
func exitLoad(exitID string, stateIdx map[string]SorterExitState) int {
	if st, ok := stateIdx[exitID]; ok {
		return st.CurrentLoad
	}
	return 0
}

// allConfiguredTargetsFull returns true if every target that exists in the exit config is Full.
// Targets not in config are ignored. Returns false if no target is in config.
func allConfiguredTargetsFull(targetIDs []string, exitIndex map[string]SorterExit, stateIdx map[string]SorterExitState) bool {
	count := 0
	for _, id := range targetIDs {
		if _, inConfig := exitIndex[id]; !inConfig {
			continue
		}
		count++
		st, hasState := stateIdx[id]
		if !hasState || !st.Full {
			return false
		}
	}
	return count > 0
}

// exitUnavailableReason returns a human-readable reason for an unavailable exit.
func exitUnavailableReason(st SorterExitState) string {
	switch {
	case st.Blocked:
		return "exit blocked"
	case st.HasError:
		return "exit has error"
	case st.Maintenance:
		return "exit in maintenance"
	case !st.Available:
		return "exit not available"
	default:
		return "exit unavailable"
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildExitStateIndex construye un mapa ExitID → SorterExitState para lookup O(1).
func buildExitStateIndex(states []SorterExitState) map[string]SorterExitState {
	idx := make(map[string]SorterExitState, len(states))
	for _, s := range states {
		idx[s.ExitID] = s
	}
	return idx
}

// isExitAvailable devuelve true si el exit existe en el índice de configuración,
// está habilitado, y su estado operativo (si se proporcionó) indica que puede recibir cajas.
//
// Política de estado ausente: si no se proveyó estado para el exit, se asume disponible.
// Esto permite que el sorter siga operando cuando el sistema de estados no reporta un exit.
func isExitAvailable(exitID string, exitIndex map[string]SorterExit, stateIndex map[string]SorterExitState) bool {
	cfg, ok := exitIndex[exitID]
	if !ok || !cfg.Enabled {
		return false
	}

	state, ok := stateIndex[exitID]
	if !ok {
		// Política documentada: sin estado = asumir disponible.
		return true
	}

	if !state.Available {
		return false
	}
	if state.Blocked || state.Full || state.HasError || state.Maintenance {
		return false
	}
	return true
}

// assignmentMatchesToCandidates convierte los resultados de EvaluateAssignments
// en CandidateEvaluations para incluirlos en la traza de la decisión.
func assignmentMatchesToCandidates(matches []AssignmentMatch) []CandidateEvaluation {
	result := make([]CandidateEvaluation, 0, len(matches))
	for _, m := range matches {
		ce := CandidateEvaluation{
			TargetType:   m.Assignment.TargetType,
			TargetID:     m.Assignment.TargetID,
			Eligible:     m.Matched,
			AssignmentID: m.Assignment.ID,
			Conditions:   m.ConditionResults,
		}
		if !m.Matched {
			ce.RejectedReason = m.Reason
		}
		result = append(result, ce)
	}
	return result
}
