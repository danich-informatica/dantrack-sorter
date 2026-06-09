package sorter

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// BoxBuilder
// ---------------------------------------------------------------------------

func TestNewBoxMinimal(t *testing.T) {
	box := NewBox("BOX-001").Build()
	if box.BoxID != "BOX-001" {
		t.Errorf("BoxID = %q, want %q", box.BoxID, "BOX-001")
	}
}

func TestBoxBuilderFields(t *testing.T) {
	box := NewBox("BOX-002").
		QR("QR123").
		SKU("SKU-100").
		Variety("Fuji").
		Caliber("80").
		Category("Premium").
		Client("ClientA").
		PackageType("box_18kg").
		Destination("export").
		Flow("line_1").
		Source("scanner_2").
		ReadStatus("ok").
		QualityStatus("approved").
		Build()

	if box.QR != "QR123" {
		t.Errorf("QR = %q, want %q", box.QR, "QR123")
	}
	if box.SKU != "SKU-100" {
		t.Errorf("SKU = %q, want %q", box.SKU, "SKU-100")
	}
	if box.Variety != "Fuji" {
		t.Errorf("Variety = %q, want %q", box.Variety, "Fuji")
	}
	if box.Caliber != "80" {
		t.Errorf("Caliber = %q, want %q", box.Caliber, "80")
	}
	if box.Category != "Premium" {
		t.Errorf("Category = %q, want %q", box.Category, "Premium")
	}
	if box.Client != "ClientA" {
		t.Errorf("Client = %q, want %q", box.Client, "ClientA")
	}
	if box.PackageType != "box_18kg" {
		t.Errorf("PackageType = %q, want %q", box.PackageType, "box_18kg")
	}
	if box.Destination != "export" {
		t.Errorf("Destination = %q, want %q", box.Destination, "export")
	}
	if box.Flow != "line_1" {
		t.Errorf("Flow = %q, want %q", box.Flow, "line_1")
	}
	if box.Source != "scanner_2" {
		t.Errorf("Source = %q, want %q", box.Source, "scanner_2")
	}
	if box.ReadStatus != "ok" {
		t.Errorf("ReadStatus = %q, want %q", box.ReadStatus, "ok")
	}
	if box.QualityStatus != "approved" {
		t.Errorf("QualityStatus = %q, want %q", box.QualityStatus, "approved")
	}
}

func TestBoxBuilderTags(t *testing.T) {
	box := NewBox("BOX-003").
		Tag("lote", "L001").
		Tag("turno", "mañana").
		Build()

	if box.Tags == nil {
		t.Fatal("Tags = nil, want map with 2 entries")
	}
	if box.Tags["lote"] != "L001" {
		t.Errorf("Tags[lote] = %q, want %q", box.Tags["lote"], "L001")
	}
	if box.Tags["turno"] != "mañana" {
		t.Errorf("Tags[turno] = %q, want %q", box.Tags["turno"], "mañana")
	}
}

func TestBoxBuilderErrorFlags(t *testing.T) {
	box := NewBox("BOX-004").
		ErrorFlag("damaged_label").
		ErrorFlag("weight_mismatch").
		Build()

	if len(box.ErrorFlags) != 2 {
		t.Fatalf("ErrorFlags length = %d, want 2", len(box.ErrorFlags))
	}
	if box.ErrorFlags[0] != "damaged_label" {
		t.Errorf("ErrorFlags[0] = %q, want %q", box.ErrorFlags[0], "damaged_label")
	}
}

func TestBoxBuilderTimestamp(t *testing.T) {
	ts := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	box := NewBox("BOX-005").Timestamp(ts).Build()
	if !box.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", box.Timestamp, ts)
	}
}

func TestBoxBuilderImmutability(t *testing.T) {
	b1 := NewBox("BOX-006").Tag("k", "v1")
	b2 := b1.Tag("k", "v2")

	box1 := b1.Build()
	box2 := b2.Build()

	if box1.Tags["k"] == box2.Tags["k"] {
		t.Error("builders should be independent; both have same tag value")
	}
}

