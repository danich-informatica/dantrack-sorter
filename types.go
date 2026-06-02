package sorter

import "time"

// ---------------------------------------------------------------------------
// Action — resultado de una decisión de ruteo
// ---------------------------------------------------------------------------

// Action representa la acción que el orquestador debe ejecutar.
// ActionReject es una decisión válida; no es un error Go.
type Action string

const (
	// ActionRoute indica que la caja debe ser enviada al destino seleccionado.
	ActionRoute Action = "route"
	// ActionReject indica que la caja no tiene destino válido y debe ser rechazada.
	// Es una decisión operacional legítima, no un error técnico.
	ActionReject Action = "reject"
	// ActionRecirculate indica que la caja debe volver a recircular.
	ActionRecirculate Action = "recirculate"
	// ActionError indica que ocurrió un error interno al intentar decidir.
	// Diferente de ActionReject: aquí la librería no pudo producir una decisión.
	ActionError Action = "error"
	// ActionNoop indica que no se requiere acción (uso diagnóstico/testing).
	ActionNoop Action = "noop"
)

// ---------------------------------------------------------------------------
// Rule — regla aplicada para llegar a una decisión
// ---------------------------------------------------------------------------

// Rule identifica qué lógica produjo la decisión.
// Es un tipo tipado, no un string libre, para permitir filtrado en auditoría.
type Rule string

const (
	RuleNone Rule = ""

	// Presorter
	RulePresorterErrorControlPreferred Rule = "presorter_error_control_preferred"
	RulePresorterLeastLoaded           Rule = "presorter_least_loaded"
	RulePresorterRoundRobin            Rule = "presorter_round_robin"
	RulePresorterWeighted              Rule = "presorter_weighted"
	RulePresorterFallbackDefaultPark   Rule = "presorter_fallback_default_park"
	RulePresorterRejectNoAvailablePark Rule = "presorter_reject_no_available_park"

	// Sorter
	RuleSorterAssignmentMatched     Rule = "sorter_assignment_matched"
	RuleSorterFallbackDefaultExit   Rule = "sorter_fallback_default_exit"
	RuleSorterRejectNoMatch         Rule = "sorter_reject_no_match"
	RuleSorterRejectNoAvailableExit Rule = "sorter_reject_no_available_exit"
)

// ---------------------------------------------------------------------------
// BalanceStrategy — estrategia de balanceo entre destinos válidos
// ---------------------------------------------------------------------------

// BalanceStrategy define cómo elegir entre múltiples destinos elegibles.
type BalanceStrategy string

const (
	// BalanceLeastLoaded elige el destino con menor carga actual.
	// Es la estrategia recomendada para primera integración real.
	BalanceLeastLoaded BalanceStrategy = "least_loaded"
	// BalanceRoundRobin rota entre destinos. Requiere estado interno en el Engine.
	// El contador se reinicia si el proceso reinicia.
	BalanceRoundRobin BalanceStrategy = "round_robin"
	// BalanceWeighted elige según pesos configurados.
	BalanceWeighted BalanceStrategy = "weighted"
)

// ---------------------------------------------------------------------------
// AmbiguityPolicy — política para resolver asignaciones ambiguas de igual prioridad
// ---------------------------------------------------------------------------

// AmbiguityPolicy define cómo actuar cuando múltiples asignaciones de igual prioridad
// apuntan a distintos exits.
type AmbiguityPolicy string

const (
	// AmbiguityPolicyError devuelve ErrAmbiguousAssignment. Es el comportamiento por defecto.
	AmbiguityPolicyError AmbiguityPolicy = "error"
	// AmbiguityPolicyFirstWins elige la primera asignación según el orden estable de EvaluateAssignments.
	AmbiguityPolicyFirstWins AmbiguityPolicy = "first_wins"
)

// ---------------------------------------------------------------------------
// MatchOperator — operador de condición para matching de asignaciones
// ---------------------------------------------------------------------------

// MatchOperator define cómo comparar un campo del BoxContext contra un valor.
type MatchOperator string

