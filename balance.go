package sorter

import "sync/atomic"

// parkCandidate agrupa un park configurado con su índice original en la config,
// permitiendo desempates estables por posición.
type parkCandidate struct {
	cfg   ParkConfig
	index int // posición en PresorterConfig.Parks
}

// selectLeastLoaded elige el park con menor CurrentLoad.
// Si no hay estado para un park, se asume CurrentLoad = 0.
// Desempata por posición en config (menor index primero) para estabilidad.
//
// Precondición: len(candidates) > 0.
func selectLeastLoaded(candidates []parkCandidate, stateIdx map[string]*ParkState) parkCandidate {
	best := candidates[0]
	bestLoad := currentLoadOf(best.cfg.ParkID, stateIdx)

	for _, c := range candidates[1:] {
		load := currentLoadOf(c.cfg.ParkID, stateIdx)
		if load < bestLoad || (load == bestLoad && c.index < best.index) {
			best = c
			bestLoad = load
		}
	}
	return best
}

// selectRoundRobin elige un park rotando entre candidatos usando un contador atómico.
// Es seguro para concurrencia sin mutex.
//
// Precondición: len(candidates) > 0.
func selectRoundRobin(candidates []parkCandidate, counter *uint64) parkCandidate {
	n := uint64(len(candidates))
	idx := atomic.AddUint64(counter, 1) - 1
	return candidates[idx%n]
}

// selectWeighted elige el park con mayor peso efectivo (Weight).
// Weight <= 0 se trata como 1.
// Desempata por posición en config (menor index primero) para estabilidad.
//
// Precondición: len(candidates) > 0.
func selectWeighted(candidates []parkCandidate) parkCandidate {
	best := candidates[0]
	bestWeight := effectiveWeight(best.cfg.Weight)

	for _, c := range candidates[1:] {
		w := effectiveWeight(c.cfg.Weight)
		if w > bestWeight || (w == bestWeight && c.index < best.index) {
			best = c
			bestWeight = w
		}
	}
	return best
}

// effectiveWeight normaliza Weight: <= 0 se trata como 1.
func effectiveWeight(w int) int {
	if w <= 0 {
		return 1
	}
	return w
}

// currentLoadOf retorna el CurrentLoad de un park desde el índice de estados.
// Si no hay estado para el park, retorna 0.
func currentLoadOf(parkID string, stateIdx map[string]*ParkState) int {
	if s := stateIdx[parkID]; s != nil {
		return s.CurrentLoad
	}
	return 0
}

// hasAccumulatedLoad returns true if at least one candidate has AccumulatedLoad > 0
// in the state index. This triggers global balance mode.
func hasAccumulatedLoad(candidates []parkCandidate, stateIdx map[string]*ParkState) bool {
	for _, c := range candidates {
		if s := stateIdx[c.cfg.ParkID]; s != nil && s.AccumulatedLoad > 0 {
			return true
		}
	}
	return false
}

// selectGlobalBalanced elige el park con menor AccumulatedLoad.
// Si hay empate en AccumulatedLoad, desempata por menor CurrentLoad.
// Si persiste empate, respeta orden de configuración (menor index primero).
// AccumulatedLoad == 0 se trata como valor real (no como "sin información").
//
// Precondición: len(candidates) > 0.
func selectGlobalBalanced(candidates []parkCandidate, stateIdx map[string]*ParkState) parkCandidate {
	best := candidates[0]
	bestAccum := accumulatedLoadOf(best.cfg.ParkID, stateIdx)
	bestCurrent := currentLoadOf(best.cfg.ParkID, stateIdx)

	for _, c := range candidates[1:] {
		accum := accumulatedLoadOf(c.cfg.ParkID, stateIdx)
		current := currentLoadOf(c.cfg.ParkID, stateIdx)

		if accum < bestAccum {
			best = c
			bestAccum = accum
			bestCurrent = current
		} else if accum == bestAccum {
			if current < bestCurrent {
				best = c
				bestAccum = accum
				bestCurrent = current
			} else if current == bestCurrent && c.index < best.index {
				best = c
				bestAccum = accum
				bestCurrent = current
			}
		}
	}
	return best
}

// accumulatedLoadOf retorna el AccumulatedLoad de un park desde el índice de estados.
// Si no hay estado para el park, retorna 0.
func accumulatedLoadOf(parkID string, stateIdx map[string]*ParkState) int {
	if s := stateIdx[parkID]; s != nil {
		return s.AccumulatedLoad
	}
	return 0
}