func TestErrorBox(t *testing.T) {
	box := ErrorBox("BOX-ERR", "no_read")
	if box.BoxID != "BOX-ERR" {
		t.Errorf("BoxID = %q, want %q", box.BoxID, "BOX-ERR")
	}
	if box.ReadStatus != "no_read" {
		t.Errorf("ReadStatus = %q, want %q", box.ReadStatus, "no_read")
	}
	if !boxRequiresErrorControl(box) {
		t.Error("ErrorBox should be detected as requiring error control")
	}
}

func TestErrorBoxWithErrorStatus(t *testing.T) {
	box := ErrorBox("BOX-ERR2", "error")
	if !boxRequiresErrorControl(box) {
		t.Error("ErrorBox with 'error' ReadStatus should require error control")
	}
}

// ---------------------------------------------------------------------------
// AssignmentBuilder
// ---------------------------------------------------------------------------

func TestNewAssignmentDefaults(t *testing.T) {
	a := NewAssignment("a1", 10).Build()
	if a.ID != "a1" {
		t.Errorf("ID = %q, want %q", a.ID, "a1")
	}
	if a.Priority != 10 {
		t.Errorf("Priority = %d, want 10", a.Priority)
	}
	if !a.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestAssignmentBuilderEnabled(t *testing.T) {
	a := NewAssignment("a1", 5).Enabled(false).Build()
	if a.Enabled {
		t.Error("Enabled = true, want false")
	}
	a2 := NewAssignment("a2", 5).Enabled(true).Build()
	if !a2.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestAssignmentBuilderDisabled(t *testing.T) {
	a := NewAssignment("a1", 5).Disabled().Build()
	if a.Enabled {
		t.Error("Enabled = true, want false")
	}
}

func TestAssignmentBuilderMatchEquals(t *testing.T) {
	a := NewAssignment("a1", 10).
		MatchEquals("variety", "Fuji").
		Build()
	if len(a.Conditions) != 1 {
		t.Fatalf("Conditions length = %d, want 1", len(a.Conditions))
	}
	if a.Conditions[0].Operator != OpEquals {
		t.Errorf("Operator = %q, want %q", a.Conditions[0].Operator, OpEquals)
	}
	if a.Conditions[0].Values[0] != "Fuji" {
		t.Errorf("Values[0] = %q, want %q", a.Conditions[0].Values[0], "Fuji")
	}
}

func TestAssignmentBuilderMatchIn(t *testing.T) {
	a := NewAssignment("a1", 10).
		MatchIn("caliber", "80", "85").
		Build()
	if len(a.Conditions) != 1 {
		t.Fatalf("Conditions length = %d, want 1", len(a.Conditions))
	}
	if a.Conditions[0].Operator != OpIn {
		t.Errorf("Operator = %q, want %q", a.Conditions[0].Operator, OpIn)
	}
	if len(a.Conditions[0].Values) != 2 {
		t.Errorf("Values length = %d, want 2", len(a.Conditions[0].Values))
	}
}

func TestAssignmentBuilderMatchGeneric(t *testing.T) {
	a := NewAssignment("a1", 10).
		Match("sku", OpNotIn, "SKU-BAD", "SKU-OLD").
		Build()
	if a.Conditions[0].Operator != OpNotIn {
		t.Errorf("Operator = %q, want %q", a.Conditions[0].Operator, OpNotIn)
	}
	if len(a.Conditions[0].Values) != 2 {
		t.Errorf("Values length = %d, want 2", len(a.Conditions[0].Values))
	}
}

func TestAssignmentBuilderMatchExists(t *testing.T) {
	a := NewAssignment("a1", 10).
		MatchExists("tags.lote").
		Build()
	if a.Conditions[0].Operator != OpExists {
		t.Errorf("Operator = %q, want %q", a.Conditions[0].Operator, OpExists)
	}
}

func TestAssignmentBuilderToExit(t *testing.T) {
	a := NewAssignment("a1", 10).
		ToExit("exit_A").
		Build()
	if a.TargetType != TargetTypeExit {
		t.Errorf("TargetType = %q, want %q", a.TargetType, TargetTypeExit)
	}
	if len(a.TargetIDs) != 1 || a.TargetIDs[0] != "exit_A" {
		t.Errorf("TargetIDs = %v, want [exit_A]", a.TargetIDs)
	}
}

func TestAssignmentBuilderToExits(t *testing.T) {
	a := NewAssignment("a1", 10).
		ToExits("exit_A", "exit_B").
		Build()
	if a.TargetType != TargetTypeExit {
		t.Errorf("TargetType = %q, want %q", a.TargetType, TargetTypeExit)
	}
	if len(a.TargetIDs) != 2 {
		t.Fatalf("TargetIDs length = %d, want 2", len(a.TargetIDs))
	}
	if a.TargetIDs[0] != "exit_A" || a.TargetIDs[1] != "exit_B" {
		t.Errorf("TargetIDs = %v, want [exit_A, exit_B]", a.TargetIDs)
	}
}

func TestAssignmentBuilderToPark(t *testing.T) {
	a := NewAssignment("a1", 10).
		ToPark("park_1").
		Build()
	if a.TargetType != TargetTypePark {
		t.Errorf("TargetType = %q, want %q", a.TargetType, TargetTypePark)
	}
	if len(a.TargetIDs) != 1 || a.TargetIDs[0] != "park_1" {
		t.Errorf("TargetIDs = %v, want [park_1]", a.TargetIDs)
	}
}

func TestAssignmentBuilderToParks(t *testing.T) {
	a := NewAssignment("a1", 10).
		ToParks("park_1", "park_2").
		Build()
	if a.TargetType != TargetTypePark {
		t.Errorf("TargetType = %q, want %q", a.TargetType, TargetTypePark)
	}
	if len(a.TargetIDs) != 2 {
		t.Errorf("TargetIDs length = %d, want 2", len(a.TargetIDs))
	}
}

func TestAssignmentBuilderValidBetween(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	a := NewAssignment("a1", 10).
		ValidBetween(from, to).
		Build()
	if !a.ValidFrom.Equal(from) {
		t.Errorf("ValidFrom = %v, want %v", a.ValidFrom, from)
	}
	if !a.ValidTo.Equal(to) {
		t.Errorf("ValidTo = %v, want %v", a.ValidTo, to)
	}
}

func TestAssignmentBuilderMetadata(t *testing.T) {
	a := NewAssignment("a1", 10).
		Metadata("source", "db").
		Metadata("version", "2").
		Build()
	if a.Metadata == nil {
		t.Fatal("Metadata = nil, want map with 2 entries")
	}
	if a.Metadata["source"] != "db" {
		t.Errorf("Metadata[source] = %q, want %q", a.Metadata["source"], "db")
	}
	if a.Metadata["version"] != "2" {
		t.Errorf("Metadata[version] = %q, want %q", a.Metadata["version"], "2")
	}
}

func TestAssignmentBuilderPassesValidation(t *testing.T) {
	a := NewAssignment("a1", 10).
		MatchEquals("variety", "Fuji").
		ToExit("exit_A").
		Build()

	err := validateAssignment(a, 0)
	if err != nil {
		t.Errorf("assignment built with builder failed validation: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration: builders + real engine
// ---------------------------------------------------------------------------

func TestBuilders_EvaluateAssignments(t *testing.T) {
	box := NewBox("BOX-INT").
		Variety("Fuji").
		Caliber("80").
		Build()

	assignments := AssignmentSet{
		Assignments: []Assignment{
			NewAssignment("a1", 10).
				MatchEquals("variety", "Fuji").
				MatchIn("caliber", "80", "85").
				ToExit("exit_A").
				Build(),
			NewAssignment("a2", 5).
				MatchEquals("variety", "Gala").
				ToExit("exit_B").
				Build(),
		},
	}

	evalTime := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	matches := EvaluateAssignments(box, assignments, evalTime)

	if len(matches) != 2 {
		t.Fatalf("matches length = %d, want 2", len(matches))
	}
	if !matches[0].Matched {
		t.Error("a1 should match")
	}
	if matches[0].Assignment.ID != "a1" {
		t.Errorf("first match ID = %q, want %q", matches[0].Assignment.ID, "a1")
	}
	if matches[1].Matched {
		t.Error("a2 should not match (variety != Gala)")
	}
}

func TestBuilders_ResolveSorter(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Sorter: &SorterConfig{
			SorterID:      "s1",
			Exits:         []SorterExit{{ExitID: "exit_A", Enabled: true}, {ExitID: "exit_B", Enabled: true}},
			DefaultExitID: "exit_A",
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	decision, err := engine.ResolveSorter(context.Background(), SorterRequest{
		TraceID: "trace-builder-test",
		Box:     NewBox("BOX-S1").Variety("Fuji").Build(),
		Assignments: AssignmentSet{
			Assignments: []Assignment{
				NewAssignment("a1", 10).
					MatchEquals("variety", "Fuji").
					ToExit("exit_A").
					Build(),
			},
		},
		ExitStates: ExitStates(
			ExitAvailable("exit_A", 10),
			ExitAvailable("exit_B", 5),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolveSorter: %v", err)
	}
	if decision.Action != ActionRoute {
		t.Errorf("Action = %q, want %q", decision.Action, ActionRoute)
	}
	if decision.ExitID != "exit_A" {
		t.Errorf("ExitID = %q, want %q", decision.ExitID, "exit_A")
	}
}

func TestBuilders_ResolvePresorter(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Presorter: &PresorterConfig{
			Parks: []ParkConfig{
				{ParkID: "park_1", Enabled: true},
				{ParkID: "park_2", Enabled: true},
			},
			BalanceStrategy: BalanceLeastLoaded,
			DefaultParkID:   "park_1",
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	decision, err := engine.ResolvePresorter(context.Background(), PresorterRequest{
		TraceID: "trace-builder-presorter",
		Box:     NewBox("BOX-P1").QR("QR-OK").Build(),
		ParkStates: ParkStates(
			ParkAvailable("park_1", 45),
			ParkAvailable("park_2", 12),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolvePresorter: %v", err)
	}
	if decision.Action != ActionRoute {
		t.Errorf("Action = %q, want %q", decision.Action, ActionRoute)
	}
	if decision.ParkID != "park_2" {
		t.Errorf("ParkID = %q, want %q (least loaded)", decision.ParkID, "park_2")
	}
}

func TestBuilders_PresorterWithWindowBalance(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		Presorter: &PresorterConfig{
			Parks: []ParkConfig{
				{ParkID: "park_1", Enabled: true},
				{ParkID: "park_2", Enabled: true},
			},
			BalanceStrategy: BalanceLeastLoaded,
			DefaultParkID:   "park_1",
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	decision, err := engine.ResolvePresorter(context.Background(), PresorterRequest{
		TraceID: "trace-builder-window",
		Box:     NewBox("BOX-W1").Build(),
		ParkStates: ParkStates(
			ParkAvailableWithWindow("park_1", 10, 50),
			ParkAvailableWithWindow("park_2", 20, 30),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ResolvePresorter: %v", err)
	}
	if decision.Action != ActionRoute {
		t.Errorf("Action = %q, want %q", decision.Action, ActionRoute)
	}
	// park_2 has lower AccumulatedLoad (30 < 50).
	if decision.ParkID != "park_2" {
		t.Errorf("ParkID = %q, want %q (lower AccumulatedLoad)", decision.ParkID, "park_2")
	}
}
