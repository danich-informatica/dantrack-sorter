package sorter

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var availTestTime = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)

func availSkuAssign(id string, sku string, targetIDs ...string) Assignment {
	a := Assignment{
		ID:         id,
		Enabled:    true,
		TargetType: TargetTypeExit,
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpEquals, Values: []string{sku}},
		},
	}
	if len(targetIDs) == 1 {
		a.TargetID = targetIDs[0]
	} else if len(targetIDs) > 1 {
		a.TargetIDs = targetIDs
	}
	return a
}

func availSkuInAssign(id string, skus []string, targetIDs ...string) Assignment {
	a := Assignment{
		ID:         id,
		Enabled:    true,
		TargetType: TargetTypeExit,
		Conditions: []MatchCondition{
			{Field: "sku", Operator: OpIn, Values: skus},
		},
	}
	if len(targetIDs) == 1 {
		a.TargetID = targetIDs[0]
	} else if len(targetIDs) > 1 {
		a.TargetIDs = targetIDs
	}
	return a
}

func availExits() []SorterExit {
	return []SorterExit{
		{ExitID: "exit_1", Enabled: true},
		{ExitID: "exit_2", Enabled: true},
		{ExitID: "exit_3", Enabled: true},
	}
}

func availExitState(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true}
}

func availParks() []ParkConfig {
	return []ParkConfig{
		{ParkID: "park_1", Enabled: true},
		{ParkID: "park_2", Enabled: true},
		{ParkID: "park_3", Enabled: true},
	}
}

func availParkState(parkID string, available bool) ParkState {
	return ParkState{ParkID: parkID, Available: available}
}

// ---------------------------------------------------------------------------
// AvailableSKUsPerExit tests
// ---------------------------------------------------------------------------

func TestAvailSKUsPerExit_LegacyTargetID(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_APPLE", "exit_1"),
	}}
	states := []SorterExitState{availExitState("exit_1"), availExitState("exit_2")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if got := result["exit_1"]; len(got) != 1 || got[0] != "SKU_APPLE" {
		t.Fatalf("expected [SKU_APPLE], got %v", got)
	}
	if _, ok := result["exit_2"]; ok {
		t.Fatal("exit_2 should not have SKUs")
	}
}

func TestAvailSKUsPerExit_MultiTarget(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_ORANGE", "exit_1", "exit_2"),
	}}
	states := []SorterExitState{availExitState("exit_1"), availExitState("exit_2")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	for _, exitID := range []string{"exit_1", "exit_2"} {
		got := result[exitID]
		if len(got) != 1 || got[0] != "SKU_ORANGE" {
			t.Fatalf("%s: expected [SKU_ORANGE], got %v", exitID, got)
		}
	}
}

func TestAvailSKUsPerExit_ExitFull(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	states := []SorterExitState{{ExitID: "exit_1", Available: true, Full: true}}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if _, ok := result["exit_1"]; ok {
		t.Fatal("full exit should not have SKUs")
	}
}

func TestAvailSKUsPerExit_ExitBlocked(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	states := []SorterExitState{{ExitID: "exit_1", Available: true, Blocked: true}}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if _, ok := result["exit_1"]; ok {
		t.Fatal("blocked exit should not have SKUs")
	}
}

func TestAvailSKUsPerExit_ExitMaintenance(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	states := []SorterExitState{{ExitID: "exit_1", Available: true, Maintenance: true}}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if _, ok := result["exit_1"]; ok {
		t.Fatal("maintenance exit should not have SKUs")
	}
}

func TestAvailSKUsPerExit_ExitHasError(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	states := []SorterExitState{{ExitID: "exit_1", Available: true, HasError: true}}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if _, ok := result["exit_1"]; ok {
		t.Fatal("error exit should not have SKUs")
	}
}

func TestAvailSKUsPerExit_NoState_AssumeAvailable(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_Y", "exit_2"),
	}}
	// No states provided at all.
	result := AvailableSKUsPerExit(assignments, availExits(), nil, availTestTime)

	if got := result["exit_2"]; len(got) != 1 || got[0] != "SKU_Y" {
		t.Fatalf("expected [SKU_Y], got %v", got)
	}
}

