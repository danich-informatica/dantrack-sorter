# Public API Review — dantrack-sorter v0.1.0-rc1

**Fecha:** 2026-06-02  
**Revisor:** Arquitectura DANTRACK

---

## 1. Resumen Ejecutivo

Se revisó la API pública completa de `dantrack-sorter`. La API es consistente,
clara, y está lista para ser marcada como estable en v0.1.0. No se encontraron
problemas que requieran cambios breaking antes del release.

Se identifican algunas mejoras opcionales para v0.2.0 que no afectan la estabilidad
actual.

---

## 2. API Pública Actual

### Tipos

| Tipo | Categoría | Descripción |
|---|---|---|
| `Action` | Enum (string) | Resultado de decisión: route, reject, recirculate, error, noop |
| `Rule` | Enum (string) | Regla que produjo la decisión |
| `BalanceStrategy` | Enum (string) | Estrategia de balanceo |
| `AmbiguityPolicy` | Enum (string) | Política ante ambigüedad |
| `MatchOperator` | Enum (string) | Operador de condición |
| `TargetType` | Enum (string) | Tipo de destino (exit/park) |
| `BoxContext` | Struct | Snapshot de caja para evaluación |
| `ConditionEvaluation` | Struct | Resultado de evaluar una condición |
| `CandidateEvaluation` | Struct | Resultado de evaluar un candidato |
| `DecisionTrace` | Struct | Trazabilidad completa |
| `MatchCondition` | Struct | Condición de asignación |
| `Assignment` | Struct | Regla de mapeo caja→destino |
| `AssignmentSet` | Struct | Conjunto de asignaciones |
| `AssignmentMatch` | Struct | Resultado de evaluación de asignación |
| `ParkConfig` | Struct | Configuración de un park |
| `ParkState` | Struct | Estado operacional de un park |
| `PresorterConfig` | Struct | Configuración del presorter |
| `PresorterRequest` | Struct | Entrada para decisión presorter |
| `PresorterDecision` | Struct | Salida de decisión presorter |
| `SorterExit` | Struct | Configuración de una salida |
| `SorterExitState` | Struct | Estado operacional de una salida |
| `SorterConfig` | Struct | Configuración del sorter |
| `SorterRequest` | Struct | Entrada para decisión sorter |
| `SorterDecision` | Struct | Salida de decisión sorter |
| `EngineConfig` | Struct | Agrupador de configuración |
| `Engine` | Struct (opaco) | Motor de decisión |

### Constantes Públicas

| Constante | Tipo | Valor |
|---|---|---|
| `ActionRoute` | Action | "route" |
| `ActionReject` | Action | "reject" |
| `ActionRecirculate` | Action | "recirculate" |
| `ActionError` | Action | "error" |
| `ActionNoop` | Action | "noop" |
| `RuleNone` | Rule | "" |
| `RulePresorterErrorControlPreferred` | Rule | "presorter_error_control_preferred" |
| `RulePresorterLeastLoaded` | Rule | "presorter_least_loaded" |
| `RulePresorterRoundRobin` | Rule | "presorter_round_robin" |
| `RulePresorterWeighted` | Rule | "presorter_weighted" |
| `RulePresorterFallbackDefaultPark` | Rule | "presorter_fallback_default_park" |
| `RulePresorterRejectNoAvailablePark` | Rule | "presorter_reject_no_available_park" |
| `RuleSorterAssignmentMatched` | Rule | "sorter_assignment_matched" |
| `RuleSorterFallbackDefaultExit` | Rule | "sorter_fallback_default_exit" |
| `RuleSorterRejectNoMatch` | Rule | "sorter_reject_no_match" |
| `RuleSorterRejectNoAvailableExit` | Rule | "sorter_reject_no_available_exit" |
| `BalanceLeastLoaded` | BalanceStrategy | "least_loaded" |
| `BalanceRoundRobin` | BalanceStrategy | "round_robin" |
| `BalanceWeighted` | BalanceStrategy | "weighted" |
| `AmbiguityPolicyError` | AmbiguityPolicy | "error" |
| `AmbiguityPolicyFirstWins` | AmbiguityPolicy | "first_wins" |
| `OpEquals` | MatchOperator | "equals" |
| `OpNotEquals` | MatchOperator | "not_equals" |
| `OpIn` | MatchOperator | "in" |
| `OpNotIn` | MatchOperator | "not_in" |
| `OpExists` | MatchOperator | "exists" |
| `OpNotExists` | MatchOperator | "not_exists" |
| `TargetTypeExit` | TargetType | "exit" |
| `TargetTypePark` | TargetType | "park" |

### Errores Sentinel

| Error | Compatible con errors.Is |
|---|---|
| `ErrInvalidConfig` | Sí |
| `ErrInvalidRequest` | Sí |
| `ErrMissingRequiredField` | Sí |
| `ErrNoAssignmentMatch` | Sí |
| `ErrNoAvailableExit` | Sí |
| `ErrNoAvailablePark` | Sí |
| `ErrAmbiguousAssignment` | Sí |
| `ErrContextCanceled` | Sí |

### Funciones Públicas

