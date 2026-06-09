package sorter

import "time"

// ---------------------------------------------------------------------------
// Match helpers — shortcuts para construir MatchCondition
// ---------------------------------------------------------------------------

// Match crea una MatchCondition genérica con el operador y valores indicados.
func Match(field string, operator MatchOperator, values ...string) MatchCondition {
	return MatchCondition{Field: field, Operator: operator, Values: values}
}

// MatchEquals crea una MatchCondition con OpEquals y un solo valor.
func MatchEquals(field string, value string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpEquals, Values: []string{value}}
}

// MatchNotEquals crea una MatchCondition con OpNotEquals y un solo valor.
func MatchNotEquals(field string, value string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpNotEquals, Values: []string{value}}
}

// MatchIn crea una MatchCondition con OpIn y los valores indicados.
func MatchIn(field string, values ...string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpIn, Values: values}
}

// MatchNotIn crea una MatchCondition con OpNotIn y los valores indicados.
func MatchNotIn(field string, values ...string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpNotIn, Values: values}
}

// MatchExists crea una MatchCondition con OpExists. No requiere valores.
func MatchExists(field string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpExists}
}

// MatchNotExists crea una MatchCondition con OpNotExists. No requiere valores.
func MatchNotExists(field string) MatchCondition {
	return MatchCondition{Field: field, Operator: OpNotExists}
}

// ---------------------------------------------------------------------------
// Exit state helpers
// ---------------------------------------------------------------------------

// ExitStates agrupa múltiples SorterExitState en un slice.
func ExitStates(states ...SorterExitState) []SorterExitState {
	return states
}

// ExitAvailable crea un SorterExitState disponible con la carga indicada.
func ExitAvailable(exitID string, currentLoad int) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, CurrentLoad: currentLoad}
}

// ExitFull crea un SorterExitState marcado como lleno.
func ExitFull(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: true, Full: true}
}

// ExitBlocked crea un SorterExitState marcado como bloqueado.
func ExitBlocked(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: false, Blocked: true}
}

// ExitMaintenance crea un SorterExitState en mantenimiento.
func ExitMaintenance(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: false, Maintenance: true}
}

// ExitError crea un SorterExitState con error.
func ExitError(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: false, HasError: true}
}

// ExitUnavailable crea un SorterExitState no disponible sin razón específica.
func ExitUnavailable(exitID string) SorterExitState {
	return SorterExitState{ExitID: exitID, Available: false}
}

// ---------------------------------------------------------------------------
// Park state helpers
// ---------------------------------------------------------------------------

// ParkStates agrupa múltiples ParkState en un slice.
func ParkStates(states ...ParkState) []ParkState {
	return states
}

// ParkAvailable crea un ParkState disponible con la carga indicada.
func ParkAvailable(parkID string, currentLoad int) ParkState {
	return ParkState{ParkID: parkID, Available: true, CurrentLoad: currentLoad}
}

// ParkAvailableWithWindow crea un ParkState disponible con carga actual y carga acumulada por ventana.
func ParkAvailableWithWindow(parkID string, currentLoad int, accumulatedLoad int) ParkState {
	return ParkState{
		ParkID:          parkID,
		Available:       true,
		CurrentLoad:     currentLoad,
		AccumulatedLoad: accumulatedLoad,
	}
}

// ParkFull crea un ParkState marcado como lleno.
func ParkFull(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: true, Full: true}
}

// ParkBlocked crea un ParkState marcado como bloqueado.
func ParkBlocked(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: false, Blocked: true}
}

// ParkMaintenance crea un ParkState en mantenimiento.
func ParkMaintenance(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: false, Maintenance: true}
}

// ParkError crea un ParkState con error.
func ParkError(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: false, HasError: true}
}

// ParkUnavailable crea un ParkState no disponible sin razón específica.
func ParkUnavailable(parkID string) ParkState {
	return ParkState{ParkID: parkID, Available: false}
}

// ---------------------------------------------------------------------------
// Box helpers
// ---------------------------------------------------------------------------

// BoxBuilder construye un BoxContext mediante encadenamiento de métodos.
// Es inmutable: cada método retorna una copia nueva del builder.
type BoxBuilder struct {
	box BoxContext
}

// NewBox crea un BoxBuilder con el BoxID indicado.
func NewBox(boxID string) BoxBuilder {
	return BoxBuilder{box: BoxContext{BoxID: boxID}}
}

// ErrorBox crea un BoxContext para cajas con error de lectura.
// El BoxContext resultante será detectado como error por el motor de presorter.
func ErrorBox(boxID string, readStatus string) BoxContext {
	return BoxContext{BoxID: boxID, ReadStatus: readStatus}
}

func (b BoxBuilder) QR(value string) BoxBuilder {
	b.box.QR = value
	return b
}

func (b BoxBuilder) SKU(value string) BoxBuilder {
	b.box.SKU = value
	return b
}

func (b BoxBuilder) Variety(value string) BoxBuilder {
	b.box.Variety = value
	return b
}

func (b BoxBuilder) Caliber(value string) BoxBuilder {
	b.box.Caliber = value
	return b
}

func (b BoxBuilder) Category(value string) BoxBuilder {
	b.box.Category = value
	return b
}

func (b BoxBuilder) Client(value string) BoxBuilder {
	b.box.Client = value
	return b
}

func (b BoxBuilder) PackageType(value string) BoxBuilder {
	b.box.PackageType = value
	return b
}

func (b BoxBuilder) Destination(value string) BoxBuilder {
	b.box.Destination = value
	return b
}

func (b BoxBuilder) Flow(value string) BoxBuilder {
	b.box.Flow = value
	return b
}