func TestAvailSKUsPerExit_AssignmentDisabled(t *testing.T) {
	a := availSkuAssign("a1", "SKU_DISABLED", "exit_1")
	a.Enabled = false
	assignments := AssignmentSet{Assignments: []Assignment{a}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if len(result) != 0 {
		t.Fatalf("disabled assignment should be omitted, got %v", result)
	}
}

func TestAvailSKUsPerExit_AssignmentExpired(t *testing.T) {
	a := availSkuAssign("a1", "SKU_OLD", "exit_1")
	a.ValidTo = availTestTime.Add(-1 * time.Hour)
	assignments := AssignmentSet{Assignments: []Assignment{a}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if len(result) != 0 {
		t.Fatalf("expired assignment should be omitted, got %v", result)
	}
}

func TestAvailSKUsPerExit_AssignmentFuture(t *testing.T) {
	a := availSkuAssign("a1", "SKU_FUTURE", "exit_1")
	a.ValidFrom = availTestTime.Add(1 * time.Hour)
	assignments := AssignmentSet{Assignments: []Assignment{a}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if len(result) != 0 {
		t.Fatalf("future assignment should be omitted, got %v", result)
	}
}

func TestAvailSKUsPerExit_OpEquals(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_EQ", "exit_1"),
	}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if got := result["exit_1"]; len(got) != 1 || got[0] != "SKU_EQ" {
		t.Fatalf("expected [SKU_EQ], got %v", got)
	}
}

func TestAvailSKUsPerExit_OpIn(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuInAssign("a1", []string{"SKU_A", "SKU_B", "SKU_C"}, "exit_1"),
	}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	got := result["exit_1"]
	expected := []string{"SKU_A", "SKU_B", "SKU_C"}
	if len(got) != 3 {
		t.Fatalf("expected 3 SKUs, got %v", got)
	}
	for i, sku := range expected {
		if got[i] != sku {
			t.Fatalf("index %d: expected %s, got %s", i, sku, got[i])
		}
	}
}

func TestAvailSKUsPerExit_NoSKUCondition_Omitted(t *testing.T) {
	a := Assignment{
		ID:       "a1",
		Enabled:  true,
		TargetID: "exit_1",
		Conditions: []MatchCondition{
			{Field: "variety", Operator: OpEquals, Values: []string{"Fuji"}},
		},
	}
	assignments := AssignmentSet{Assignments: []Assignment{a}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if len(result) != 0 {
		t.Fatalf("assignment without sku condition should be omitted, got %v", result)
	}
}

func TestAvailSKUsPerExit_DuplicatesEliminated(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_DUP", "exit_1"),
		availSkuAssign("a2", "SKU_DUP", "exit_1"),
	}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if got := result["exit_1"]; len(got) != 1 || got[0] != "SKU_DUP" {
		t.Fatalf("expected single [SKU_DUP], got %v", got)
	}
}

func TestAvailSKUsPerExit_ResultSorted(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "ZZZ", "exit_1"),
		availSkuAssign("a2", "AAA", "exit_1"),
		availSkuAssign("a3", "MMM", "exit_1"),
	}}
	states := []SorterExitState{availExitState("exit_1")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	got := result["exit_1"]
	expected := []string{"AAA", "MMM", "ZZZ"}
	if len(got) != 3 {
		t.Fatalf("expected 3 SKUs, got %v", got)
	}
	for i, sku := range expected {
		if got[i] != sku {
			t.Fatalf("index %d: expected %s, got %s", i, sku, got[i])
		}
	}
}

// ---------------------------------------------------------------------------
// AvailableSKUsPerPark tests
// ---------------------------------------------------------------------------

func TestAvailSKUsPerPark_AllSKUsInAllParks(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_1", "exit_1"),
		availSkuAssign("a2", "SKU_2", "exit_2"),
	}}
	parks := availParks()
	states := []ParkState{
		availParkState("park_1", true),
		availParkState("park_2", true),
		availParkState("park_3", true),
	}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	for _, parkID := range []string{"park_1", "park_2", "park_3"} {
		got := result[parkID]
		if len(got) != 2 || got[0] != "SKU_1" || got[1] != "SKU_2" {
			t.Fatalf("%s: expected [SKU_1 SKU_2], got %v", parkID, got)
		}
	}
}

func TestAvailSKUsPerPark_ParkFull(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	parks := availParks()
	states := []ParkState{
		{ParkID: "park_1", Available: true, Full: true},
		availParkState("park_2", true),
	}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if _, ok := result["park_1"]; ok {
		t.Fatal("full park should be excluded")
	}
	if got := result["park_2"]; len(got) != 1 || got[0] != "SKU_X" {
		t.Fatalf("park_2 expected [SKU_X], got %v", got)
	}
}

func TestAvailSKUsPerPark_ParkBlocked(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	parks := availParks()
	states := []ParkState{
		{ParkID: "park_1", Available: true, Blocked: true},
		availParkState("park_2", true),
	}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if _, ok := result["park_1"]; ok {
		t.Fatal("blocked park should be excluded")
	}
}

func TestAvailSKUsPerPark_ParkMaintenance(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	parks := availParks()
	states := []ParkState{
		{ParkID: "park_1", Available: true, Maintenance: true},
		availParkState("park_2", true),
	}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if _, ok := result["park_1"]; ok {
		t.Fatal("maintenance park should be excluded")
	}
}

func TestAvailSKUsPerPark_ParkDisabled(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1"),
	}}
	parks := []ParkConfig{
		{ParkID: "park_1", Enabled: false},
		{ParkID: "park_2", Enabled: true},
	}
	states := []ParkState{
		availParkState("park_1", true),
		availParkState("park_2", true),
	}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if _, ok := result["park_1"]; ok {
		t.Fatal("disabled park should be excluded")
	}
	if got := result["park_2"]; len(got) != 1 || got[0] != "SKU_X" {
		t.Fatalf("park_2 expected [SKU_X], got %v", got)
	}
}

