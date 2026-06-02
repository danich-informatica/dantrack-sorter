# Catastro de Estado Actual — dantrack-sorter

**Fecha de auditoría:** 2026-06-02  
**Auditor:** Arquitectura DANTRACK (agente)  
**Commit base:** HEAD (sin VCS detectado)

---

## 1. Resumen Ejecutivo

El repositorio está **parcialmente sano con una inconsistencia crítica**: la documentación de la Iteración 05 existe y declara implementación completa, pero **los archivos de código y tests de la Iteración 05 NO existen**. La documentación fue generada sin que el código correspondiente se haya creado en el repositorio.

| Aspecto | Estado |
|---|---|
| Código Iter 01–04 | Sano, compilando, 105 tests PASS |
| Documentación Iter 05 | **Contaminada** — describe código inexistente |
| README | Desactualizado — dice Iter 05 "Próxima" pero hay doc que dice "Completada" |
| Cobertura | 95.7% sobre código existente |
| Dependencias externas | Ninguna (solo stdlib) |
| Principios de diseño | Todos respetados en código existente |
| Listo para siguiente iteración | **NO** — se requiere limpieza documental primero |

---

## 2. Estado General del Repositorio

- **Package:** `package sorter` — correcto.
- **Module:** `github.com/dantrack/dantrack-sorter` — correcto.
- **Go version:** 1.23 — correcto.
- **Dependencias externas:** ninguna — correcto.
- **Compilación:** `go build ./...` exitoso.
- **go vet:** sin warnings.
- **go fmt:** sin cambios.
- **go test:** 105 tests PASS, 0 FAIL.
- **Cobertura:** 95.7% de statements.

---

## 3. Estructura Real Encontrada

```
dantrack-sorter/
├── assignment.go           ✓ existe
├── assignment_test.go      ✓ existe
├── engine.go               ✓ existe
├── engine_test.go          ✓ existe
├── errors.go               ✓ existe
├── go.mod                  ✓ existe
├── README.md               ✓ existe
├── sorter.go               ✓ existe
├── sorter_test.go          ✓ existe
├── types.go                ✓ existe
├── validation.go           ✓ existe
├── validation_test.go      ✓ existe
├── docs/
│   ├── ITERATION_01_PROPOSAL.md           ✓ existe
│   ├── ITERATION_02_FOUNDATION.md         ✓ existe
│   ├── ITERATION_03_ASSIGNMENT_MATCHING.md ✓ existe
│   ├── ITERATION_04_SORTER_DECISION_ENGINE.md ✓ existe
│   └── ITERATION_05_PRESORTER_DECISION_ENGINE.md ✓ existe
├── presorter.go            ✗ NO EXISTE
├── presorter_test.go       ✗ NO EXISTE
├── balance.go              ✗ NO EXISTE
└── balance_test.go         ✗ NO EXISTE
```

---

## 4. Iteraciones Detectadas

### Iteración 01 — Proposal

| Aspecto | Estado |
|---|---|
| Documento | Existe: `docs/ITERATION_01_PROPOSAL.md` |
| Contenido | Correcto: alcance, arquitectura, modelos, API, plan |
| Inconsistencias | Mínimas: menciona `DecisionID` UUID v4 original, luego corregido en nota |
| Archivos | Solo documentación — correcto para una propuesta |

**Veredicto:** OK.

---

### Iteración 02 — Foundation

| Aspecto | Estado |
|---|---|
| Documento | Existe: `docs/ITERATION_02_FOUNDATION.md` |
| Archivos esperados | `types.go`, `errors.go`, `validation.go`, `validation_test.go` — todos existen |
| Tests | 33 tests originales (ahora expandidos a ~48 en `validation_test.go` tras adiciones de Iter 04) |
| Código | Correcto y funcional |

**Veredicto:** OK — completada correctamente.

---

### Iteración 03 — Assignment Matching

| Aspecto | Estado |
|---|---|
| Documento | Existe: `docs/ITERATION_03_ASSIGNMENT_MATCHING.md` |
| Archivos esperados | `assignment.go`, `assignment_test.go` — ambos existen |
| Tests | 37 tests (incluyendo subtests de campos directos) |
| API | `EvaluateAssignments` implementada correctamente |
| Operadores | Los 6 operadores implementados |
| Tags | `tags.*` implementado |
| Trazabilidad | `ConditionEvaluation` completo |

