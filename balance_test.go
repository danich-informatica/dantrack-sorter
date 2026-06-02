package sorter

import (
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// selectLeastLoaded
// ---------------------------------------------------------------------------

func TestSelectLeastLoaded_ChoosesLowest(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
		{cfg: ParkConfig{ParkID: "p2"}, index: 1},
		{cfg: ParkConfig{ParkID: "p3"}, index: 2},
	}
	stateIdx := map[string]*ParkState{
		"p1": {ParkID: "p1", CurrentLoad: 10},
		"p2": {ParkID: "p2", CurrentLoad: 3},
		"p3": {ParkID: "p3", CurrentLoad: 7},
	}
	result := selectLeastLoaded(candidates, stateIdx)
	if result.cfg.ParkID != "p2" {
		t.Errorf("expected p2 (load=3), got %q", result.cfg.ParkID)
	}
}

func TestSelectLeastLoaded_MissingStateAsZero(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
		{cfg: ParkConfig{ParkID: "p2"}, index: 1},
	}
	stateIdx := map[string]*ParkState{
		"p1": {ParkID: "p1", CurrentLoad: 5},
		// p2 no tiene estado → carga 0
	}
	result := selectLeastLoaded(candidates, stateIdx)
	if result.cfg.ParkID != "p2" {
		t.Errorf("expected p2 (no state = load 0), got %q", result.cfg.ParkID)
	}
}

func TestSelectLeastLoaded_TiebreakByIndex(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p2"}, index: 1},
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
	}
	stateIdx := map[string]*ParkState{
		"p1": {ParkID: "p1", CurrentLoad: 5},
		"p2": {ParkID: "p2", CurrentLoad: 5},
	}
	result := selectLeastLoaded(candidates, stateIdx)
	if result.cfg.ParkID != "p1" {
		t.Errorf("expected p1 (lower index tiebreak), got %q", result.cfg.ParkID)
	}
}

func TestSelectLeastLoaded_SingleCandidate(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
	}
	stateIdx := map[string]*ParkState{
		"p1": {ParkID: "p1", CurrentLoad: 99},
	}
	result := selectLeastLoaded(candidates, stateIdx)
	if result.cfg.ParkID != "p1" {
		t.Errorf("expected p1, got %q", result.cfg.ParkID)
	}
}

// ---------------------------------------------------------------------------
// selectWeighted
// ---------------------------------------------------------------------------

func TestSelectWeighted_ChoosesHighest(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1", Weight: 5}, index: 0},
		{cfg: ParkConfig{ParkID: "p2", Weight: 10}, index: 1},
		{cfg: ParkConfig{ParkID: "p3", Weight: 3}, index: 2},
	}
	result := selectWeighted(candidates)
	if result.cfg.ParkID != "p2" {
		t.Errorf("expected p2 (weight=10), got %q", result.cfg.ParkID)
	}
}

func TestSelectWeighted_ZeroWeightTreatedAsOne(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1", Weight: 0}, index: 0},
		{cfg: ParkConfig{ParkID: "p2", Weight: -1}, index: 1},
		{cfg: ParkConfig{ParkID: "p3", Weight: 2}, index: 2},
	}
	result := selectWeighted(candidates)
	if result.cfg.ParkID != "p3" {
		t.Errorf("expected p3 (weight=2, others normalized to 1), got %q", result.cfg.ParkID)
	}
}

func TestSelectWeighted_TiebreakByIndex(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p2", Weight: 5}, index: 1},
		{cfg: ParkConfig{ParkID: "p1", Weight: 5}, index: 0},
	}
	result := selectWeighted(candidates)
	if result.cfg.ParkID != "p1" {
		t.Errorf("expected p1 (lower index tiebreak), got %q", result.cfg.ParkID)
	}
}

// ---------------------------------------------------------------------------
// selectRoundRobin
// ---------------------------------------------------------------------------

func TestSelectRoundRobin_Rotates(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
		{cfg: ParkConfig{ParkID: "p2"}, index: 1},
		{cfg: ParkConfig{ParkID: "p3"}, index: 2},
	}
	var counter uint64

	r1 := selectRoundRobin(candidates, &counter)
	r2 := selectRoundRobin(candidates, &counter)
	r3 := selectRoundRobin(candidates, &counter)
	r4 := selectRoundRobin(candidates, &counter)

	if r1.cfg.ParkID != "p1" {
		t.Errorf("call 1: expected p1, got %q", r1.cfg.ParkID)
	}
	if r2.cfg.ParkID != "p2" {
		t.Errorf("call 2: expected p2, got %q", r2.cfg.ParkID)
	}
	if r3.cfg.ParkID != "p3" {
		t.Errorf("call 3: expected p3, got %q", r3.cfg.ParkID)
	}
	if r4.cfg.ParkID != "p1" {
		t.Errorf("call 4: expected p1 (wrap-around), got %q", r4.cfg.ParkID)
	}
}

func TestSelectRoundRobin_SingleCandidate_NoPanic(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
	}
	var counter uint64

	for i := 0; i < 5; i++ {
		r := selectRoundRobin(candidates, &counter)
		if r.cfg.ParkID != "p1" {
			t.Errorf("call %d: expected p1, got %q", i+1, r.cfg.ParkID)
		}
	}
}

func TestSelectRoundRobin_ConcurrencySafe(t *testing.T) {
	candidates := []parkCandidate{
		{cfg: ParkConfig{ParkID: "p1"}, index: 0},
		{cfg: ParkConfig{ParkID: "p2"}, index: 1},
		{cfg: ParkConfig{ParkID: "p3"}, index: 2},
	}
	var counter uint64

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := selectRoundRobin(candidates, &counter)
			if r.cfg.ParkID == "" {
				t.Error("got empty ParkID from concurrent selectRoundRobin")
			}
		}()
	}
	wg.Wait()

	// Counter should have advanced exactly 100 times.
	if counter != 100 {
		t.Errorf("expected counter=100, got %d", counter)
	}
}

// ---------------------------------------------------------------------------
// effectiveWeight
// ---------------------------------------------------------------------------

func TestEffectiveWeight_Values(t *testing.T) {
	cases := []struct {
		input    int
		expected int
	}{
		{-5, 1},
		{-1, 1},
		{0, 1},
		{1, 1},
		{5, 5},
		{100, 100},
	}
	for _, tc := range cases {
		got := effectiveWeight(tc.input)
		if got != tc.expected {
			t.Errorf("effectiveWeight(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// currentLoadOf
// ---------------------------------------------------------------------------

func TestCurrentLoadOf_WithState(t *testing.T) {
	stateIdx := map[string]*ParkState{
		"p1": {ParkID: "p1", CurrentLoad: 42},
	}
	if got := currentLoadOf("p1", stateIdx); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestCurrentLoadOf_NilState(t *testing.T) {
	stateIdx := map[string]*ParkState{}
	if got := currentLoadOf("p1", stateIdx); got != 0 {
		t.Errorf("expected 0 for missing state, got %d", got)
	}
}
