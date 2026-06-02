# Iteración 02 — Foundation: dantrack-sorter

**Autor:** Arquitectura DANTRACK  
**Fecha:** 2026-06-01  
**Estado:** Completada  
**Módulo:** `github.com/dantrack/dantrack-sorter`  
**Package:** `package sorter`

---

## 1. Resumen Ejecutivo

La Iteración 02 implementó la base de la librería `dantrack-sorter`: modelos de dominio, errores sentinel y validaciones. No se implementó lógica de decisión. La librería compila, pasa `go vet` y tiene 33 tests unitarios en verde.

---

## 2. Objetivo de la Iteración 02

Establecer los cimientos del motor de decisión:

- Modelos de dominio claros, tipados y sin dependencias externas.
- Errores sentinel distinguibles mediante `errors.Is`.
- Validaciones de configuración y de request antes de procesar decisiones.
- Tests unitarios con stdlib pura que sirvan de contrato para iteraciones siguientes.

---

## 3. Alcance Implementado

| Área | Implementado |
|---|---|
| Modelos de dominio | Sí — `types.go` |
| Errores sentinel | Sí — `errors.go` |
| Validación de config | Sí — `ValidateConfig`, `ValidatePresorterConfig`, `ValidateSorterConfig` |
| Validación de request | Sí — `ValidatePresorterRequest`, `ValidateSorterRequest` |
| Validación de assignments | Sí — operadores, rangos temporales, TargetType |
| Tests | Sí — 33 tests, stdlib pura |
| Motor de decisión | **No** — Iteración 04 |
| Assignment matching | **No** — Iteración 03 |
| Balanceo | **No** — Iteración 05 |

---

## 4. Archivos Creados/Modificados

| Archivo | Acción | Descripción |
|---|---|---|
| `types.go` | Creado | Todos los modelos de dominio |
| `errors.go` | Creado | Errores sentinel |
| `validation.go` | Creado | Funciones de validación |
| `validation_test.go` | Creado | 33 tests unitarios |
| `README.md` | Actualizado | Estado real de la librería |
| `go.mod` | Sin cambios | `go 1.23` |
| `docs/ITERATION_01_PROPOSAL.md` | Actualizado | Nota de implementación agregada |

---

## 5. Modelos Principales Agregados

### Tipos de control

| Tipo | Descripción |
|---|---|
| `Action` | Acción resultante: `route`, `reject`, `recirculate`, `error`, `noop` |
| `Rule` | Regla aplicada, tipada con constantes (no string libre) |
| `BalanceStrategy` | Estrategia de balanceo: `least_loaded`, `round_robin`, `weighted` |
| `MatchOperator` | Operador de condición: `equals`, `not_equals`, `in`, `not_in`, `exists`, `not_exists` |
| `TargetType` | Tipo de destino: `exit`, `park` |

### Modelos de entrada

| Tipo | Descripción |
|---|---|
| `BoxContext` | Snapshot de la caja para evaluación (16 campos) |
| `Assignment` | Mapeo de condiciones a destino, con vigencia temporal |
| `AssignmentSet` | Conjunto de asignaciones para evaluación |
| `MatchCondition` | Condición individual: campo, operador, valores |
| `PresorterRequest` | Entrada para decisión de presorter (con `TraceID`, `EvalTime`) |
| `SorterRequest` | Entrada para decisión de sorter (con `TraceID`, `EvalTime`) |

### Modelos de configuración

| Tipo | Descripción |
|---|---|
| `EngineConfig` | Configuración raíz: agrupa `PresorterConfig` y `SorterConfig` |
| `PresorterConfig` | Parks, estrategia de balanceo, parque default, flag de error control |
| `ParkConfig` | ID, habilitado, error control, prioridad, carga máxima |
| `ParkState` | Estado operacional de un parque (entregado por el orquestador) |
| `SorterConfig` | SorterID, exits, exit default |
| `SorterExit` | ID, habilitado, prioridad, capacidad |
| `SorterExitState` | Estado operacional de una salida (entregado por el orquestador) |

### Modelos de decisión y trazabilidad