**Veredicto:** OK — completada correctamente.

---

### Iteración 04 — Sorter Decision Engine

| Aspecto | Estado |
|---|---|
| Documento | Existe: `docs/ITERATION_04_SORTER_DECISION_ENGINE.md` |
| Archivos esperados | `engine.go`, `engine_test.go`, `sorter.go`, `sorter_test.go` — todos existen |
| Tests | 7 tests engine + 26 tests sorter = 33 tests nuevos |
| API | `NewEngine`, `ResolveSorter` implementados |
| Ambigüedad | Implementada con `AmbiguityPolicy` |
| Fallback | DefaultExitID, RejectExitID implementados |
| Trazabilidad | TraceID, CorrelationID, CandidateEvaluations correctos |

**Veredicto:** OK — completada correctamente.

---

### Iteración 05 — Presorter Decision Engine + Balance Strategies

| Aspecto | Estado |
|---|---|
| Documento | Existe: `docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md` |
| `presorter.go` | **NO EXISTE** |
| `presorter_test.go` | **NO EXISTE** |
| `balance.go` | **NO EXISTE** |
| `balance_test.go` | **NO EXISTE** |
| `ResolvePresorter` | **NO IMPLEMENTADO** |
| `Engine.rrCounter` | **NO EXISTE** en engine.go |
| `ParkConfig.Weight` | **NO EXISTE** en types.go |
| `ParkState.Maintenance` | **NO EXISTE** en types.go (sí existe en SorterExitState) |
| `RulePresorterWeighted` | **NO EXISTE** en types.go |
| `PresorterDecision.Reason` | **NO EXISTE** (DecisionTrace.Reason sí existe) |
| `PresorterDecision.RuleApplied` | **NO EXISTE** |
| Tests esperados (41) | **NO EXISTEN** |

**Veredicto:** INCONSISTENCIA CRÍTICA. La documentación fue creada sin implementación correspondiente. El documento declara "Completada" con 146 tests en verde, pero el repositorio solo tiene 105 tests y ninguno de los archivos de la Iteración 05 existe.

---

## 5. APIs Públicas Reales

### Funciones

| Función | Archivo | Estado |
|---|---|---|
| `NewEngine(cfg EngineConfig) (*Engine, error)` | engine.go | Implementada |
| `(*Engine).ResolveSorter(ctx, req) (SorterDecision, error)` | sorter.go | Implementada |
| `(*Engine).ResolvePresorter(ctx, req) (PresorterDecision, error)` | — | **NO IMPLEMENTADA** |
| `EvaluateAssignments(box, set, evalTime) []AssignmentMatch` | assignment.go | Implementada |
| `ValidateConfig(cfg EngineConfig) error` | validation.go | Implementada |
| `ValidatePresorterConfig(cfg PresorterConfig) error` | validation.go | Implementada |
| `ValidateSorterConfig(cfg SorterConfig) error` | validation.go | Implementada |
| `ValidatePresorterRequest(req PresorterRequest) error` | validation.go | Implementada |
| `ValidateSorterRequest(req SorterRequest) error` | validation.go | Implementada |

### Tipos públicos

Tipos de control: `Action`, `Rule`, `BalanceStrategy`, `MatchOperator`, `TargetType`, `AmbiguityPolicy`.

Modelos de entrada: `BoxContext`, `Assignment`, `AssignmentSet`, `MatchCondition`, `PresorterRequest`, `SorterRequest`.

Modelos de configuración: `EngineConfig`, `PresorterConfig`, `ParkConfig`, `ParkState`, `SorterConfig`, `SorterExit`, `SorterExitState`.

Modelos de decisión: `PresorterDecision`, `SorterDecision`, `DecisionTrace`, `CandidateEvaluation`, `ConditionEvaluation`, `AssignmentMatch`.

### Constantes públicas

**Actions:** `ActionRoute`, `ActionReject`, `ActionRecirculate`, `ActionError`, `ActionNoop`.

**Rules:** `RuleNone`, `RulePresorterErrorControlPreferred`, `RulePresorterLeastLoaded`, `RulePresorterRoundRobin`, `RulePresorterFallbackDefaultPark`, `RulePresorterRejectNoAvailablePark`, `RuleSorterAssignmentMatched`, `RuleSorterFallbackDefaultExit`, `RuleSorterRejectNoMatch`, `RuleSorterRejectNoAvailableExit`.

