package sorter

import (
	"slices"
	"sort"
	"strings"
	"time"
)

// EvaluateAssignments evalúa todas las asignaciones del set contra el BoxContext.
//
// Reglas de evaluación:
//   - Si evalTime es zero value, se usa time.Now().
//   - Se retornan todas las evaluaciones (matched y no-matched) para trazabilidad.
//   - Los resultados se ordenan por Priority descendente (mayor prioridad primero).
//   - Una asignación sin condiciones es match universal si está enabled y vigente.
func EvaluateAssignments(box BoxContext, set AssignmentSet, evalTime time.Time) []AssignmentMatch {
	if evalTime.IsZero() {
		evalTime = time.Now()
	}

	results := make([]AssignmentMatch, 0, len(set.Assignments))
	for _, a := range set.Assignments {
		results = append(results, evaluateAssignment(box, a, evalTime))
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Assignment.Priority > results[j].Assignment.Priority
	})

	return results
}

// evaluateAssignment evalúa una asignación individual contra el BoxContext.
func evaluateAssignment(box BoxContext, a Assignment, evalTime time.Time) AssignmentMatch {
	if !a.Enabled {
		return AssignmentMatch{
			Assignment: a,
			Matched:    false,
			Reason:     "disabled",
		}
	}

	if !isWithinTimeRange(evalTime, a.ValidFrom, a.ValidTo) {
		return AssignmentMatch{
			Assignment: a,
			Matched:    false,
			Reason:     "out_of_time_range",
		}
	}

	// Sin condiciones: match universal.
	if len(a.Conditions) == 0 {
		return AssignmentMatch{
			Assignment: a,
			Matched:    true,
			Reason:     "matched",
		}
	}

	condResults := make([]ConditionEvaluation, 0, len(a.Conditions))
	for _, cond := range a.Conditions {
		ce := evaluateCondition(box, cond)
		condResults = append(condResults, ce)
		if !ce.Matched {
			// Falló una condición: el resto se evalúa igualmente para trazabilidad.
			// Marcamos el resultado final como no-matched al terminar.
		}
	}

	allMatched := true
	for _, ce := range condResults {
		if !ce.Matched {
			allMatched = false
			break
		}
	}

	reason := "matched"
	if !allMatched {
		reason = "condition_not_matched"
	}

	return AssignmentMatch{
		Assignment:       a,
		Matched:          allMatched,
		Reason:           reason,
		ConditionResults: condResults,
	}
}

// isWithinTimeRange verifica si t está dentro de [from, to].
// Zero value en from = sin límite inferior.
// Zero value en to   = sin límite superior.
func isWithinTimeRange(t, from, to time.Time) bool {
	if !from.IsZero() && t.Before(from) {
		return false
	}
	if !to.IsZero() && t.After(to) {
		return false
	}
	return true
}

// evaluateCondition evalúa una MatchCondition contra el BoxContext.
func evaluateCondition(box BoxContext, cond MatchCondition) ConditionEvaluation {
	actual, fieldKnown := resolveField(box, cond.Field)

	ce := ConditionEvaluation{
		Field:    cond.Field,
		Operator: cond.Operator,
		Expected: cond.Values,
		Actual:   actual,
	}

	if !fieldKnown {
		ce.Matched = false
		return ce
	}

	ce.Matched = applyOperator(cond.Operator, actual, cond.Values)
	return ce
}

// resolveField extrae el valor de un campo del BoxContext por nombre.
// Devuelve (valor, true) si el campo existe en el schema.
// Devuelve ("", false) si el campo no existe.
//
// Campos directos: box_id, qr, sku, variety, caliber, category, client,
// package_type, destination, flow, source, read_status, quality_status.
// Tags: campo con prefijo "tags." (ej: "tags.lote" → box.Tags["lote"]).
func resolveField(box BoxContext, field string) (string, bool) {
	if strings.HasPrefix(field, "tags.") {
		key := strings.TrimPrefix(field, "tags.")
		val := box.Tags[key] // zero value si ausente
		return val, true     // el campo "tags.*" siempre es conocido
	}

	switch field {
	case "box_id":
		return box.BoxID, true
	case "qr":
		return box.QR, true
	case "sku":
		return box.SKU, true
	case "variety":
		return box.Variety, true
	case "caliber":
		return box.Caliber, true
	case "category":
		return box.Category, true
	case "client":
		return box.Client, true
	case "package_type":
		return box.PackageType, true
	case "destination":
		return box.Destination, true
	case "flow":
		return box.Flow, true
	case "source":
		return box.Source, true
	case "read_status":
		return box.ReadStatus, true
	case "quality_status":
		return box.QualityStatus, true
	default:
		return "", false
	}
}

// applyOperator aplica un MatchOperator sobre un valor actual y los valores esperados.
func applyOperator(op MatchOperator, actual string, values []string) bool {
	switch op {
	case OpEquals:
		if len(values) == 0 {
			return false
		}
		return actual == values[0]

	case OpNotEquals:
		if len(values) == 0 {
			return false
		}
		return actual != values[0]

	case OpIn:
		return slices.Contains(values, actual)

	case OpNotIn:
		return !slices.Contains(values, actual)

	case OpExists:
		return actual != ""

	case OpNotExists:
		return actual == ""

	default:
		return false
	}
}

// assignmentTargetIDs returns the effective target IDs for an assignment.
// If TargetIDs has non-empty elements, returns those (filtered, deduped).
// Otherwise falls back to TargetID (single legacy field).
// Returns nil if both are empty.
func assignmentTargetIDs(a Assignment) []string {
	if len(a.TargetIDs) > 0 {
		seen := make(map[string]struct{}, len(a.TargetIDs))
		out := make([]string, 0, len(a.TargetIDs))
		for _, id := range a.TargetIDs {
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
		if len(out) > 0 {
			return out
		}
	}
	if a.TargetID != "" {
		return []string{a.TargetID}
	}
	return nil
}