const (
	OpEquals    MatchOperator = "equals"
	OpNotEquals MatchOperator = "not_equals"
	OpIn        MatchOperator = "in"
	OpNotIn     MatchOperator = "not_in"
	OpExists    MatchOperator = "exists"
	OpNotExists MatchOperator = "not_exists"
)

// ---------------------------------------------------------------------------
// TargetType — tipo de destino al que apunta una asignación
// ---------------------------------------------------------------------------

// TargetType identifica si el destino es una salida de sorter o un parque de presorter.
type TargetType string

const (
	TargetTypeExit TargetType = "exit"
	TargetTypePark TargetType = "park"
)

// ---------------------------------------------------------------------------
// BoxContext — contexto de una caja para decisión de routing
// ---------------------------------------------------------------------------

// BoxContext representa la información de una caja en el momento de una decisión.
// No es una entidad de persistencia; es un snapshot del contexto para evaluación.
type BoxContext struct {
	BoxID         string
	QR            string
	SKU           string
	Variety       string
	Caliber       string
	Category      string
	Client        string
	PackageType   string
	Destination   string
	Flow          string
	Source        string
	Tags          map[string]string
	ReadStatus    string
	QualityStatus string
	ErrorFlags    []string
	Timestamp     time.Time
}

// ---------------------------------------------------------------------------
// Trazabilidad
// ---------------------------------------------------------------------------

// ConditionEvaluation registra el resultado de evaluar una condición de asignación.
type ConditionEvaluation struct {
	Field    string
	Operator MatchOperator
	Expected []string
	Actual   string
	Matched  bool
}

// CandidateEvaluation registra por qué un candidato fue elegido o descartado.
type CandidateEvaluation struct {
	TargetType     TargetType
	TargetID       string
	Eligible       bool
	RejectedReason string
	Rule           Rule
	AssignmentID   string
	Conditions     []ConditionEvaluation
}

// DecisionTrace contiene el rastro completo de cómo se llegó a una decisión.
// Es embebido en PresorterDecision y SorterDecision.
type DecisionTrace struct {
	TraceID              string
	CorrelationID        string
	RuleApplied          Rule
	Reason               string
	DiagnosticMessage    string
	CandidateEvaluations []CandidateEvaluation
}

// ---------------------------------------------------------------------------
// Assignment — asignación que mapea condiciones de caja a un destino
// ---------------------------------------------------------------------------

// MatchCondition define una condición que debe cumplir un campo del BoxContext.
type MatchCondition struct {
	// Field es el campo del BoxContext a evaluar.
	// Campos directos: "sku", "variety", "caliber", "category", "client",
	// "package_type", "destination", "flow", "source", "read_status", "quality_status".
	// Tags: "tags.<nombre_del_tag>" (ej: "tags.lote").
	Field    string
	Operator MatchOperator
	Values   []string
}

// Assignment define el mapeo de una caja a un destino según condiciones.
// La librería recibe assignments ya cargados; no accede a base de datos.
type Assignment struct {
	ID         string
	Enabled    bool
	Priority   int
	Conditions []MatchCondition
	TargetType TargetType
	TargetID   string
	ValidFrom  time.Time
	ValidTo    time.Time
	Metadata   map[string]string
}

// AssignmentSet agrupa un conjunto de asignaciones para evaluación.
type AssignmentSet struct {
	Assignments []Assignment
}

// AssignmentMatch es el resultado de evaluar una asignación contra un BoxContext.
// Se retorna tanto para matches como no-matches para permitir trazabilidad completa.
type AssignmentMatch struct {
	Assignment Assignment
	Matched    bool
	Reason     string
	// ConditionResults es nil cuando Matched == false por disabled o vigencia temporal.
	ConditionResults []ConditionEvaluation
}

// ---------------------------------------------------------------------------
// Presorter
// ---------------------------------------------------------------------------

// ParkConfig define la configuración estática de un parque.
type ParkConfig struct {
	ParkID          string
	Enabled         bool
	HasErrorControl bool
	Priority        int
	MaxLoad         int
	Weight          int
	Tags            map[string]string
}