**Balance strategies:** `BalanceLeastLoaded`, `BalanceRoundRobin`, `BalanceWeighted`.

**Operators:** `OpEquals`, `OpNotEquals`, `OpIn`, `OpNotIn`, `OpExists`, `OpNotExists`.

**Target types:** `TargetTypeExit`, `TargetTypePark`.

**Ambiguity policies:** `AmbiguityPolicyError`, `AmbiguityPolicyFirstWins`.

### Errores sentinel

`ErrInvalidConfig`, `ErrInvalidRequest`, `ErrMissingRequiredField`, `ErrNoAssignmentMatch`, `ErrNoAvailableExit`, `ErrNoAvailablePark`, `ErrAmbiguousAssignment`, `ErrContextCanceled`.

---

## 6. Tipos y Modelos Principales

Los tipos están correctamente definidos en `types.go`. Observaciones:

- `ParkConfig` no tiene campo `Weight` (documentado en Iter 05 pero nunca implementado).
- `ParkState` no tiene campo `Maintenance` (sí existe en `SorterExitState`).
- `PresorterDecision` no tiene `Reason` ni `RuleApplied` directos (solo a través de `Trace`).
- Las constantes de Rule para presorter existen (`RulePresorterLeastLoaded`, etc.) pero NO existe `RulePresorterWeighted` (documentada en Iter 05 pero no implementada).
- `Engine` struct no tiene `rrCounter` para round-robin atómico.

---

## 7. Validaciones

**Estado:** Completas para Iter 02–04.

- `ValidateConfig` — valida que al menos un motor esté configurado.
- `ValidatePresorterConfig` — parks no vacíos, ParkIDs únicos, BalanceStrategy válida, DefaultParkID existente.
- `ValidateSorterConfig` — SorterID, exits, ExitIDs únicos, DefaultExitID existente, RejectExitID existente, AmbiguityPolicy válida.
- `ValidatePresorterRequest` — BoxContext válido, ParkStates sin duplicados.
- `ValidateSorterRequest` — BoxContext válido, assignments válidas, ExitStates sin duplicados.
- `validateAssignment` — ID, Priority ≥ 0, TargetType, TargetID, rangos temporales, condiciones.
- `validateMatchCondition` — Field no vacío, Operator válido, Values según operador.

**Cobertura de validaciones:** Buena pero no 100%. Líneas no cubiertas menores (edge cases en `validateAssignment` y `validateMatchCondition` que no se alcanzan en los tests existentes).

---

## 8. Assignment Matching

**Estado:** Implementado completamente.

- `EvaluateAssignments` — evalúa todas las asignaciones con resultado completo.
- Operadores: `equals`, `not_equals`, `in`, `not_in`, `exists`, `not_exists`.
- Campos directos: 13 campos resueltos por switch.
- Tags dinámicos: `tags.*` vía prefijo.
- Vigencia temporal: `ValidFrom`/`ValidTo` con zero-value tolerance.
- Match universal: assignment sin condiciones.
- Ordenación: Priority descendente con `sort.SliceStable`.
- Trazabilidad: `ConditionEvaluation` para cada condición evaluada.

**Sin problemas detectados.**

---

## 9. Sorter Decision Engine

**Estado:** Implementado completamente.

- `ResolveSorter` — flujo completo con 14 pasos de decisión.
- Disponibilidad de exits: 6 condiciones de exclusión.
- Fallback: DefaultExitID cuando exits matched están bloqueados.
- Reject: RejectExitID o ActionReject sin exit.
- Ambigüedad: `checkAmbiguity` con dos políticas (`error`, `first_wins`).
- Trazabilidad: TraceID, CorrelationID, RuleApplied, Reason, CandidateEvaluations.
- `NewEngine` copia configs para aislamiento de mutación externa.
- `exitIndex` para lookup O(1).

**Sin problemas detectados.**

---

## 10. Presorter Decision Engine

**Estado:** NO IMPLEMENTADO.

