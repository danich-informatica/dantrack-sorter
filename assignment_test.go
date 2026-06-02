package sorter

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func baseBox() BoxContext {
	return BoxContext{
		BoxID:         "BOX-001",
		QR:            "QR-001",
		SKU:           "SKU-APPLE",
		Variety:       "FUJI",
		Caliber:       "80",
		Category:      "CAT-A",
		Client:        "CLIENT-X",
		PackageType:   "PALLET",
		Destination:   "DEST-01",
		Flow:          "FLOW-A",
		Source:        "SCANNER-1",
		ReadStatus:    "ok",
		QualityStatus: "pass",
		Tags: map[string]string{
			"lote":    "LOTE-2026",
			"turno":   "mañana",
			"present": "",
		},
	}
}

func baseAssignment(id string, priority int) Assignment {
	return Assignment{
		ID:         id,
		Enabled:    true,
		Priority:   priority,
		TargetType: TargetTypeExit,
		TargetID:   "exit-1",
	}
}

func withCondition(a Assignment, field string, op MatchOperator, values ...string) Assignment {
	a.Conditions = append(a.Conditions, MatchCondition{
		Field:    field,
		Operator: op,
		Values:   values,
	})
	return a
}

func singleAssignmentSet(a Assignment) AssignmentSet {
	return AssignmentSet{Assignments: []Assignment{a}}
}

func fixedTime() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Operator: equals
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_EqualsMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpEquals, "SKU-APPLE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_EqualsNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpEquals, "SKU-OTHER")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match")
	}
}

// ---------------------------------------------------------------------------
// Operator: not_equals
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_NotEqualsMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotEquals, "SKU-OTHER")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_NotEqualsNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotEquals, "SKU-APPLE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match")
	}
}

// ---------------------------------------------------------------------------
// Operator: in
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_InMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpIn, "SKU-PEAR", "SKU-APPLE", "SKU-GRAPE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_InNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpIn, "SKU-PEAR", "SKU-GRAPE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match")
	}
}

// ---------------------------------------------------------------------------
// Operator: not_in
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_NotInMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotIn, "SKU-PEAR", "SKU-GRAPE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_NotInNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotIn, "SKU-APPLE", "SKU-PEAR")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match")
	}
}

// ---------------------------------------------------------------------------
// Operator: exists / not_exists
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_ExistsMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpExists)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_ExistsNoMatch(t *testing.T) {
	box := baseBox()
	box.SKU = ""
	a := withCondition(baseAssignment("a1", 1), "sku", OpExists)
	results := EvaluateAssignments(box, singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match for empty SKU with exists")
	}
}

func TestEvaluateAssignments_NotExistsMatch(t *testing.T) {
	box := baseBox()
	box.SKU = ""
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotExists)
	results := EvaluateAssignments(box, singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match for empty SKU with not_exists, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_NotExistsNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpNotExists)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match for non-empty SKU with not_exists")
	}
}

// ---------------------------------------------------------------------------
// Direct field: sku
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_DirectFieldSKU(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpEquals, "SKU-APPLE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match on sku field")
	}
	if results[0].ConditionResults[0].Actual != "SKU-APPLE" {
		t.Errorf("expected Actual=SKU-APPLE, got %q", results[0].ConditionResults[0].Actual)
	}
}

// ---------------------------------------------------------------------------
// All direct fields resolved correctly
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_AllDirectFields(t *testing.T) {
	box := baseBox()
	cases := []struct {
		field    string
		expected string
	}{
		{"box_id", "BOX-001"},
		{"qr", "QR-001"},
		{"sku", "SKU-APPLE"},
		{"variety", "FUJI"},
		{"caliber", "80"},
		{"category", "CAT-A"},
		{"client", "CLIENT-X"},
		{"package_type", "PALLET"},
		{"destination", "DEST-01"},
		{"flow", "FLOW-A"},
		{"source", "SCANNER-1"},
		{"read_status", "ok"},
		{"quality_status", "pass"},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			a := withCondition(baseAssignment("a-"+tc.field, 1), tc.field, OpEquals, tc.expected)
			results := EvaluateAssignments(box, singleAssignmentSet(a), fixedTime())
			if !results[0].Matched {
				t.Errorf("field %q: expected match, got reason: %s", tc.field, results[0].Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Multiple conditions
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_MultipleConditionsAllMatch(t *testing.T) {
	a := baseAssignment("a1", 1)
	a = withCondition(a, "sku", OpEquals, "SKU-APPLE")
	a = withCondition(a, "variety", OpEquals, "FUJI")
	a = withCondition(a, "caliber", OpIn, "75", "80", "85")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected all conditions to match, got reason: %s", results[0].Reason)
	}
	if len(results[0].ConditionResults) != 3 {
		t.Errorf("expected 3 ConditionResults, got %d", len(results[0].ConditionResults))
	}
}

func TestEvaluateAssignments_MultipleConditionsOneFailsAllReturnedForTrace(t *testing.T) {
	a := baseAssignment("a1", 1)
	a = withCondition(a, "sku", OpEquals, "SKU-APPLE")
	a = withCondition(a, "variety", OpEquals, "GALA") // esto falla
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match when one condition fails")
	}
	if len(results[0].ConditionResults) != 2 {
		t.Errorf("expected 2 ConditionResults for traceability, got %d", len(results[0].ConditionResults))
	}
	if results[0].ConditionResults[0].Matched != true {
		t.Error("first condition (sku) should have matched")
	}
	if results[0].ConditionResults[1].Matched != false {
		t.Error("second condition (variety) should not have matched")
	}
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_TagMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "tags.lote", OpEquals, "LOTE-2026")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected tag match, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_TagNoMatch(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "tags.lote", OpEquals, "LOTE-OTHER")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected tag no match")
	}
}

