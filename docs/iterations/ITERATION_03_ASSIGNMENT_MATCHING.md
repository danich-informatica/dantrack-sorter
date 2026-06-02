# Iteración 03 — Assignment Matching: dantrack-sorter

**Autor:** Arquitectura DANTRACK  
**Fecha:** 2026-06-01  
**Estado:** Completada  
**Módulo:** `github.com/dantrack/dantrack-sorter`  
**Package:** `package sorter`

---

## 1. Resumen Ejecutivo

La Iteración 03 implementó el motor de evaluación de asignaciones. La función `EvaluateAssignments` recibe un `BoxContext`, un `AssignmentSet` y un instante de evaluación, y devuelve la lista completa de evaluaciones —matched y no-matched— ordenada por prioridad descendente. La librería sigue sin lógica de decisión final: evalúa pero no decide.

70/70 tests PASS. `go vet` limpio. Sin dependencias externas.

---

## 2. Objetivo de la Iteración 03

- Implementar `EvaluateAssignments` con soporte para 6 operadores, 13 campos directos y tags dinámicos.
- Garantizar trazabilidad completa por condición.
- Cubrir los edge cases: disabled, vigencia temporal, campo desconocido, sin condiciones, evalTime zero.

---

## 3. Alcance Implementado

| Área | Implementado |
|---|---|
| `EvaluateAssignments` (función pública) | Sí |
| Resolución de campos de `BoxContext` por nombre | Sí — switch idiomático, sin reflection |
| Resolución de tags (`tags.*`) | Sí |
| Operador `equals` | Sí |
| Operador `not_equals` | Sí |
| Operador `in` | Sí |
| Operador `not_in` | Sí |
| Operador `exists` | Sí |
| Operador `not_exists` | Sí |
| Filtro por `Enabled` | Sí |
| Filtro por vigencia temporal (`ValidFrom`/`ValidTo`) | Sí |
| Soporte para zero values en `ValidFrom`/`ValidTo` | Sí |
| Soporte para `evalTime` zero value (→ `time.Now()`) | Sí |
| Match universal (sin condiciones) | Sí |
| Ordenación por `Priority` descendente | Sí |
| `ConditionEvaluation` para trazabilidad por condición | Sí |
| Campo desconocido: no panic, no match | Sí |
| Tipo `AssignmentMatch` | Sí — agregado en `types.go` |
| Motor de decisión (`ResolveSorter`) | **No** — Iteración 04 |
| Balanceo | **No** — Iteración 05 |

---

## 4. Archivos Creados/Modificados

| Archivo | Acción | Descripción |
|---|---|---|
| `assignment.go` | Creado | `EvaluateAssignments`, `evaluateAssignment`, `resolveField`, `applyOperator`, `isWithinTimeRange` |
| `assignment_test.go` | Creado | 37 tests (25 de assignment + 13 subtests de campos directos) |
| `types.go` | Modificado | `AssignmentMatch` agregado |
| `README.md` | Actualizado | Iteración 03 marcada como completada |

---

## 5. API Agregada

```go
// EvaluateAssignments evalúa todas las asignaciones del set contra el BoxContext.
// Si evalTime es zero value, usa time.Now().
// Retorna todas las evaluaciones (matched y no-matched) ordenadas por Priority desc.
func EvaluateAssignments(box BoxContext, set AssignmentSet, evalTime time.Time) []AssignmentMatch

// AssignmentMatch es el resultado de evaluar una asignación contra un BoxContext.
type AssignmentMatch struct {
    Assignment       Assignment
    Matched          bool
    Reason           string
    ConditionResults []ConditionEvaluation // nil si disabled o fuera de vigencia
}
```

Funciones internas (no exportadas):

| Función | Propósito |
|---|---|
| `evaluateAssignment` | Evalúa una asignación individual |
| `isWithinTimeRange` | Verifica vigencia temporal con zero-value tolerance |
| `resolveField` | Extrae valor de `BoxContext` por nombre de campo |
| `applyOperator` | Aplica un operador sobre un valor actual y valores esperados |

---

## 6. Reglas de Matching Implementadas

### Orden de evaluación (cortocircuito por razón, condiciones completas para traza)

1. `Enabled == false` → `Matched: false, Reason: "disabled"` (sin evaluar condiciones).
2. Fuera de `[ValidFrom, ValidTo]` → `Matched: false, Reason: "out_of_time_range"`.
3. Sin condiciones → `Matched: true, Reason: "matched"` (match universal).
4. Evaluar cada condición. Si alguna falla → `Matched: false, Reason: "condition_not_matched"`. **Todas las condiciones se evalúan igual para trazabilidad completa.**

### Resolución de campos