- `ResolvePresorter` no existe como método.
- El `Engine` struct tiene campo `presorterCfg *PresorterConfig` listo para uso.
- `NewEngine` acepta y valida `PresorterConfig`.
- Los tipos de entrada/salida existen (`PresorterRequest`, `PresorterDecision`).
- Las validaciones existen (`ValidatePresorterRequest`, `ValidatePresorterConfig`).
- Las constantes de Rule para presorter existen (parcialmente).
- **Falta toda la lógica de decisión.**

---

## 11. Balance Strategies

**Estado:** NO IMPLEMENTADO.

- `BalanceStrategy` type y constantes existen.
- `PresorterConfig.BalanceStrategy` existe.
- `ValidatePresorterConfig` valida que la estrategia sea conocida.
- **No hay implementación de `selectLeastLoaded`, `selectRoundRobin`, `selectWeighted`.**
- **No hay `balance.go`.**

---

## 12. Trazabilidad

**Estado:** Implementada para Sorter. Pendiente para Presorter.

- `DecisionTrace` — struct completo con TraceID, CorrelationID, RuleApplied, Reason, DiagnosticMessage, CandidateEvaluations.
- `CandidateEvaluation` — incluye TargetType, TargetID, Eligible, RejectedReason, Rule, AssignmentID, Conditions.
- `ConditionEvaluation` — Field, Operator, Expected, Actual, Matched.
- TraceID/CorrelationID vienen del request (no se generan) — correcto.
- Sorter: completo y testeado.
- Presorter: la estructura existe pero no se usa porque no hay implementación.

---

## 13. Errores y Decisiones Válidas

**Diseño correcto:**

- `ActionReject` NO produce error Go — es una decisión válida. Verificado en tests.
- Errores Go solo para: contexto cancelado, request inválido, config inválida, ambigüedad irresoluble.
- Los errores sentinel son distinguibles entre sí (test explícito `TestErrors_AreDistinguishable`).
- `errors.Is` funciona correctamente con wrapping vía `fmt.Errorf("%w", ...)`.

---

## 14. Documentación

| Documento | Existe | Limpio | Corresponde a iteración correcta | Problemas |
|---|---|---|---|---|
| `ITERATION_01_PROPOSAL.md` | ✓ | Mayormente limpio | ✓ | Nota de corrección sobre UUID inline; aceptable |
| `ITERATION_02_FOUNDATION.md` | ✓ | Limpio | ✓ | Ninguno |
| `ITERATION_03_ASSIGNMENT_MATCHING.md` | ✓ | Limpio | ✓ | Ninguno |
| `ITERATION_04_SORTER_DECISION_ENGINE.md` | ✓ | Limpio | ✓ | Incluye plan para Iter 05 al final (aceptable) |
| `ITERATION_05_PRESORTER_DECISION_ENGINE.md` | ✓ | **CONTAMINADO** | ✗ | Describe código que NO existe. Dice "Completada" con 146 tests. Es falso. |

### Detalle de ITERATION_05_PRESORTER_DECISION_ENGINE.md

Este documento:
- Declara estado "Completada".
- Lista archivos creados/modificados que no existen (`presorter.go`, `balance.go`, etc.).
- Declara 41 tests nuevos que no existen.
- Declara 146 tests totales — el repo solo tiene 105.
- Declara modificaciones a `types.go` y `engine.go` que no se realizaron.
- Incluye sección "Resultado de Tests" con `ok` ficticio.
- Incluye "Next Agent Instructions" para Iteración 06 basadas en premisas falsas.

**Diagnóstico:** Este documento fue generado como output de planificación o fue creado por un agente que no persistió los cambios en el código. La documentación existe pero la implementación no.

---

## 15. README

| Aspecto | Estado |
|---|---|
| Explica objetivo | ✓ Bien |
| Muestra uso básico | ✓ Conceptual, suficiente |
| Refleja estado real | **Parcial** — dice Iter 05 "Próxima", que es correcto respecto al código; pero contradice el doc Iter 05 que dice "Completada" |
| Tiene ejemplos | Mínimos (uso conceptual) — aceptable para estado actual |
| Falta actualizar | Sí — debería reflejar que Iter 04 es la última implementada |

**Tabla de iteraciones en README:**

```
| 04 | Completada | ...            ← CORRECTO
| 05 | Próxima    | Presorter...   ← CORRECTO respecto al código
```

El README está más alineado con la realidad que el doc de Iter 05.

---

## 16. Tests y Cobertura