func TestEvaluateAssignments_TagExistsAbsent(t *testing.T) {
	// tag "nonexistent" no está en el box → exists debe fallar
	a := withCondition(baseAssignment("a1", 1), "tags.nonexistent", OpExists)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected exists to fail for absent tag")
	}
}

func TestEvaluateAssignments_TagNotExistsAbsent(t *testing.T) {
	// tag "nonexistent" no está en el box → not_exists debe pasar
	a := withCondition(baseAssignment("a1", 1), "tags.nonexistent", OpNotExists)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected not_exists to pass for absent tag, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_TagPresentButEmpty(t *testing.T) {
	// tag "present" existe pero tiene valor vacío → exists debe fallar
	a := withCondition(baseAssignment("a1", 1), "tags.present", OpExists)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected exists to fail for tag with empty value")
	}
}

// ---------------------------------------------------------------------------
// Disabled assignment
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_DisabledDoesNotMatch(t *testing.T) {
	a := baseAssignment("a1", 1)
	a.Enabled = false
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected disabled assignment to not match")
	}
	if results[0].Reason != "disabled" {
		t.Errorf("expected reason 'disabled', got %q", results[0].Reason)
	}
	if results[0].ConditionResults != nil {
		t.Error("expected nil ConditionResults for disabled assignment")
	}
}

// ---------------------------------------------------------------------------
// Temporal validity
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_ValidFrom_Before(t *testing.T) {
	a := baseAssignment("a1", 1)
	a.ValidFrom = fixedTime().Add(1 * time.Hour) // comienza en el futuro
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match: evalTime is before ValidFrom")
	}
	if results[0].Reason != "out_of_time_range" {
		t.Errorf("expected reason 'out_of_time_range', got %q", results[0].Reason)
	}
}

func TestEvaluateAssignments_ValidTo_After(t *testing.T) {
	a := baseAssignment("a1", 1)
	a.ValidTo = fixedTime().Add(-1 * time.Hour) // ya expiró
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match: evalTime is after ValidTo")
	}
	if results[0].Reason != "out_of_time_range" {
		t.Errorf("expected reason 'out_of_time_range', got %q", results[0].Reason)
	}
}

func TestEvaluateAssignments_WithinRange(t *testing.T) {
	a := baseAssignment("a1", 1)
	a.ValidFrom = fixedTime().Add(-1 * time.Hour)
	a.ValidTo = fixedTime().Add(1 * time.Hour)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match within time range, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_ZeroValidFrom_NoLowerBound(t *testing.T) {
	a := baseAssignment("a1", 1)
	// ValidFrom zero = sin límite inferior
	a.ValidTo = fixedTime().Add(1 * time.Hour)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match with zero ValidFrom, got reason: %s", results[0].Reason)
	}
}

func TestEvaluateAssignments_ZeroValidTo_NoUpperBound(t *testing.T) {
	a := baseAssignment("a1", 1)
	a.ValidFrom = fixedTime().Add(-1 * time.Hour)
	// ValidTo zero = sin límite superior
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected match with zero ValidTo, got reason: %s", results[0].Reason)
	}
}

// ---------------------------------------------------------------------------
// No conditions = universal match
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_NoConditions_MatchesAll(t *testing.T) {
	a := baseAssignment("a1", 1) // sin condiciones
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if !results[0].Matched {
		t.Errorf("expected universal match for assignment without conditions, got reason: %s", results[0].Reason)
	}
}