| Campo en `MatchCondition.Field` | Mapeado a |
|---|---|
| `box_id` | `BoxContext.BoxID` |
| `qr` | `BoxContext.QR` |
| `sku` | `BoxContext.SKU` |
| `variety` | `BoxContext.Variety` |
| `caliber` | `BoxContext.Caliber` |
| `category` | `BoxContext.Category` |
| `client` | `BoxContext.Client` |
| `package_type` | `BoxContext.PackageType` |
| `destination` | `BoxContext.Destination` |
| `flow` | `BoxContext.Flow` |
| `source` | `BoxContext.Source` |
| `read_status` | `BoxContext.ReadStatus` |
| `quality_status` | `BoxContext.QualityStatus` |
| `tags.<key>` | `BoxContext.Tags[key]` |
| Desconocido | `("", false)` → condición falla sin panic |

### Operadores

| Operador | Semántica |
|---|---|
| `equals` | `actual == values[0]` |
| `not_equals` | `actual != values[0]` |
| `in` | `slices.Contains(values, actual)` |
| `not_in` | `!slices.Contains(values, actual)` |
| `exists` | `actual != ""` |
| `not_exists` | `actual == ""` |

**Tag `exists`/`not_exists`:** si el tag está ausente en el mapa, `actual` es `""` (zero value de Go); `not_exists` pasa, `exists` falla. Un tag con valor explícitamente vacío (`""`) también falla `exists`.

---

## 7. Tests Implementados

Archivo: `assignment_test.go` — 37 funciones de test (25 + 13 subtests), stdlib pura.

| Grupo | Tests |
|---|---|
| `equals` | Positivo, negativo |
| `not_equals` | Positivo, negativo |
| `in` | Positivo, negativo |
| `not_in` | Positivo, negativo |
| `exists` | Positivo (campo lleno), negativo (campo vacío) |
| `not_exists` | Positivo (campo vacío), negativo (campo lleno) |
| Campo directo | `sku` match + verificación de `Actual` |
| Todos los campos directos | 13 subtests (box_id → quality_status) |
| Múltiples condiciones | Todas match, una falla → todas retornadas para traza |
| Tags | Match, no-match, ausente+exists, ausente+not_exists, presente-vacío+exists |
| Disabled | No match, reason `"disabled"`, ConditionResults nil |
| Vigencia temporal | Before, after, within, zero ValidFrom, zero ValidTo |
| Sin condiciones | Match universal |
| Prioridad | Ordenación descendente con 3 asignaciones |
| `evalTime` zero | Usa `time.Now()` (test con margen de ±1h) |
| Campo desconocido | No panic, Matched=false, Actual="" |
| Mixed results | Matched y no-matched ambos retornados |
| `ConditionResults` | Campos Field, Operator, Expected, Actual, Matched verificados |
| Set vacío | Retorna slice vacío |

---

## 8. Comandos Ejecutados

```bash
go fmt ./...
go vet ./...
go test ./... -v
```

---

## 9. Resultado de Tests

```
go fmt ./...   OK  (formateó assignment.go, assignment_test.go, types.go)
go vet ./...   OK  (sin warnings)
go test ./...  OK  70/70 PASS  (0.800s)
```

Iteración 02: 33 tests. Iteración 03 agrega 37. Total acumulado: 70.

---

## 10. Decisiones Técnicas

| Decisión | Detalle |
|---|---|
| Todas las condiciones se evalúan siempre | No hay early return por condición fallida; se retorna `ConditionResults` completo para trazabilidad |
| `tags.*` siempre es campo conocido | `resolveField` retorna `(val, true)` aunque el tag no exista; `val=""` → `exists` falla naturalmente |
| `sort.SliceStable` para prioridad | Preserva el orden original entre asignaciones de igual prioridad (determinismo) |
| `evalTime.IsZero()` → `time.Now()` | Al inicio de `EvaluateAssignments`, una sola vez, no en cada sub-función |
| Campo desconocido: `(_, false)` | `evaluateCondition` recibe `fieldKnown`; si `false`, `Matched=false` sin panic |
| Sin dependencias externas | Solo `slices`, `sort`, `strings`, `time` (stdlib) |

---

## 11. Qué NO Se Implementó

| Pendiente | Iteración |
|---|---|
| `engine.go` / `NewEngine` | 04 |
| `ResolveSorter` | 04 |
| Selección de primera asignación matched para decisión | 04 |
| `ResolvePresorter` | 05 |
| `balance.go` / estrategias de balanceo | 05 |
| `examples/` | 06 |
| `docs/ARCHITECTURE.md` | 06 |

---

## 12. Riesgos y Pendientes