### Números reales

| Métrica | Valor |
|---|---|
| Tests totales (incluyendo subtests) | 105 (PASS + subtests reportados) |
| Tests FAIL | 0 |
| Cobertura global | **95.7%** |
| `go vet` warnings | 0 |
| `go fmt` cambios | 0 |

### Cobertura por función

| Función | Cobertura |
|---|---|
| `EvaluateAssignments` | 100% |
| `evaluateAssignment` | 100% |
| `isWithinTimeRange` | 100% |
| `evaluateCondition` | 100% |
| `resolveField` | 100% |
| `applyOperator` | 75% |
| `NewEngine` | 100% |
| `ResolveSorter` | 96% |
| `resolveNoMatch` | 100% |
| `checkAmbiguity` | 100% |
| `buildExitStateIndex` | 100% |
| `isExitAvailable` | 100% |
| `assignmentMatchesToCandidates` | 100% |
| `ValidateConfig` | 88.9% |
| `ValidatePresorterConfig` | 100% |
| `ValidateSorterConfig` | 100% |
| `validateBoxContext` | 100% |
| `ValidatePresorterRequest` | 90% |
| `ValidateSorterRequest` | 91.7% |
| `validateAssignmentSet` | 100% |
| `validateAssignment` | 78.6% |
| `validateMatchCondition` | 85.7% |

### Áreas bien cubiertas

- Assignment matching (100% en funciones core).
- Sorter decision engine (96%+).
- Validación de config presorter/sorter.
- Disponibilidad de exits.
- Ambigüedad.
- Trazabilidad.

### Áreas débiles

- `applyOperator` 75%: el default branch (operador desconocido) no se alcanza porque la validación previa lo impide. Aceptable.
- `validateAssignment` 78.6%: ramas de validación de `Priority < 0` y `TargetType` desconocido no se cubren directamente. Menor.
- `validateMatchCondition` 85.7%: menor.

### Tests faltantes

- Todo lo relacionado con Presorter (no hay implementación).
- No hay tests de concurrencia para `ResolveSorter` (aunque es stateless, sería bueno tenerlos).

### Tests frágiles

No se detectaron tests frágiles. Los tests son deterministas (usan `time.Time` fijo cuando importa) y no dependen de estado externo.

---

## 17. Comandos Ejecutados y Resultados

```
go fmt ./...     → sin cambios
go vet ./...     → sin warnings
go test ./... -count=1
  → ok  github.com/dantrack/dantrack-sorter  0.651s
  → 105 tests PASS, 0 FAIL
go test -cover ./... -count=1
  → coverage: 95.7% of statements
```

---

## 18. Inconsistencias Detectadas

### Críticas

| # | Inconsistencia | Impacto |
|---|---|---|
| 1 | `ITERATION_05_PRESORTER_DECISION_ENGINE.md` declara implementación completa pero el código NO EXISTE | Alto — documentación miente sobre el estado del repo |
| 2 | El doc Iter 05 dice "146 tests" pero el repo solo tiene 105 | Alto — números falsos |
| 3 | `presorter.go`, `balance.go`, `presorter_test.go`, `balance_test.go` no existen | Alto — la iteración 05 no se implementó |

### Medias

| # | Inconsistencia | Impacto |
|---|---|---|
| 4 | `ParkConfig.Weight` mencionado en doc Iter 05 como "agregado" pero no existe en types.go | Medio — campo necesario para estrategia weighted |
| 5 | `ParkState.Maintenance` mencionado en doc Iter 05 como "agregado" pero no existe (sí existe en SorterExitState) | Medio — inconsistencia de modelo |
| 6 | `RulePresorterWeighted` mencionada en doc Iter 05 pero no existe como constante | Medio — falta Rule para estrategia weighted |
| 7 | Doc Iter 05 dice `Engine.rrCounter uint64` fue agregado a engine.go pero no existe | Medio — round-robin no implementado |

### Menores

| # | Inconsistencia | Impacto |
|---|---|---|
| 8 | README dice "Iteración 02 — Foundation implementada" en la sección Estado, pero la tabla dice Iter 04 completada | Bajo — sección Estado desactualizada vs tabla |
| 9 | Doc Iter 04 incluye "Next Agent Instructions" para Iter 05 con detalles de implementación que pueden confundir | Bajo — instrucciones para futuro agente |
| 10 | `PresorterDecision` no tiene `Reason` ni `RuleApplied` como campos directos (solo en `Trace`) — doc Iter 05 dice que fueron "agregados" | Bajo — diseño actual es consistente; la decisión de agregar esos campos se puede tomar al implementar |

