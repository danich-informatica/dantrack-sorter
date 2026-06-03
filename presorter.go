package sorter

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ResolvePresorter decide a qué parque del presorter debe enviarse la caja descrita en req.
//
// Semántica de retorno:
//   - error Go solo para: contexto cancelado, request inválido, presorter no configurado.
//   - ActionReject NO produce error Go; es una decisión válida con trazabilidad completa.
//
// Flujo de resolución:
//  1. Verificar contexto cancelado.
//  2. Validar request.
//  3. Verificar que presorter está configurado.
//  4. Resolver evalTime.
//  5. Construir índice de ParkStates.
//  6. Filtrar parks disponibles.
//  7. Aplicar preferencia de error control si la caja lo requiere.
//  8. Seleccionar park según BalanceStrategy.
//  9. Fallback a DefaultParkID si no hay candidatos.
//  10. ActionReject si ningún park es válido.
func (e *Engine) ResolvePresorter(ctx context.Context, req PresorterRequest) (PresorterDecision, error) {
	// 1. Contexto cancelado.
	if ctx.Err() != nil {
		return PresorterDecision{}, fmt.Errorf("%w", ErrContextCanceled)
	}

	// 2. Validar request.
	if err := ValidatePresorterRequest(req); err != nil {
		return PresorterDecision{}, err
	}

	// 3. Presorter debe estar configurado.
	if e.presorterCfg == nil {
		return PresorterDecision{}, fmt.Errorf("%w: presorter not configured in engine", ErrInvalidConfig)
	}

	// 4. Resolver evalTime una sola vez; cero → now.
	evalTime := req.EvalTime
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	// 5. Índice de estados de parks para lookup O(1).
	stateIdx := buildParkStateIndex(req.ParkStates)

	// 6. Traza base con IDs del orquestador.
	trace := DecisionTrace{
		TraceID:       req.TraceID,
		CorrelationID: req.CorrelationID,
	}

	// 6b. Fixed error park: if box requires error control and ErrorParkID is configured,
	// route exclusively to that park. No fallback to other parks.
	if boxRequiresErrorControl(req.Box) && e.presorterCfg.ErrorParkID != "" {
		return e.resolveFixedErrorPark(trace, stateIdx, evalTime)
	}

	// 7. Evaluar disponibilidad de cada park configurado.
	var candidates []parkCandidate
	var allEvals []CandidateEvaluation

	for i, cfg := range e.presorterCfg.Parks {
		state := stateIdx[cfg.ParkID] // puede ser nil
		if isParkAvailable(cfg, state) {
			candidates = append(candidates, parkCandidate{cfg: cfg, index: i})
			allEvals = append(allEvals, CandidateEvaluation{
				TargetType: TargetTypePark,
				TargetID:   cfg.ParkID,
				Eligible:   true,
			})
		} else {
			reason := parkRejectedReason(cfg, state)
			allEvals = append(allEvals, CandidateEvaluation{
				TargetType:     TargetTypePark,
				TargetID:       cfg.ParkID,
				Eligible:       false,
				RejectedReason: reason,
			})
		}
	}

	// 8. Control de errores: si la caja requiere error control, preferir parks con HasErrorControl.
	errorControlRequired := boxRequiresErrorControl(req.Box)
	errorControlAvailable := false
	fallbackUsed := false

	if errorControlRequired && len(candidates) > 0 {
		var ecCandidates []parkCandidate
		for _, c := range candidates {
			if c.cfg.HasErrorControl {
				ecCandidates = append(ecCandidates, c)
			}
		}
		if len(ecCandidates) > 0 {
			candidates = ecCandidates
			errorControlAvailable = true
		} else {
			// No hay park con error control disponible; usar todos los candidatos como fallback.
			fallbackUsed = true
		}
	}

	// 9. Si hay candidatos, elegir según BalanceStrategy.
	if len(candidates) > 0 {
		chosen, rule := e.selectByStrategy(candidates, stateIdx)

		// Si la caja requería error control y se usó park con error control, aplicar regla preferida.
		if errorControlRequired && errorControlAvailable {
			rule = RulePresorterErrorControlPreferred
		}

		// Marcar candidato elegido en evaluaciones.
		for i := range allEvals {
			if allEvals[i].TargetID == chosen.cfg.ParkID && allEvals[i].Eligible {
				allEvals[i].Rule = rule
				break
			}
		}

		trace.RuleApplied = rule
		trace.Reason = presorterBalanceReason(rule, chosen.cfg.ParkID, stateIdx)
		trace.CandidateEvaluations = allEvals

		return PresorterDecision{
			ParkID:                chosen.cfg.ParkID,
			Action:                ActionRoute,
			FallbackUsed:          fallbackUsed,
			ErrorControlRequired:  errorControlRequired,
			ErrorControlAvailable: errorControlAvailable,
			BalanceStrategy:       e.presorterCfg.BalanceStrategy,
			Trace:                 trace,
			EvalTime:              evalTime,
		}, nil
	}

	// 10. Sin candidatos: intentar DefaultParkID como fallback.
	if e.presorterCfg.DefaultParkID != "" {
		defaultState := stateIdx[e.presorterCfg.DefaultParkID]
		// Buscar la config del default park.
		var defaultCfg *ParkConfig
		for i := range e.presorterCfg.Parks {
			if e.presorterCfg.Parks[i].ParkID == e.presorterCfg.DefaultParkID {
				defaultCfg = &e.presorterCfg.Parks[i]
				break
			}
		}
		if defaultCfg != nil && isParkAvailable(*defaultCfg, defaultState) {
			allEvals = append(allEvals, CandidateEvaluation{
				TargetType: TargetTypePark,
				TargetID:   defaultCfg.ParkID,
				Eligible:   true,
				Rule:       RulePresorterFallbackDefaultPark,
			})
			trace.RuleApplied = RulePresorterFallbackDefaultPark
			trace.Reason = "all candidates unavailable; routed to default park"
			trace.CandidateEvaluations = allEvals

			return PresorterDecision{
				ParkID:                defaultCfg.ParkID,
				Action:                ActionRoute,
				FallbackUsed:          true,
				ErrorControlRequired:  errorControlRequired,
				ErrorControlAvailable: false,
				BalanceStrategy:       e.presorterCfg.BalanceStrategy,
				Trace:                 trace,
				EvalTime:              evalTime,
			}, nil
		}
	}

	// 11. Sin park válido → ActionReject.
	trace.RuleApplied = RulePresorterRejectNoAvailablePark
	trace.Reason = "no available park for routing"
	trace.DiagnosticMessage = "all parks are blocked, full, in maintenance, have errors, or are disabled"
	trace.CandidateEvaluations = allEvals

	return PresorterDecision{
		Action:                ActionReject,
		FallbackUsed:          false,
		ErrorControlRequired:  errorControlRequired,
		ErrorControlAvailable: false,
		BalanceStrategy:       e.presorterCfg.BalanceStrategy,
		Trace:                 trace,
		EvalTime:              evalTime,
	}, nil
}