| Tipo | Descripción |
|---|---|
| `PresorterDecision` | Decisión de presorter con `DecisionTrace` embebido |
| `SorterDecision` | Decisión de sorter con `DecisionTrace` embebido |
| `DecisionTrace` | Rastro completo: `TraceID`, `CorrelationID`, `RuleApplied`, `Reason`, candidatos |
| `CandidateEvaluation` | Por qué un candidato fue elegido/descartado, con condiciones evaluadas |
| `ConditionEvaluation` | Resultado de evaluar una condición: campo, operador, esperado, actual, match |

### Constantes `Rule` definidas

| Constante | Valor |
|---|---|
| `RuleNone` | `""` |
| `RulePresorterErrorControlPreferred` | `"presorter_error_control_preferred"` |
| `RulePresorterLeastLoaded` | `"presorter_least_loaded"` |
| `RulePresorterRoundRobin` | `"presorter_round_robin"` |
| `RulePresorterFallbackDefaultPark` | `"presorter_fallback_default_park"` |
| `RulePresorterRejectNoAvailablePark` | `"presorter_reject_no_available_park"` |
| `RuleSorterAssignmentMatched` | `"sorter_assignment_matched"` |
| `RuleSorterFallbackDefaultExit` | `"sorter_fallback_default_exit"` |
| `RuleSorterRejectNoMatch` | `"sorter_reject_no_match"` |
| `RuleSorterRejectNoAvailableExit` | `"sorter_reject_no_available_exit"` |

---

## 6. Errores Sentinel Definidos

```go
ErrInvalidConfig        // configuración inválida; no permite operar
ErrInvalidRequest       // request inválido; campos incorrectos o inconsistentes
ErrMissingRequiredField // campo obligatorio ausente
ErrNoAssignmentMatch    // ninguna asignación aplica (sin decisión posible)
ErrNoAvailableExit      // sin salidas configuradas (ni default)
ErrNoAvailablePark      // sin parques configurados (ni default)
ErrAmbiguousAssignment  // múltiples asignaciones misma prioridad, distintos targets
ErrContextCanceled      // context.Context cancelado
```

Todos son distinguibles mediante `errors.Is`. Ninguno es un string genérico.

**Regla de uso:** los errores Go se reservan para problemas que impiden producir una decisión (config rota, request inválido). `ActionReject` es una decisión válida — no produce `error`.

---

## 7. Validaciones Implementadas

### `ValidateConfig(cfg EngineConfig) error`
- Al menos `Presorter` o `Sorter` debe estar configurado.
- Delega en `ValidatePresorterConfig` y `ValidateSorterConfig` cuando corresponde.

### `ValidatePresorterConfig(cfg PresorterConfig) error`
- Al menos un parque definido.
- `ParkID` no vacío en cada parque.
- No `ParkID` duplicados.
- `BalanceStrategy` reconocida (si se especifica).
- `DefaultParkID` referencia un `ParkID` existente (si se especifica).

### `ValidateSorterConfig(cfg SorterConfig) error`
- `SorterID` no vacío.
- Al menos una salida definida.
- `ExitID` no vacío en cada salida.
- No `ExitID` duplicados.
- `DefaultExitID` referencia un `ExitID` existente (si se especifica).

### `ValidatePresorterRequest(req PresorterRequest) error`
- `BoxContext` mínimo: `BoxID` o `QR` (al menos uno).
- `ParkID` no vacío en cada `ParkState`.
- No `ParkID` duplicados en `ParkStates`.

### `ValidateSorterRequest(req SorterRequest) error`
- `BoxContext` mínimo: `BoxID` o `QR`.
- `AssignmentSet` puede estar vacío; si trae datos, cada `Assignment` se valida.
- `ExitID` no vacío en cada `SorterExitState`.
- No `ExitID` duplicados en `ExitStates`.

### Validación de `Assignment`
- `ID` no vacío.
- `Priority >= 0`.
- `TargetType` reconocido (`exit` o `park`).
- `TargetID` no vacío.
- Si `ValidFrom` y `ValidTo` están definidos: `ValidTo` no puede ser anterior a `ValidFrom`.
- Cada `MatchCondition` valida:
  - `Field` no vacío.
  - `Operator` reconocido.
  - `Values` requeridos para operadores que no sean `exists` / `not_exists`.

---

## 8. Tests Implementados

Archivo: `validation_test.go` — 33 tests, stdlib pura, sin dependencias externas.