// ---------------------------------------------------------------------------
// Priority ordering
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_OrderedByPriorityDescending(t *testing.T) {
	set := AssignmentSet{Assignments: []Assignment{
		baseAssignment("low", 1),
		baseAssignment("high", 10),
		baseAssignment("mid", 5),
	}}
	results := EvaluateAssignments(baseBox(), set, fixedTime())
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Assignment.Priority != 10 {
		t.Errorf("expected first result priority 10, got %d", results[0].Assignment.Priority)
	}
	if results[1].Assignment.Priority != 5 {
		t.Errorf("expected second result priority 5, got %d", results[1].Assignment.Priority)
	}
	if results[2].Assignment.Priority != 1 {
		t.Errorf("expected third result priority 1, got %d", results[2].Assignment.Priority)
	}
}

// ---------------------------------------------------------------------------
// evalTime zero uses time.Now()
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_ZeroEvalTimeUsesNow(t *testing.T) {
	a := baseAssignment("a1", 1)
	// ValidFrom/ValidTo alrededor del momento actual
	a.ValidFrom = time.Now().Add(-1 * time.Hour)
	a.ValidTo = time.Now().Add(1 * time.Hour)
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), time.Time{})
	if !results[0].Matched {
		t.Errorf("expected match when evalTime is zero (should use time.Now()), got reason: %s", results[0].Reason)
	}
}

// ---------------------------------------------------------------------------
// Unknown field: no panic, no match
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_UnknownFieldNoPanic(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "nonexistent_field", OpEquals, "value")
	// No debe hacer panic
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if results[0].Matched {
		t.Error("expected no match for unknown field")
	}
	if len(results[0].ConditionResults) == 0 {
		t.Fatal("expected ConditionResults to be populated")
	}
	ce := results[0].ConditionResults[0]
	if ce.Matched {
		t.Error("ConditionEvaluation for unknown field should be Matched=false")
	}
	if ce.Actual != "" {
		t.Errorf("expected empty Actual for unknown field, got %q", ce.Actual)
	}
}

// ---------------------------------------------------------------------------
// Mixed results: matched and not-matched both returned
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_ReturnsBothMatchedAndNotMatched(t *testing.T) {
	a1 := withCondition(baseAssignment("match", 2), "sku", OpEquals, "SKU-APPLE")
	a2 := withCondition(baseAssignment("nomatch", 1), "sku", OpEquals, "SKU-OTHER")
	set := AssignmentSet{Assignments: []Assignment{a1, a2}}
	results := EvaluateAssignments(baseBox(), set, fixedTime())
	if len(results) != 2 {
		t.Fatalf("expected 2 results (matched + not-matched), got %d", len(results))
	}
	// First result (priority 2) should be matched
	if !results[0].Matched {
		t.Errorf("expected first result to be matched")
	}
	// Second result (priority 1) should not be matched
	if results[1].Matched {
		t.Errorf("expected second result to not be matched")
	}
}

// ---------------------------------------------------------------------------
// ConditionResults populated
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_ConditionResultsPopulated(t *testing.T) {
	a := withCondition(baseAssignment("a1", 1), "sku", OpEquals, "SKU-APPLE")
	results := EvaluateAssignments(baseBox(), singleAssignmentSet(a), fixedTime())
	if len(results[0].ConditionResults) != 1 {
		t.Fatalf("expected 1 ConditionResult, got %d", len(results[0].ConditionResults))
	}
	ce := results[0].ConditionResults[0]
	if ce.Field != "sku" {
		t.Errorf("expected Field=sku, got %q", ce.Field)
	}
	if ce.Operator != OpEquals {
		t.Errorf("expected Operator=equals, got %q", ce.Operator)
	}
	if len(ce.Expected) != 1 || ce.Expected[0] != "SKU-APPLE" {
		t.Errorf("expected Expected=[SKU-APPLE], got %v", ce.Expected)
	}
	if ce.Actual != "SKU-APPLE" {
		t.Errorf("expected Actual=SKU-APPLE, got %q", ce.Actual)
	}
	if !ce.Matched {
		t.Error("expected Matched=true")
	}
}

// ---------------------------------------------------------------------------
// Empty AssignmentSet
// ---------------------------------------------------------------------------

func TestEvaluateAssignments_EmptySetReturnsEmpty(t *testing.T) {
	results := EvaluateAssignments(baseBox(), AssignmentSet{}, fixedTime())
	if len(results) != 0 {
		t.Errorf("expected empty results for empty AssignmentSet, got %d", len(results))
	}
}