// resolveFixedErrorPark handles error/no-read boxes when ErrorParkID is configured.
// Routes exclusively to ErrorParkID. If unavailable, returns ActionPass (no fallback).
func (e *Engine) resolveFixedErrorPark(
	trace DecisionTrace,
	stateIdx map[string]*ParkState,
	evalTime time.Time,
) (PresorterDecision, error) {
	errorParkID := e.presorterCfg.ErrorParkID

	// Find park config.
	var errorCfg *ParkConfig
	for i := range e.presorterCfg.Parks {
		if e.presorterCfg.Parks[i].ParkID == errorParkID {
			errorCfg = &e.presorterCfg.Parks[i]
			break
		}
	}

	// Park not found in config (should not happen after validation, but defensive).
	if errorCfg == nil {
		trace.RuleApplied = RulePresorterPassErrorParkUnavailable
		trace.Reason = "fixed error park unavailable; pass without diversion"
		trace.DiagnosticMessage = "ErrorParkID not found in park config"
		trace.CandidateEvaluations = []CandidateEvaluation{{
			TargetType:     TargetTypePark,
			TargetID:       errorParkID,
			Eligible:       false,
			RejectedReason: "park config not found",
		}}

		return PresorterDecision{
			Action:                ActionPass,
			FallbackUsed:          false,
			ErrorControlRequired:  true,
			ErrorControlAvailable: false,
			BalanceStrategy:       e.presorterCfg.BalanceStrategy,
			Trace:                 trace,
			EvalTime:              evalTime,
		}, nil
	}

	state := stateIdx[errorParkID]

	if isParkAvailable(*errorCfg, state) {
		trace.RuleApplied = RulePresorterErrorFixedPark
		trace.Reason = "error/no-read routed to fixed error park"
		trace.CandidateEvaluations = []CandidateEvaluation{{
			TargetType: TargetTypePark,
			TargetID:   errorParkID,
			Eligible:   true,
			Rule:       RulePresorterErrorFixedPark,
		}}

		return PresorterDecision{
			ParkID:                errorParkID,
			Action:                ActionRoute,
			FallbackUsed:          false,
			ErrorControlRequired:  true,
			ErrorControlAvailable: errorCfg.HasErrorControl,
			BalanceStrategy:       e.presorterCfg.BalanceStrategy,
			Trace:                 trace,
			EvalTime:              evalTime,
		}, nil
	}

	// ErrorPark unavailable → ActionPass. NO fallback.
	reason := parkRejectedReason(*errorCfg, state)
	trace.RuleApplied = RulePresorterPassErrorParkUnavailable
	trace.Reason = "fixed error park unavailable; pass without diversion"
	trace.DiagnosticMessage = fmt.Sprintf("ErrorParkID %q is %s; no fallback for error boxes", errorParkID, reason)
	trace.CandidateEvaluations = []CandidateEvaluation{{
		TargetType:     TargetTypePark,
		TargetID:       errorParkID,
		Eligible:       false,
		RejectedReason: reason,
	}}

	return PresorterDecision{
		Action:                ActionPass,
		FallbackUsed:          false,
		ErrorControlRequired:  true,
		ErrorControlAvailable: false,
		BalanceStrategy:       e.presorterCfg.BalanceStrategy,
		Trace:                 trace,
		EvalTime:              evalTime,
	}, nil
}