| Grupo | Tests |
|---|---|
| `ValidateConfig` | Config solo presorter, solo sorter, ambos, ninguno |
| `ValidatePresorterConfig` | Válida, sin parques, ParkID vacío, duplicado, strategy inválida, DefaultParkID inexistente, DefaultParkID válido |
| `ValidateSorterConfig` | Válida, SorterID vacío, sin exits, ExitID vacío, duplicado, DefaultExitID inexistente |
| `ValidatePresorterRequest` | Válido, BoxID+QR vacíos, solo BoxID, solo QR, ParkState duplicado |
| `ValidateSorterRequest` | Válido, BoxID+QR vacíos, assignments vacíos, ExitState duplicado |
| `Assignment` | Operador inválido, rango temporal inválido, ID vacío, `exists` sin valores (válido), `equals` sin valores (inválido) |
| Errores | Todos los sentinels son distinguibles entre sí via `errors.Is` |

---

## 9. Comandos Ejecutados

```bash
go fmt ./...
go vet ./...
go test ./... -v
```

---

## 10. Resultado de Tests

```
go fmt ./...   OK  (formateó errors.go, types.go, validation.go, validation_test.go)
go vet ./...   OK  (sin warnings)
go test ./...  OK  33/33 PASS  (0.644s)
```

---

## 11. Decisiones Técnicas Aplicadas

| Decisión | Detalle |
|---|---|
| Sin UUID interno | `TraceID` y `CorrelationID` vienen del orquestador; la librería los copia a la decisión, nunca los genera |
| `EvalTime time.Time` | Value type (no puntero); zero value → `time.Now()` en iteraciones futuras; en tests siempre pasar valor explícito |
| `Rule` tipado | Tipo `Rule string` con constantes; nunca string libre; permite filtrar auditoría programáticamente |
| `ActionReject` ≠ error | Decisión válida; `error` Go solo para config/request inválido o ambigüedad irresoluble |
| `DecisionTrace` embebido | Trazabilidad unificada en `PresorterDecision.Trace` y `SorterDecision.Trace` |
| `slices.Contains` para enums | Idiomático Go 1.21+; sin reflection, sin bucles manuales |
| `validateBoxContext` flexible | Acepta si hay `BoxID` **o** `QR`; ambos vacíos es error |
| `exists`/`not_exists` sin valores | Operadores de existencia no requieren `Values`; validación correcta |
| `Priority >= 0` | `Priority == 0` es válido como menor prioridad; negativo es error |
| `AssignmentSet` vacío es válido | La ausencia de asignaciones producirá `ActionReject` en iteraciones futuras, no un error |

---

## 12. Qué NO Se Implementó Todavía

| Pendiente | Iteración |
|---|---|
| Resolución de campos de `BoxContext` por nombre string | 03 |
| Evaluación de `MatchCondition` contra `BoxContext` | 03 |
| `EvaluateAssignments` | 03 |
| Filtrado por vigencia temporal (`EvalTime`) | 03 |
| `engine.go` / `NewEngine` | 04 |
| `ResolveSorter` | 04 |
| `ResolvePresorter` | 05 |
| Estrategias de balanceo (`balance.go`) | 05 |
| `examples/` | 06 |
| `docs/ARCHITECTURE.md` | 06 |

---

## 13. Riesgos y Pendientes

| # | Riesgo/Pendiente | Acción |
|---|---|---|
| 1 | `EvalTime` zero value: el comportamiento futuro (`time.Now()`) debe ser explícito y documentado en la función que lo use | Documentar en Iteración 03 cuando se use por primera vez |
| 2 | `validateBoxContext` acepta con solo `BoxID` o solo `QR`: algunos packings pueden requerir ambos | Configurable si se necesita en Iteración 03/04 |
| 3 | `AssignmentSet` sin asignaciones produce comportamiento no definido todavía | Definir en Iteración 03: vacío → `ActionReject` con `RuleSorterRejectNoMatch` |
| 4 | `Priority < 0` rechazado, pero `Priority == 0` aceptado: asegurarse que la lógica de ordenación en Iteración 03 lo maneje correctamente | Verificar en Iteración 03 |
| 5 | Las constantes `Rule` y `Action` son extensibles; si se agrega una nueva constante, los switch exhaustivos deben actualizarse | Agregar test de exhaustividad en Iteración 04 |

