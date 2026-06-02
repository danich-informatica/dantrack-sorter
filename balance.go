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