func TestAvailSKUsPerPark_NoState_AssumeAvailable(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_Z", "exit_1"),
	}}
	parks := availParks()
	// No states → all assumed available.
	result := AvailableSKUsPerPark(assignments, parks, nil, availTestTime)

	for _, parkID := range []string{"park_1", "park_2", "park_3"} {
		got := result[parkID]
		if len(got) != 1 || got[0] != "SKU_Z" {
			t.Fatalf("%s: expected [SKU_Z], got %v", parkID, got)
		}
	}
}

func TestAvailSKUsPerPark_AssignmentInactive(t *testing.T) {
	disabled := availSkuAssign("a1", "SKU_D", "exit_1")
	disabled.Enabled = false

	expired := availSkuAssign("a2", "SKU_E", "exit_1")
	expired.ValidTo = availTestTime.Add(-1 * time.Hour)

	future := availSkuAssign("a3", "SKU_F", "exit_1")
	future.ValidFrom = availTestTime.Add(1 * time.Hour)

	assignments := AssignmentSet{Assignments: []Assignment{disabled, expired, future}}
	parks := availParks()
	states := []ParkState{availParkState("park_1", true)}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if len(result) != 0 {
		t.Fatalf("inactive assignments should produce empty result, got %v", result)
	}
}

func TestAvailSKUsPerPark_DuplicatesEliminated(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_DUP", "exit_1"),
		availSkuAssign("a2", "SKU_DUP", "exit_2"),
	}}
	parks := []ParkConfig{{ParkID: "park_1", Enabled: true}}
	states := []ParkState{availParkState("park_1", true)}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	if got := result["park_1"]; len(got) != 1 || got[0] != "SKU_DUP" {
		t.Fatalf("expected single [SKU_DUP], got %v", got)
	}
}

func TestAvailSKUsPerPark_ResultSorted(t *testing.T) {
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "ZZZ", "exit_1"),
		availSkuAssign("a2", "AAA", "exit_1"),
		availSkuAssign("a3", "MMM", "exit_1"),
	}}
	parks := []ParkConfig{{ParkID: "park_1", Enabled: true}}
	states := []ParkState{availParkState("park_1", true)}
	result := AvailableSKUsPerPark(assignments, parks, states, availTestTime)

	got := result["park_1"]
	expected := []string{"AAA", "MMM", "ZZZ"}
	if len(got) != 3 {
		t.Fatalf("expected 3 SKUs, got %v", got)
	}
	for i, sku := range expected {
		if got[i] != sku {
			t.Fatalf("index %d: expected %s, got %s", i, sku, got[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestAvailSKUsPerExit_ExitNotInConfig(t *testing.T) {
	// Assignment targets an exit not in config → should not appear.
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_GHOST", "exit_unknown"),
	}}
	states := []SorterExitState{availExitState("exit_unknown")}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if _, ok := result["exit_unknown"]; ok {
		t.Fatal("exit not in config should not appear in result")
	}
}

func TestAvailSKUsPerExit_EmptyAssignments(t *testing.T) {
	result := AvailableSKUsPerExit(AssignmentSet{}, availExits(), nil, availTestTime)
	if len(result) != 0 {
		t.Fatalf("empty assignments should return empty map, got %v", result)
	}
}

func TestAvailSKUsPerPark_EmptyAssignments(t *testing.T) {
	parks := availParks()
	states := []ParkState{availParkState("park_1", true)}
	result := AvailableSKUsPerPark(AssignmentSet{}, parks, states, availTestTime)
	if len(result) != 0 {
		t.Fatalf("empty assignments should return empty map, got %v", result)
	}
}

func TestAvailSKUsPerExit_MultiTarget_PartialAvailability(t *testing.T) {
	// exit_1 available, exit_2 full → SKU only in exit_1.
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_PARTIAL", "exit_1", "exit_2"),
	}}
	states := []SorterExitState{
		availExitState("exit_1"),
		{ExitID: "exit_2", Available: true, Full: true},
	}
	result := AvailableSKUsPerExit(assignments, availExits(), states, availTestTime)

	if got := result["exit_1"]; len(got) != 1 || got[0] != "SKU_PARTIAL" {
		t.Fatalf("exit_1 expected [SKU_PARTIAL], got %v", got)
	}
	if _, ok := result["exit_2"]; ok {
		t.Fatal("exit_2 is full, should not have SKUs")
	}
}

func TestAvailSKUsPerExit_DisabledExitInConfig(t *testing.T) {
	exits := []SorterExit{
		{ExitID: "exit_1", Enabled: false},
		{ExitID: "exit_2", Enabled: true},
	}
	assignments := AssignmentSet{Assignments: []Assignment{
		availSkuAssign("a1", "SKU_X", "exit_1", "exit_2"),
	}}
	states := []SorterExitState{availExitState("exit_1"), availExitState("exit_2")}
	result := AvailableSKUsPerExit(assignments, exits, states, availTestTime)

	if _, ok := result["exit_1"]; ok {
		t.Fatal("disabled exit in config should not have SKUs")
	}
	if got := result["exit_2"]; len(got) != 1 || got[0] != "SKU_X" {
		t.Fatalf("exit_2 expected [SKU_X], got %v", got)
	}
}
