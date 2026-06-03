package sorter

import (
	"sort"
	"time"
)

// AvailableSKUsPerExit returns a map of exit ID → sorted list of unique SKUs
// that have at least one active assignment targeting that exit.
//
// An exit is considered available if it exists in exits config AND its state
// (if present) is not Full, Blocked, HasError, Maintenance, or Available==false.
// If no state exists for a configured exit, it is assumed available.
//
// SKUs are inferred from assignment MatchConditions:
//   - Field=="sku" with OpEquals → Values[0]
//   - Field=="sku" with OpIn → all Values
//   - Assignments without a clear "sku" condition are omitted.
//
// Multi-target: if an assignment has TargetIDs [exit_1, exit_2], the SKU
// appears in both exits that are available.
//
// The function is pure, deterministic, and does not mutate inputs.
func AvailableSKUsPerExit(
	assignments AssignmentSet,
	exits []SorterExit,
	exitStates []SorterExitState,
	evalTime time.Time,
) map[string][]string {
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	// Build set of configured exit IDs.
	configuredExits := make(map[string]struct{}, len(exits))
	for _, e := range exits {
		if e.Enabled {
			configuredExits[e.ExitID] = struct{}{}
		}
	}

	// Index exit states by ExitID.
	stateIdx := make(map[string]*SorterExitState, len(exitStates))
	for i := range exitStates {
		stateIdx[exitStates[i].ExitID] = &exitStates[i]
	}

	// Determine available exits.
	availableExits := make(map[string]struct{}, len(configuredExits))
	for exitID := range configuredExits {
		if exitAvailable(exitID, stateIdx) {
			availableExits[exitID] = struct{}{}
		}
	}

	// Collect SKUs per exit.
	result := make(map[string][]string)
	skuSets := make(map[string]map[string]struct{})

	for _, a := range assignments.Assignments {
		if !assignmentActive(a, evalTime) {
			continue
		}

		skus := extractSKUs(a)
		if len(skus) == 0 {
			continue
		}

		targets := assignmentTargetIDs(a)
		for _, targetID := range targets {
			if _, ok := availableExits[targetID]; !ok {
				continue
			}
			if skuSets[targetID] == nil {
				skuSets[targetID] = make(map[string]struct{})
			}
			for _, sku := range skus {
				skuSets[targetID][sku] = struct{}{}
			}
		}
	}

	// Convert sets to sorted slices.
	for exitID, set := range skuSets {
		sorted := make([]string, 0, len(set))
		for sku := range set {
			sorted = append(sorted, sku)
		}
		sort.Strings(sorted)
		result[exitID] = sorted
	}

	return result
}

// AvailableSKUsPerPark returns a map of park ID → sorted list of unique SKUs
// from all active assignments, for each available park.
//
// A park is considered available if it exists in parks config, is Enabled,
// and its state (if present) is not Full, Blocked, HasError, Maintenance,
// or Available==false. If no state exists for a configured park, it is assumed available.
//
// All active SKUs appear in every available park (Opción A from spec):
// there is no assignment→park targeting restriction for this view.
//
// The function is pure, deterministic, and does not mutate inputs.
func AvailableSKUsPerPark(
	assignments AssignmentSet,
	parks []ParkConfig,
	parkStates []ParkState,
	evalTime time.Time,
) map[string][]string {
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	// Index park states by ParkID.
	stateIdx := make(map[string]*ParkState, len(parkStates))
	for i := range parkStates {
		stateIdx[parkStates[i].ParkID] = &parkStates[i]
	}

	// Determine available parks.
	availableParks := make([]string, 0, len(parks))
	for _, p := range parks {
		if !p.Enabled {
			continue
		}
		if parkAvailableForView(p.ParkID, stateIdx) {
			availableParks = append(availableParks, p.ParkID)
		}
	}

	if len(availableParks) == 0 {
		return make(map[string][]string)
	}

	// Collect all unique SKUs from active assignments.
	skuSet := make(map[string]struct{})
	for _, a := range assignments.Assignments {
		if !assignmentActive(a, evalTime) {
			continue
		}
		for _, sku := range extractSKUs(a) {
			skuSet[sku] = struct{}{}
		}
	}

	if len(skuSet) == 0 {
		return make(map[string][]string)
	}

	// Build sorted SKU list once.
	sortedSKUs := make([]string, 0, len(skuSet))
	for sku := range skuSet {
		sortedSKUs = append(sortedSKUs, sku)
	}
	sort.Strings(sortedSKUs)

	// Assign same SKU list to every available park.
	result := make(map[string][]string, len(availableParks))
	for _, parkID := range availableParks {
		cp := make([]string, len(sortedSKUs))
		copy(cp, sortedSKUs)
		result[parkID] = cp
	}

	return result
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// assignmentActive returns true if the assignment is enabled and within time range.
func assignmentActive(a Assignment, evalTime time.Time) bool {
	if !a.Enabled {
		return false
	}
	return isWithinTimeRange(evalTime, a.ValidFrom, a.ValidTo)
}

// extractSKUs extracts SKU values from an assignment's conditions.
// Returns nil if no clear "sku" condition with OpEquals or OpIn is found.
func extractSKUs(a Assignment) []string {
	var skus []string
	for _, cond := range a.Conditions {
		if cond.Field != "sku" {
			continue
		}
		switch cond.Operator {
		case OpEquals:
			if len(cond.Values) > 0 && cond.Values[0] != "" {
				skus = append(skus, cond.Values[0])
			}
		case OpIn:
			for _, v := range cond.Values {
				if v != "" {
					skus = append(skus, v)
				}
			}
		}
	}
	return skus
}

// exitAvailable determines if an exit is available based on its state.
// If no state is present, the exit is assumed available.
func exitAvailable(exitID string, stateIdx map[string]*SorterExitState) bool {
	st, ok := stateIdx[exitID]
	if !ok {
		return true
	}
	if st.Full || st.Blocked || st.HasError || st.Maintenance {
		return false
	}
	if !st.Available {
		return false
	}
	return true
}

// parkAvailableForView determines if a park is available based on its state.
// If no state is present, the park is assumed available.
func parkAvailableForView(parkID string, stateIdx map[string]*ParkState) bool {
	st, ok := stateIdx[parkID]
	if !ok {
		return true
	}
	if st.Full || st.Blocked || st.HasError || st.Maintenance {
		return false
	}
	if !st.Available {
		return false
	}
	return true
}
