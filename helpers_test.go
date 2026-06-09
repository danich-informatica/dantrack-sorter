package sorter

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Match helpers
// ---------------------------------------------------------------------------

func TestMatchEquals(t *testing.T) {
	mc := MatchEquals("variety", "Fuji")
	if mc.Field != "variety" {
		t.Errorf("Field = %q, want %q", mc.Field, "variety")
	}
	if mc.Operator != OpEquals {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpEquals)
	}
	if len(mc.Values) != 1 || mc.Values[0] != "Fuji" {
		t.Errorf("Values = %v, want [Fuji]", mc.Values)
	}
}

func TestMatchNotEquals(t *testing.T) {
	mc := MatchNotEquals("client", "C1")
	if mc.Operator != OpNotEquals {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpNotEquals)
	}
	if len(mc.Values) != 1 || mc.Values[0] != "C1" {
		t.Errorf("Values = %v, want [C1]", mc.Values)
	}
}

func TestMatchIn(t *testing.T) {
	mc := MatchIn("caliber", "80", "85", "90")
	if mc.Operator != OpIn {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpIn)
	}
	if len(mc.Values) != 3 {
		t.Errorf("Values length = %d, want 3", len(mc.Values))
	}
	if mc.Values[0] != "80" || mc.Values[1] != "85" || mc.Values[2] != "90" {
		t.Errorf("Values = %v, want [80, 85, 90]", mc.Values)
	}
}

func TestMatchNotIn(t *testing.T) {
	mc := MatchNotIn("client", "C1", "C2")
	if mc.Operator != OpNotIn {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpNotIn)
	}
	if len(mc.Values) != 2 {
		t.Errorf("Values length = %d, want 2", len(mc.Values))
	}
}

func TestMatchExists(t *testing.T) {
	mc := MatchExists("tags.lote")
	if mc.Operator != OpExists {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpExists)
	}
	if mc.Field != "tags.lote" {
		t.Errorf("Field = %q, want %q", mc.Field, "tags.lote")
	}
	if mc.Values != nil {
		t.Errorf("Values = %v, want nil", mc.Values)
	}
}

func TestMatchNotExists(t *testing.T) {
	mc := MatchNotExists("tags.lote")
	if mc.Operator != OpNotExists {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpNotExists)
	}
}

func TestMatchGeneric(t *testing.T) {
	mc := Match("sku", OpIn, "SKU-1", "SKU-2")
	if mc.Operator != OpIn {
		t.Errorf("Operator = %q, want %q", mc.Operator, OpIn)
	}
	if mc.Field != "sku" {
		t.Errorf("Field = %q, want %q", mc.Field, "sku")
	}
	if len(mc.Values) != 2 {
		t.Errorf("Values length = %d, want 2", len(mc.Values))
	}
}

// ---------------------------------------------------------------------------
// ExitStates / ExitState helpers
// ---------------------------------------------------------------------------

func TestExitStatesSlice(t *testing.T) {
	states := ExitStates(
		ExitAvailable("exit_A", 10),
		ExitBlocked("exit_B"),
	)
	if len(states) != 2 {
		t.Fatalf("ExitStates length = %d, want 2", len(states))
	}
	if states[0].ExitID != "exit_A" {
		t.Errorf("states[0].ExitID = %q, want %q", states[0].ExitID, "exit_A")
	}
	if states[1].ExitID != "exit_B" {
		t.Errorf("states[1].ExitID = %q, want %q", states[1].ExitID, "exit_B")
	}
}

func TestExitAvailable(t *testing.T) {
	s := ExitAvailable("exit_A", 30)
	if s.ExitID != "exit_A" {
		t.Errorf("ExitID = %q, want %q", s.ExitID, "exit_A")
	}
	if !s.Available {
		t.Error("Available = false, want true")
	}
	if s.CurrentLoad != 30 {
		t.Errorf("CurrentLoad = %d, want 30", s.CurrentLoad)
	}
}

func TestExitFull(t *testing.T) {
	s := ExitFull("exit_B")
	if !s.Full {
		t.Error("Full = false, want true")
	}
	if !s.Available {
		t.Error("Available = false, want true")
	}
}

func TestExitBlocked(t *testing.T) {
	s := ExitBlocked("exit_C")
	if !s.Blocked {
		t.Error("Blocked = false, want true")
	}
	if s.Available {
		t.Error("Available = true, want false")
	}
}

func TestExitMaintenance(t *testing.T) {
	s := ExitMaintenance("exit_D")
	if !s.Maintenance {
		t.Error("Maintenance = false, want true")
	}
	if s.Available {
		t.Error("Available = true, want false")
	}
}

func TestExitError(t *testing.T) {
	s := ExitError("exit_E")
	if !s.HasError {
		t.Error("HasError = false, want true")
	}
	if s.Available {
		t.Error("Available = true, want false")
	}
}

func TestExitUnavailable(t *testing.T) {
	s := ExitUnavailable("exit_F")
	if s.Available {
		t.Error("Available = true, want false")
	}
}

// ---------------------------------------------------------------------------
// ParkStates / ParkState helpers
// ---------------------------------------------------------------------------

func TestParkStatesSlice(t *testing.T) {
	states := ParkStates(
		ParkAvailable("park_1", 10),
		ParkFull("park_2"),
		ParkMaintenance("park_3"),
	)
	if len(states) != 3 {
		t.Fatalf("ParkStates length = %d, want 3", len(states))
	}
	if states[0].ParkID != "park_1" {
		t.Errorf("states[0].ParkID = %q, want %q", states[0].ParkID, "park_1")
	}
}

func TestParkAvailable(t *testing.T) {
	s := ParkAvailable("park_1", 45)
	if s.ParkID != "park_1" {
		t.Errorf("ParkID = %q, want %q", s.ParkID, "park_1")
	}
	if !s.Available {
		t.Error("Available = false, want true")
	}
	if s.CurrentLoad != 45 {
		t.Errorf("CurrentLoad = %d, want 45", s.CurrentLoad)
	}
}

func TestParkAvailableWithWindow(t *testing.T) {
	s := ParkAvailableWithWindow("park_1", 45, 120)
	if !s.Available {
		t.Error("Available = false, want true")
	}
	if s.CurrentLoad != 45 {
		t.Errorf("CurrentLoad = %d, want 45", s.CurrentLoad)
	}
	if s.AccumulatedLoad != 120 {
		t.Errorf("AccumulatedLoad = %d, want 120", s.AccumulatedLoad)
	}
}

func TestParkFull(t *testing.T) {
	s := ParkFull("park_2")
	if !s.Full {
		t.Error("Full = false, want true")
	}
}

func TestParkBlocked(t *testing.T) {
	s := ParkBlocked("park_3")
	if !s.Blocked {
		t.Error("Blocked = false, want true")
	}
	if s.Available {
		t.Error("Available = true, want false")
	}
}

func TestParkMaintenance(t *testing.T) {
	s := ParkMaintenance("park_4")
	if !s.Maintenance {
		t.Error("Maintenance = false, want true")
	}
}

func TestParkError(t *testing.T) {
	s := ParkError("park_5")
	if !s.HasError {
		t.Error("HasError = false, want true")
	}
}

func TestParkUnavailable(t *testing.T) {
	s := ParkUnavailable("park_6")
	if s.Available {
		t.Error("Available = true, want false")
	}
}