---

## 19. Riesgos Técnicos

| Riesgo | Severidad | Mitigación recomendada |
|---|---|---|
| Documentación contaminada puede confundir a futuros agentes/desarrolladores | Alta | Eliminar o reescribir doc Iter 05 |
| Round-robin con estado interno (rrCounter) rompe pureza del Engine | Media | Documentar claramente; es aceptable para este caso de uso |
| `ParkState.Maintenance` falta pero `SorterExitState.Maintenance` existe — asimetría | Media | Agregar campo al implementar presorter |
| El doc Iter 05 tiene "Next Agent Instructions" para Iter 06 basadas en premisas falsas | Media | No seguir esas instrucciones |
| No hay validación cruzada entre ParkState y ParkConfig (estados pueden referenciar parks inexistentes) | Baja | Política de ignorar silenciosamente — documentada y aceptable |
| `DefaultParkID` fallback aún no tiene tests reales (solo validación de config) | Baja | Se cubrirá al implementar presorter |

---

## 20. Qué Está Listo

- **Tipos y modelos base:** completos para Sorter y parcialmente para Presorter.
- **Errores sentinel:** todos definidos y distinguibles.
- **Validaciones:** completas para config y requests de ambos motores.
- **Assignment matching:** completo, testeado, con trazabilidad.
- **Sorter decision engine:** completo, testeado, con trazabilidad.
- **Infraestructura del Engine:** `NewEngine` acepta y valida config de presorter.
- **Principios de diseño:** todos respetados (sin DB, sin hardware, sin UUID auto, EvalTime como time.Time, ActionReject no es error, Rule tipado).

---

## 21. Qué Falta

| Faltante | Prioridad |
|---|---|
| Implementar `ResolvePresorter` | Alta — es el siguiente paso de negocio |
| Implementar estrategias de balanceo (`balance.go`) | Alta — dependencia de ResolvePresorter |
| Agregar `ParkConfig.Weight` | Alta — necesario para `weighted` |
| Agregar `ParkState.Maintenance` | Alta — consistencia con SorterExitState |
| Agregar `RulePresorterWeighted` | Alta — constante de regla faltante |
| Agregar `Engine.rrCounter` (o mecanismo de round-robin) | Alta — necesario para `round_robin` |
| Decidir si `PresorterDecision` necesita `Reason`/`RuleApplied` directos | Media — o se usa solo `Trace` |
| Limpiar documentación de Iter 05 | Alta — está contaminada |
| Actualizar README sección Estado | Baja — menor pero debería actualizarse |
| Tests de concurrencia para `ResolveSorter` | Baja — es stateless en su estado actual |

---

## 22. Recomendación de Siguiente Iteración

**Recomendación: A. Iteración 05.1 — Documentation Cleanup + Real Implementation**

Justificación:

1. El código de Iter 01–04 está limpio, correcto y bien testeado.
2. La documentación de Iter 05 es **completamente contaminada** — describe código inexistente.
3. No se puede avanzar a Iter 06 (Examples) sin tener Iter 05 implementada realmente.
4. No hay errores de diseño reales en el código existente.

La acción correcta es:

1. Reescribir/reemplazar `docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md` marcándolo como "Pendiente" o eliminarlo.
2. Implementar la Iteración 05 real (crear los archivos faltantes).

**Opción C (corrección técnica) no aplica** porque el código existente está correcto. El problema es puramente que la implementación de Iter 05 no se realizó pero la documentación dice que sí.

---

## 23. Plan Recomendado

### Paso 1 — Limpieza documental

- Reescribir `docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md` cambiando estado a "Pendiente — Documento de diseño, no de implementación".
- O eliminarlo y crear uno nuevo tras la implementación real.
- Actualizar README: sección "Estado" debe decir "Iteración 04 — Sorter Decision Engine completada".

### Paso 2 — Implementar Iteración 05 real