| # | Riesgo/Pendiente | Acción |
|---|---|---|
| 1 | `EvaluateAssignments` retorna todos (matched + no-matched). El engine en Iteración 04 debe filtrar por `Matched == true` y tomar el primero de mayor prioridad | Documentar en Iteración 04 |
| 2 | Ambigüedad: si dos asignaciones tienen misma prioridad y ambas matchean → `ErrAmbiguousAssignment` o first-wins. La política es configurable | Implementar en Iteración 04 al construir `ResolveSorter` |
| 3 | `tags.*` retorna `("", true)` para tags ausentes. Esto es correcto semánticamente pero puede sorprender: un tag ausente no es lo mismo que un tag con valor vacío desde el punto de vista del operador de negocio | Documentado en sección de reglas; revisable si hay feedback |
| 4 | `sort.SliceStable` tiene costo O(n log n). Si `AssignmentSet` tiene > 10.000 asignaciones, puede impactar. En escenarios industriales típicos (< 200) no es relevante | Benchmark si se detecta problema |

---

## 13. Siguiente Paso: Iteración 04 — Sorter Decision Engine

Implementar `engine.go` y `sorter.go`:

- `NewEngine(cfg EngineConfig) (*Engine, error)` — crea el motor con config validada.
- `(*Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)` — decide a qué salida enviar la caja.
- Internamente usa `EvaluateAssignments` para obtener candidates.
- Verifica disponibilidad de exits usando `SorterExitState`.
- Aplica fallback a `DefaultExitID` si el exit del match no está disponible.
- Si no hay match ni default disponible: `ActionReject` con `RuleSorterRejectNoMatch`.
- Política de ambigüedad: configurable (error / first-wins).
- `DecisionTrace` completamente poblado.
- `context.Context` controlado: si cancelado, devolver `ErrContextCanceled`.

---

## Next Agent Instructions

### Iteración 04: Sorter Decision Engine

**Objetivo:** implementar `engine.go` y `sorter.go`. El presorter NO va en esta iteración.

**Archivos a crear:**
- `engine.go` — `Engine` struct, `NewEngine`, `EngineConfig` (ya existe en types.go)
- `sorter.go` — `(*Engine).ResolveSorter`
- `engine_test.go` — tests de integración del sorter
- `sorter_test.go` — tests unitarios de la lógica de decisión

**Dependencias ya disponibles:** `EvaluateAssignments`, todos los tipos, todos los errores.

**Firma de `NewEngine`:**

```go
func NewEngine(cfg EngineConfig) (*Engine, error)
// Llama ValidateConfig internamente. Devuelve ErrInvalidConfig si falla.
// Inicializa estado interno (round-robin counter si aplica).
```

**Firma de `ResolveSorter`:**

```go
func (e *Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)
```

**Flujo interno de `ResolveSorter`:**

1. Verificar `ctx.Err()` al inicio → `ErrContextCanceled` si cancelado.
2. `ValidateSorterRequest(req)` → `ErrInvalidRequest` si inválido.
3. Resolver `evalTime`: si zero, usar `time.Now()`.
4. `EvaluateAssignments(req.Box, req.Assignments, evalTime)`.
5. Filtrar por `Matched == true`.
6. Si ninguno: intentar `DefaultExitID` si configurado y disponible → `ActionReject` con `RuleSorterFallbackDefaultExit`. Si no: `ActionReject` con `RuleSorterRejectNoMatch`.
7. Si hay matches: tomar el de mayor prioridad. Si ambigüedad (misma prioridad, distinto exit) → según política de `EngineConfig`.
8. Verificar disponibilidad del exit en `req.ExitStates`.
9. Si no disponible: intentar siguiente match por prioridad (cascada). Si todos bloqueados: ir a `DefaultExitID` o `ActionReject`.
10. Construir `SorterDecision` con `DecisionTrace` completo: `TraceID`, `CorrelationID`, `RuleApplied`, `Reason`, `CandidateEvaluations`.

**Política de ambigüedad (campo en `SorterConfig`):**

```go
type AmbiguityPolicy string

const (
    AmbiguityPolicyError     AmbiguityPolicy = "error"      // default
    AmbiguityPolicyFirstWins AmbiguityPolicy = "first_wins"
)
```

Agregar `AmbiguityPolicy AmbiguityPolicy` a `SorterConfig`.

**Tests obligatorios en `engine_test.go` / `sorter_test.go`:**
- Match directo: una asignación matchea, exit disponible.
- Fallback a DefaultExit: exit del match bloqueado.
- Reject sin match.
- Reject sin default.
- Ambigüedad con política error.
- Ambigüedad con política first-wins.
- Context cancelado devuelve error.
- Request inválido devuelve error.
- `DecisionTrace` completamente poblado.
- `TraceID` y `CorrelationID` copiados del request.

**No implementar en Iteración 04:**
- `ResolvePresorter`, balanceo, presorter config, examples.

**Criterio de éxito:** `go vet ./...` limpio, `go test ./...` verde, todos los paths de decisión del sorter cubiertos.