| Función | Firma |
|---|---|
| `NewEngine` | `func NewEngine(cfg EngineConfig) (*Engine, error)` |
| `EvaluateAssignments` | `func EvaluateAssignments(box BoxContext, set AssignmentSet, evalTime time.Time) []AssignmentMatch` |
| `ValidateConfig` | `func ValidateConfig(cfg EngineConfig) error` |
| `ValidatePresorterConfig` | `func ValidatePresorterConfig(cfg PresorterConfig) error` |
| `ValidateSorterConfig` | `func ValidateSorterConfig(cfg SorterConfig) error` |
| `ValidatePresorterRequest` | `func ValidatePresorterRequest(req PresorterRequest) error` |
| `ValidateSorterRequest` | `func ValidateSorterRequest(req SorterRequest) error` |

### Métodos Públicos

| Método | Firma |
|---|---|
| `Engine.ResolveSorter` | `func (e *Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)` |
| `Engine.ResolvePresorter` | `func (e *Engine) ResolvePresorter(ctx context.Context, req PresorterRequest) (PresorterDecision, error)` |

---

## 3. APIs Estables para v0.1.0

Las siguientes se consideran estables y no deben cambiar en minor versions:

- `NewEngine(EngineConfig) (*Engine, error)` — constructor.
- `Engine.ResolveSorter(ctx, SorterRequest) (SorterDecision, error)` — decisión sorter.
- `Engine.ResolvePresorter(ctx, PresorterRequest) (PresorterDecision, error)` — decisión presorter.
- `EvaluateAssignments(box, set, evalTime) []AssignmentMatch` — evaluación pura.
- Todos los errores sentinel con `errors.Is`.
- Todos los tipos de request/decision structs (campos pueden crecer, no reducirse).
- Todas las constantes Action, Rule, BalanceStrategy, MatchOperator, TargetType.
- Validate* functions.

---

## 4. APIs que Podrían Cambiar en v0.2.0

| API | Cambio posible | Razón |
|---|---|---|
| `ActionRecirculate`, `ActionError`, `ActionNoop` | Podrían removerse si no se usan | Definidas pero nunca producidas por el motor actual |
| `ErrNoAssignmentMatch`, `ErrNoAvailableExit`, `ErrNoAvailablePark` | Podrían removerse | Nunca son devueltos actualmente; el motor usa ActionReject |
| `PresorterConfig.ErrorControlFlag` | Podría removerse | Campo definido pero no usado en la implementación actual |
| `Assignment.Metadata` | Podría cambiar | Campo pass-through; semántica no definida |
| `ParkConfig.Tags`, `SorterExit.Tags` | Podrían cambiar | Campos pass-through sin uso actual |

---

## 5. Riesgos de Compatibilidad

| Riesgo | Impacto | Mitigación |
|---|---|---|
| Campos struct sin uso (`ErrorControlFlag`, Tags, Metadata) | Bajo | No rompen; son forward-compatible |
| Actions/Errors nunca producidos | Bajo | El orquestador puede ignorarlos; documentar como reservados |
| `EvaluateAssignments` expuesta públicamente | Bajo | Útil para debugging; difícil de cambiar sin breaking |
| `AssignmentSet` como struct wrapper | Bajo | Podría haber sido `[]Assignment` directamente; pero es forward-compatible |
| Round-robin counter sin persistencia | Bajo | Documentado; no afecta correctitud |

---

## 6. Recomendaciones

### Para v0.1.0 (aplicar ahora si son menores)

1. **Documentar constantes reservadas**: `ActionRecirculate`, `ActionError`, `ActionNoop` están
   definidas pero nunca producidas. Agregar comentario "reserved for future use".
2. **`PresorterConfig.ErrorControlFlag`**: campo nunca leído por la implementación.
   Documentar como reservado o remover. Recomendación: mantener con comentario "reserved".

### Para v0.2.0 (no aplicar ahora)

1. Considerar remover errores sentinel no producidos (`ErrNoAssignmentMatch`, etc.) o
   documentarlos como deprecated.
2. Considerar hacer `AssignmentSet` un type alias si no se agregan métodos.
3. Evaluar si `Assignment.Metadata` necesita semántica definida.

---

## 7. Cambios Aplicados

| Cambio | Archivo | Tipo |
|---|---|---|
| Ninguno | — | — |

No se requieren cambios de API para v0.1.0-rc1. La API es consistente y funcional.

---

## 8. Cambios NO Aplicados

| Cambio | Razón para no aplicar |
|---|---|
| Remover `ActionRecirculate`/`ActionError`/`ActionNoop` | Son forward-compatible; útiles para extensión futura |
| Remover errores sentinel no usados | Son forward-compatible; orquestador puede usarlos para custom logic |
| Remover `PresorterConfig.ErrorControlFlag` | Puede ser útil en v0.2.0; no molesta |
| Cambiar `AssignmentSet` a `[]Assignment` | Breaking change innecesario |
| Agregar métodos a Engine (String, Config, etc.) | No requerido; YAGNI |

---

## 9. Conclusión

La API pública de `dantrack-sorter` está lista para v0.1.0-rc1:

- Nombres claros y consistentes entre Sorter y Presorter.
- Todos los tipos tipados (Rule, Action, BalanceStrategy, etc.).
- Errores compatibles con `errors.Is`.
- `ActionReject` como decisión válida (no error Go).
- TraceID/CorrelationID sin generación automática.
- EvalTime como `time.Time`.
- Sin dependencias externas.
- Campos forward-compatible (structs pueden crecer sin breaking).