// ParkState describe el estado operacional actual de un parque.
// Lo entrega el orquestador; la librería no lo calcula.
type ParkState struct {
	ParkID      string
	Available   bool
	Blocked     bool
	Full        bool
	HasError    bool
	Maintenance bool
	CurrentLoad int
}

// PresorterConfig define la configuración del motor de presorter.
type PresorterConfig struct {
	Parks           []ParkConfig
	BalanceStrategy BalanceStrategy
	// DefaultParkID es el parque de fallback final si ningún candidato está disponible.
	// Debe referenciar un ParkID existente en Parks.
	DefaultParkID string
	// ErrorControlFlag define qué ErrorFlag de BoxContext indica que la caja
	// requiere control de errores (ej: "requires_error_control").
	ErrorControlFlag string
}

// PresorterRequest es la entrada para una decisión de presorter.
type PresorterRequest struct {
	// TraceID es entregado por el orquestador para correlación en logs.
	// La librería lo copia a PresorterDecision. Si viene vacío, se deja vacío.
	TraceID string
	// CorrelationID es un ID de correlación alternativo (ej: ID de sesión, lote).
	CorrelationID string
	Box           BoxContext
	ParkStates    []ParkState
	// EvalTime es el instante de evaluación temporal para asignaciones y vigencias.
	// Si es zero value, la lógica futura usará time.Now().
	// En tests, siempre pasar un valor explícito para determinismo.
	EvalTime time.Time
}

// PresorterDecision es la decisión de presorter con trazabilidad completa.
type PresorterDecision struct {
	ParkID                string
	Action                Action
	FallbackUsed          bool
	ErrorControlRequired  bool
	ErrorControlAvailable bool
	BalanceStrategy       BalanceStrategy
	Trace                 DecisionTrace
	EvalTime              time.Time
}

// ---------------------------------------------------------------------------
// Sorter
// ---------------------------------------------------------------------------

// SorterExit define la configuración estática de una salida del sorter.
type SorterExit struct {
	ExitID   string
	Enabled  bool
	Priority int
	Capacity int
	Tags     map[string]string
}

// SorterExitState describe el estado operacional actual de una salida.
// Lo entrega el orquestador; la librería no lo calcula.
type SorterExitState struct {
	ExitID      string
	Available   bool
	Blocked     bool
	Full        bool
	HasError    bool
	Maintenance bool
	CurrentLoad int
}

// SorterConfig define la configuración del motor de sorter.
type SorterConfig struct {
	SorterID string
	Exits    []SorterExit
	// DefaultExitID es la salida de fallback cuando el exit del match no está disponible.
	// Debe referenciar un ExitID existente en Exits.
	DefaultExitID string
	// RejectExitID es la salida física de rechazo cuando no hay match de asignación.
	// Debe referenciar un ExitID existente en Exits. Opcional.
	RejectExitID string
	// AmbiguityPolicy define cómo actuar ante asignaciones ambiguas de igual prioridad.
	// Vacío equivale a AmbiguityPolicyError.
	AmbiguityPolicy AmbiguityPolicy
}

// SorterRequest es la entrada para una decisión de sorter.
type SorterRequest struct {
	// TraceID es entregado por el orquestador para correlación en logs.
	TraceID string
	// CorrelationID es un ID de correlación alternativo.
	CorrelationID string
	Box           BoxContext
	Assignments   AssignmentSet
	ExitStates    []SorterExitState
	// EvalTime es el instante de evaluación. Si es zero value, la lógica futura usará time.Now().
	EvalTime time.Time
}

// SorterDecision es la decisión de sorter con trazabilidad completa.
type SorterDecision struct {
	SorterID     string
	ExitID       string
	Action       Action
	AssignmentID string
	FallbackUsed bool
	Rejected     bool
	Trace        DecisionTrace
	EvalTime     time.Time
}

// ---------------------------------------------------------------------------
// Engine config
// ---------------------------------------------------------------------------

// EngineConfig agrupa la configuración del motor de decisión.
// Al menos uno de Presorter o Sorter debe estar configurado.
type EngineConfig struct {
	Presorter *PresorterConfig
	Sorter    *SorterConfig
}