Crear:
- `presorter.go` — `ResolvePresorter`, `boxRequiresErrorControl`, `isParkAvailable`, helpers.
- `balance.go` — `selectLeastLoaded`, `selectRoundRobin`, `selectWeighted`.
- `presorter_test.go` — tests del motor de presorter.
- `balance_test.go` — tests de estrategias de balanceo.

Modificar:
- `types.go` — agregar `ParkConfig.Weight`, `ParkState.Maintenance`, `RulePresorterWeighted`.
- `engine.go` — agregar `rrCounter uint64` al struct Engine.

### Paso 3 — Verificar

```bash
go fmt ./...
go vet ./...
go test ./... -count=1
go test -cover ./...
```

### Paso 4 — Documentar resultado real

Reescribir `docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md` con el estado real post-implementación.

---

## 24. Next Agent Instructions

```
Iteración 05 — Presorter Decision Engine + Balance Strategies (IMPLEMENTACIÓN REAL)

Contexto:
- Iteraciones 02-04 completadas y verificadas con 105 tests en verde.
- La documentación de Iter 05 existente es FALSA — fue generada sin implementación.
- El código de Iter 05 NO EXISTE: no hay presorter.go, balance.go, ni sus tests.
- Los tipos base para Presorter ya existen (PresorterRequest, PresorterDecision,
  ParkConfig, ParkState, PresorterConfig, constantes de Rule parciales).
- Las validaciones de PresorterConfig y PresorterRequest ya existen.
- El Engine ya acepta y almacena PresorterConfig.

Paso 0 — Limpieza documental:
- Reescribir docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md como "Pendiente"
  o eliminarlo. Será recreado al final con datos reales.
- Actualizar README.md sección Estado: "Iteración 04 completada".

Paso 1 — Modificar types.go:
- Agregar ParkConfig.Weight int.
- Agregar ParkState.Maintenance bool.
- Agregar RulePresorterWeighted Rule = "presorter_weighted".
- Evaluar si PresorterDecision necesita Reason/RuleApplied directos (recomendado:
  no, usar Trace.Reason y Trace.RuleApplied como hace SorterDecision).

Paso 2 — Modificar engine.go:
- Agregar rrCounter uint64 al struct Engine.
- Importar sync/atomic si es necesario.

Paso 3 — Crear balance.go:
- selectLeastLoaded: menor CurrentLoad, desempate por index en config.
- selectRoundRobin: atomic.AddUint64 sobre Engine.rrCounter, candidates[counter % len].
- selectWeighted: mayor Weight (effective weight: <=0 → 1), desempate por index.
- Tipo parkCandidate si es útil para el flujo interno.

Paso 4 — Crear presorter.go:
- ResolvePresorter(ctx, req) (PresorterDecision, error).
- boxRequiresErrorControl(box BoxContext) bool.
- isParkAvailable(cfg ParkConfig, state *ParkState) bool.
- buildParkStateIndex(states []ParkState) map[string]*ParkState.
- Flujo: ctx check → validate → config check → evalTime → park states →
  filter available → error control → balance → fallback → reject.
- ActionReject NO es error Go.

Paso 5 — Crear balance_test.go:
- Tests unitarios para cada estrategia de balanceo.
- Tests de edge cases (un solo candidato, weight 0, estado faltante).

Paso 6 — Crear presorter_test.go:
- Tests del flujo completo de ResolvePresorter.
- Tests de disponibilidad de parks (cada condición de exclusión).
- Tests de error control (preferencia, fallback).
- Tests de fallback a DefaultParkID.
- Tests de reject sin parks disponibles.
- Tests de trazabilidad.
- Test de concurrencia round-robin (100 goroutines).
- Test de context cancelado, request inválido, config no configurada.

Paso 7 — Verificar:
  go fmt ./...
  go vet ./...
  go test ./... -count=1
  go test -cover ./...

Paso 8 — Documentar:
- Reescribir docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md con datos REALES.
- Actualizar README.md con Iter 05 = Completada.

Reglas de diseño inviolables:
- Sin DB, sin hardware, sin dependencias externas.
- TraceID y CorrelationID vienen del request.
- EvalTime es time.Time (zero → time.Now()).
- ActionReject no es error Go.
- Rule es tipo tipado.
- No generar UUIDs.

Objetivo de tests: +35 a +45 tests nuevos (target ~150 totales).
Objetivo de cobertura: mantener >90% global.
```