func (b BoxBuilder) Source(value string) BoxBuilder {
	b.box.Source = value
	return b
}

// Tag agrega un tag al BoxContext. Copia el mapa existente para mantener inmutabilidad.
func (b BoxBuilder) Tag(key, value string) BoxBuilder {
	tags := make(map[string]string, len(b.box.Tags)+1)
	for k, v := range b.box.Tags {
		tags[k] = v
	}
	tags[key] = value
	b.box.Tags = tags
	return b
}

func (b BoxBuilder) ReadStatus(value string) BoxBuilder {
	b.box.ReadStatus = value
	return b
}

func (b BoxBuilder) QualityStatus(value string) BoxBuilder {
	b.box.QualityStatus = value
	return b
}

// ErrorFlag agrega un flag de error al BoxContext. Copia el slice para mantener inmutabilidad.
func (b BoxBuilder) ErrorFlag(value string) BoxBuilder {
	flags := make([]string, len(b.box.ErrorFlags)+1)
	copy(flags, b.box.ErrorFlags)
	flags[len(b.box.ErrorFlags)] = value
	b.box.ErrorFlags = flags
	return b
}

func (b BoxBuilder) Timestamp(value time.Time) BoxBuilder {
	b.box.Timestamp = value
	return b
}

// Build retorna el BoxContext construido.
func (b BoxBuilder) Build() BoxContext {
	return b.box
}

// ---------------------------------------------------------------------------
// Assignment builder
// ---------------------------------------------------------------------------

// AssignmentBuilder construye un Assignment mediante encadenamiento de métodos.
// Es inmutable: cada método retorna una copia nueva del builder.
// Default: Enabled = true.
type AssignmentBuilder struct {
	a Assignment
}

// NewAssignment crea un AssignmentBuilder con ID y prioridad.
// El assignment está Enabled por defecto.
func NewAssignment(id string, priority int) AssignmentBuilder {
	return AssignmentBuilder{a: Assignment{ID: id, Priority: priority, Enabled: true}}
}

func (b AssignmentBuilder) Enabled(enabled bool) AssignmentBuilder {
	b.a.Enabled = enabled
	return b
}

// Disabled setea Enabled = false.
func (b AssignmentBuilder) Disabled() AssignmentBuilder {
	b.a.Enabled = false
	return b
}

// Match agrega una MatchCondition genérica.
func (b AssignmentBuilder) Match(field string, operator MatchOperator, values ...string) AssignmentBuilder {
	b.a.Conditions = appendCondition(b.a.Conditions, MatchCondition{Field: field, Operator: operator, Values: values})
	return b
}

// MatchIn agrega una condición OpIn.
func (b AssignmentBuilder) MatchIn(field string, values ...string) AssignmentBuilder {
	b.a.Conditions = appendCondition(b.a.Conditions, MatchCondition{Field: field, Operator: OpIn, Values: values})
	return b
}

// MatchEquals agrega una condición OpEquals.
func (b AssignmentBuilder) MatchEquals(field string, value string) AssignmentBuilder {
	b.a.Conditions = appendCondition(b.a.Conditions, MatchCondition{Field: field, Operator: OpEquals, Values: []string{value}})
	return b
}

// MatchExists agrega una condición OpExists.
func (b AssignmentBuilder) MatchExists(field string) AssignmentBuilder {
	b.a.Conditions = appendCondition(b.a.Conditions, MatchCondition{Field: field, Operator: OpExists})
	return b
}

// ToExit setea el destino como un exit único (TargetIDs con un elemento).
func (b AssignmentBuilder) ToExit(exitID string) AssignmentBuilder {
	b.a.TargetType = TargetTypeExit
	b.a.TargetIDs = []string{exitID}
	return b
}

// ToExits setea múltiples exits como destino.
func (b AssignmentBuilder) ToExits(exitIDs ...string) AssignmentBuilder {
	b.a.TargetType = TargetTypeExit
	b.a.TargetIDs = exitIDs
	return b
}

// ToPark setea el destino como un park único (TargetIDs con un elemento).
func (b AssignmentBuilder) ToPark(parkID string) AssignmentBuilder {
	b.a.TargetType = TargetTypePark
	b.a.TargetIDs = []string{parkID}
	return b
}

// ToParks setea múltiples parks como destino.
func (b AssignmentBuilder) ToParks(parkIDs ...string) AssignmentBuilder {
	b.a.TargetType = TargetTypePark
	b.a.TargetIDs = parkIDs
	return b
}

// ValidBetween setea la vigencia temporal de la asignación.
func (b AssignmentBuilder) ValidBetween(from, to time.Time) AssignmentBuilder {
	b.a.ValidFrom = from
	b.a.ValidTo = to
	return b
}

// Metadata agrega un par clave-valor al mapa de metadata. Copia el mapa para mantener inmutabilidad.
func (b AssignmentBuilder) Metadata(key, value string) AssignmentBuilder {
	meta := make(map[string]string, len(b.a.Metadata)+1)
	for k, v := range b.a.Metadata {
		meta[k] = v
	}
	meta[key] = value
	b.a.Metadata = meta
	return b
}

// Build retorna el Assignment construido.
func (b AssignmentBuilder) Build() Assignment {
	return b.a
}

// appendCondition copia el slice y agrega una condición para mantener inmutabilidad del builder.
func appendCondition(conditions []MatchCondition, mc MatchCondition) []MatchCondition {
	newConds := make([]MatchCondition, len(conditions)+1)
	copy(newConds, conditions)
	newConds[len(conditions)] = mc
	return newConds
}