// selectByStrategy elige un park según la BalanceStrategy configurada.
// Retorna el candidato elegido y la Rule correspondiente.
//
// Precondición: len(candidates) > 0.
func (e *Engine) selectByStrategy(candidates []parkCandidate, stateIdx map[string]*ParkState) (parkCandidate, Rule) {
	switch e.presorterCfg.BalanceStrategy {
	case BalanceRoundRobin:
		return selectRoundRobin(candidates, &e.rrCounter), RulePresorterRoundRobin
	case BalanceWeighted:
		return selectWeighted(candidates), RulePresorterWeighted
	default:
		// least_loaded es el default (incluye cadena vacía).
		// If any candidate has AccumulatedLoad > 0, use global balance.
		if hasAccumulatedLoad(candidates, stateIdx) {
			return selectGlobalBalanced(candidates, stateIdx), RulePresorterGlobalBalance
		}
		return selectLeastLoaded(candidates, stateIdx), RulePresorterLeastLoaded
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildParkStateIndex construye un mapa ParkID → *ParkState para lookup O(1).
func buildParkStateIndex(states []ParkState) map[string]*ParkState {
	idx := make(map[string]*ParkState, len(states))
	for i := range states {
		idx[states[i].ParkID] = &states[i]
	}
	return idx
}

// isParkAvailable devuelve true si el park puede recibir cajas.
//
// Política de estado ausente: si state es nil (no se proveyó estado), se asume
// disponible. Permite operar con resiliencia parcial cuando el sistema de estados
// no reporta todos los parks.
func isParkAvailable(cfg ParkConfig, state *ParkState) bool {
	if !cfg.Enabled {
		return false
	}
	if state == nil {
		// Sin estado → asumir disponible (política documentada).
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

// boxRequiresErrorControl determina si una caja requiere control de errores.
//
// Retorna true si:
//   - ReadStatus ∈ {error, no_read, no-read, noread} (case-insensitive, trimmed).
//   - QualityStatus ∈ {error, rejected, bad} (case-insensitive, trimmed).
//   - len(ErrorFlags) > 0.
func boxRequiresErrorControl(box BoxContext) bool {
	if len(box.ErrorFlags) > 0 {
		return true
	}

	rs := strings.ToLower(strings.TrimSpace(box.ReadStatus))
	switch rs {
	case "error", "no_read", "no-read", "noread":
		return true
	}

	qs := strings.ToLower(strings.TrimSpace(box.QualityStatus))
	switch qs {
	case "error", "rejected", "bad":
		return true
	}

	return false
}

// parkRejectedReason devuelve una razón legible por la que un park fue descartado.
func parkRejectedReason(cfg ParkConfig, state *ParkState) string {
	if !cfg.Enabled {
		return "park disabled"
	}
	if state == nil {
		return "unknown"
	}
	if !state.Available {
		return "park not available"
	}
	if state.Blocked {
		return "park blocked"
	}
	if state.Full {
		return "park full"
	}
	if state.HasError {
		return "park has error"
	}
	if state.Maintenance {
		return "park in maintenance"
	}
	return "unknown"
}

// presorterBalanceReason builds a Reason string for the presorter balance decision.
func presorterBalanceReason(rule Rule, parkID string, stateIdx map[string]*ParkState) string {
	switch rule {
	case RulePresorterGlobalBalance:
		accum := accumulatedLoadOf(parkID, stateIdx)
		current := currentLoadOf(parkID, stateIdx)
		return fmt.Sprintf("global balance: chose %s (accumulated=%d, current=%d)", parkID, accum, current)
	case RulePresorterLeastLoaded:
		current := currentLoadOf(parkID, stateIdx)
		return fmt.Sprintf("least loaded: chose %s (current=%d)", parkID, current)
	default:
		return "park selected by balance strategy"
	}
}