---

## 14. Siguiente Paso: Iteración 03 — Assignment Matching

Implementar `assignment.go`:

- Función `EvaluateAssignments(box BoxContext, set AssignmentSet, evalTime time.Time) []AssignmentMatch`.
- Resolver campos de `BoxContext` por nombre string (switch idiomático, sin reflect).
- Evaluar cada `MatchCondition` según su operador.
- Filtrar por `Enabled == true`.
- Filtrar por vigencia temporal: `ValidFrom` ≤ `evalTime` ≤ `ValidTo` (tolerando zero values).
- Ordenar matches por `Priority` descendente.
- Retornar lista de `AssignmentMatch` con resultado y razón de match/no-match.
- Tests table-driven exhaustivos para todas las combinaciones de operador/campo/tag.

---

## Next Agent Instructions

### Iteración 03: Assignment Matching

**Implementar `assignment.go`** con la lógica de evaluación de asignaciones.

**Modelos ya disponibles** (no volver a crear):
- `BoxContext`, `Assignment`, `AssignmentSet`, `MatchCondition`, `MatchOperator`
- `ConditionEvaluation` (ya definida en `types.go` — usar para resultado de condición)

**Nuevo tipo a agregar en `types.go`:**

```go
// AssignmentMatch es el resultado de evaluar una asignación contra un BoxContext.
type AssignmentMatch struct {
    Assignment Assignment
    Matched    bool
    Reason     string
    // ConditionResults puede ser nil si Matched == false por razón de vigencia o habilitación.
    ConditionResults []ConditionEvaluation
}
```

**Función pública principal:**

```go
// EvaluateAssignments evalúa todas las asignaciones vigentes y habilitadas contra el BoxContext.
// Retorna la lista completa de evaluaciones ordenada por Priority descendente.
// evalTime controla la vigencia temporal; si es zero, usa time.Now().
func EvaluateAssignments(box BoxContext, set AssignmentSet, evalTime time.Time) []AssignmentMatch
```

**Reglas de implementación:**

1. Filtrar `Enabled == false` → `Matched: false, Reason: "disabled"`.
2. Filtrar por vigencia: si `evalTime` está fuera de `[ValidFrom, ValidTo]` → `Matched: false, Reason: "out of time range"`. Respetar zero values: zero `ValidFrom` = sin límite inferior; zero `ValidTo` = sin límite superior.
3. Evaluar cada `MatchCondition` en orden; si alguna falla → `Matched: false`.
4. Resolver campos de `BoxContext` por `MatchCondition.Field` con switch idiomático (no reflection). Campos soportados: `"sku"`, `"variety"`, `"caliber"`, `"category"`, `"client"`, `"package_type"`, `"destination"`, `"flow"`, `"source"`, `"read_status"`, `"quality_status"`. Tags: prefijo `"tags."` → buscar en `BoxContext.Tags`.
5. Ordenar resultado por `Priority` descendente (mayor prioridad primero).
6. Retornar todas las evaluaciones (matched y no-matched) para trazabilidad completa.

**Operadores a implementar:**

| Operador | Semántica |
|---|---|
| `equals` | `actual == values[0]` |
| `not_equals` | `actual != values[0]` |
| `in` | `actual` está en `values` |
| `not_in` | `actual` no está en `values` |
| `exists` | campo no vacío (o tag presente) |
| `not_exists` | campo vacío (o tag ausente) |

**Tests en `assignment_test.go`:**
- Table-driven para cada operador con casos positivos y negativos.
- Test de campo directo (`sku`, `variety`, etc.) y campo de tag (`tags.lote`).
- Test de vigencia temporal: antes, durante, después del rango.
- Test de assignment deshabilitado.
- Test de asignación sin condiciones (match universal).
- Test de ordenación por prioridad.
- Test de `evalTime` zero value: debe comportarse como `time.Now()` (test con margen temporal).

**No implementar en Iteración 03:**
- `ResolveSorter`, `ResolvePresorter`, balanceo, engine.

**Criterio de éxito:** `go vet ./...` limpio, `go test ./...` verde, cobertura > 90% en `assignment.go`.
