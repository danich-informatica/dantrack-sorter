# Iteración 04 — Sorter Decision Engine

## 1. Resumen ejecutivo

Esta iteración implementa el núcleo de la librería `dantrack-sorter`: el motor de
decisión del sorter industrial. Dado un `SorterRequest` con la caja, las asignaciones
activas y los estados operativos de los exits, el motor devuelve una `SorterDecision`
determinista con acción, exit elegido y trazabilidad completa.

Se implementó `NewEngine` (constructor seguro que valida la config y construye un
índice de exits para lookup O(1)) y `(*Engine).ResolveSorter` (función de decisión
pura que no abre conexiones, no genera IDs, no controla hardware).

---

## 2. Estado

**Completada**

---

## 3. Objetivo de la Iteración 04

Implementar el motor de decisión del sorter: `NewEngine` y `(*Engine).ResolveSorter`.
Dado un `SorterRequest` (caja + asignaciones + estados de exits), la función devuelve
una `SorterDecision` con acción, exit elegido y trazabilidad completa.

`ResolveSorter` usa `EvaluateAssignments` (implementada en Iteración 03) como
dependencia interna ya disponible.

---

## 4. Alcance implementado

- `Engine` struct con `sorterCfg`, `presorterCfg` y `exitIndex` (mapa O(1)).
- `NewEngine(cfg EngineConfig) (*Engine, error)` — constructor seguro.
- `(*Engine).ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)`.
- Helpers internos: `resolveNoMatch`, `checkAmbiguity`, `buildExitStateIndex`,
  `isExitAvailable`, `assignmentMatchesToCandidates`.
- Nuevos tipos en `types.go`: `AmbiguityPolicy`, constantes `AmbiguityPolicyError` /
  `AmbiguityPolicyFirstWins`, campos `SorterConfig.RejectExitID` y
  `SorterConfig.AmbiguityPolicy`.
- Validación de los nuevos campos en `validation.go`.

---

## 5. Archivos creados/modificados

| Archivo | Acción | Cambios |
|---|---|---|
| `types.go` | modificado | `AmbiguityPolicy` type + constantes; `SorterConfig.RejectExitID`, `SorterConfig.AmbiguityPolicy` |
| `validation.go` | modificado | Validación de `AmbiguityPolicy` y `RejectExitID`; `validAmbiguityPolicies` |
| `validation_test.go` | modificado | +6 tests para los nuevos campos |
| `engine.go` | **creado** | `Engine` struct + `NewEngine` |
| `sorter.go` | **creado** | `ResolveSorter`, `resolveNoMatch`, `checkAmbiguity`, helpers |
| `engine_test.go` | **creado** | 7 tests de construcción y aislamiento |
| `sorter_test.go` | **creado** | 26 tests de decisión |
| `README.md` | modificado | Iter 04 = Completada; Iter 05 = Próxima |

---

## 6. API agregada

### `NewEngine`

```go
func NewEngine(cfg EngineConfig) (*Engine, error)
```

Crea un `Engine` válido a partir de una `EngineConfig`. Llama `ValidateConfig`
internamente. Copia los valores de config para aislar al Engine de mutaciones externas.
Construye `exitIndex` (mapa `ExitID → SorterExit`) para lookup O(1) en `ResolveSorter`.
No abre conexiones, no inicializa hardware, no genera IDs.

Errores posibles: `ErrInvalidConfig`, `ErrMissingRequiredField`.

### `(*Engine).ResolveSorter`

```go
func (e *Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)
```

Decide a qué salida del sorter debe enviarse la caja descrita en `req`.
Devuelve una `SorterDecision` con acción, exit elegido, flag de fallback, flag de
rechazo y trazabilidad completa.

`ActionReject` NO produce error Go. Es una decisión de negocio válida.
Solo devuelve error Go para: contexto cancelado, request inválido, sorter no
configurado, ambigüedad irresoluble.

---

## 7. Flujo de decisión del sorter

```
 1.  ctx.Err() != nil                         → ErrContextCanceled
 2.  ValidateSorterRequest(req)               → error si inválido
 3.  e.sorterCfg == nil                       → ErrInvalidConfig
 4.  evalTime = req.EvalTime; si cero → time.Now() (una sola vez)
 5.  exitStateIdx = buildExitStateIndex(req.ExitStates)
 6.  allMatches = EvaluateAssignments(box, assignments, evalTime)   ← dependencia de Iter 03
 7.  matched = filter(allMatches, Matched == true)
 8.  len(matched) == 0                        → resolveNoMatch (ver sección 9)
 9.  topPriority = matched[0].Assignment.Priority
10.  topMatches = matches con Priority == topPriority
11.  len(topMatches) > 1                      → checkAmbiguity (ver sección 11)
12.  candidates = assignmentMatchesToCandidates(allMatches)
13.  for m in matched (orden prioridad descendente):
       exit en config Y Enabled Y disponible  → ActionRoute, RuleSorterAssignmentMatched
       exit no disponible                     → anotar candidato, continuar
14.  todos los exits matched no disponibles   → ver sección 9 y sección 10
```

---

## 8. Reglas de disponibilidad de exits

Función `isExitAvailable(exitID, exitIndex, stateIndex) bool`:

| Condición | Resultado |
|---|---|
| ExitID no existe en `exitIndex` | no disponible |
| `SorterExit.Enabled == false` | no disponible |
| Sin `SorterExitState` para el exit | **disponible** (política documentada) |
| `state.Available == false` | no disponible |
| `state.Blocked == true` | no disponible |
| `state.Full == true` | no disponible |
| `state.HasError == true` | no disponible |
| `state.Maintenance == true` | no disponible |
| Estado presente y todos los flags ok | disponible |

**Política de estado ausente**: si no se proveyó estado para un exit configurado y
habilitado, se asume disponible. Permite operar con resiliencia parcial cuando el
sistema de estados no reporta todos los exits.

---

## 9. Reglas de fallback

### Sin match — exit de rechazo o default disponible

```
RejectExitID != "" && disponible  → ActionReject, RuleSorterRejectNoMatch,      Rejected=true,  FallbackUsed=false
DefaultExitID != "" && disponible → ActionRoute,  RuleSorterRejectNoMatch,      Rejected=false, FallbackUsed=true
```

### Con match pero todos los exits del match bloqueados

```
DefaultExitID != "" && disponible → ActionRoute, RuleSorterFallbackDefaultExit, FallbackUsed=true
```

---

## 10. Reglas de rechazo

### Sin match — sin exit disponible

```
ningún exit válido → ActionReject, RuleSorterRejectNoAvailableExit, Rejected=true
```

### Con match pero todos los exits bloqueados y sin default

```
sin DefaultExitID disponible → ActionReject, RuleSorterRejectNoAvailableExit, Rejected=true
```

### Tabla de semántica de retorno

| Situación | error Go | `Action` | `Rejected` | `FallbackUsed` |
|---|---|---|---|---|
| Contexto cancelado | `ErrContextCanceled` | — | — | — |
| Request inválido | `ErrInvalidRequest` / `ErrMissingRequiredField` | — | — | — |
| Sorter no configurado | `ErrInvalidConfig` | — | — | — |
| Ambigüedad irresoluble | `ErrAmbiguousAssignment` | — | — | — |
| Match + exit disponible | `nil` | `route` | `false` | `false` |
| Match + exits bloqueados + DefaultExitID | `nil` | `route` | `false` | `true` |
| Match + exits bloqueados + sin default | `nil` | `reject` | `true` | `false` |
| Sin match + RejectExitID disponible | `nil` | `reject` | `true` | `false` |
| Sin match + DefaultExitID disponible | `nil` | `route` | `false` | `true` |
| Sin match + sin exits | `nil` | `reject` | `true` | `false` |

---

## 11. Manejo de ambigüedad

Ambigüedad se activa cuando múltiples assignments de igual prioridad apuntan a
exits DISTINTOS. Si apuntan al mismo exit, no hay ambigüedad.

| `AmbiguityPolicy` | Comportamiento |
|---|---|
| `""` (vacío) | Equivale a `"error"` (default seguro) |
| `"error"` | Devuelve `ErrAmbiguousAssignment` |
| `"first_wins"` | Usa el primer match en orden estable de `EvaluateAssignments` |

`first_wins` no modifica el slice: el orden estable proviene de `sort.SliceStable`
en `EvaluateAssignments`. El primer elemento de `matched` es siempre el correcto.

---

## 12. Trazabilidad implementada

Toda `SorterDecision` incluye en su campo `Trace`:

| Campo | Contenido |
|---|---|
| `Trace.TraceID` | Copiado de `SorterRequest.TraceID` (generado por orquestador) |
| `Trace.CorrelationID` | Copiado de `SorterRequest.CorrelationID` (generado por orquestador) |
| `Trace.RuleApplied` | Regla `Rule` que determinó la decisión |
| `Trace.Reason` | Descripción en lenguaje natural de la decisión |
| `Trace.DiagnosticMessage` | Mensaje técnico adicional (cuando aplica) |
| `Trace.CandidateEvaluations` | Todas las asignaciones evaluadas + disponibilidad de cada exit intentado |

`CandidateEvaluations` incluye tanto matches como no-matches, y tanto exits
elegidos como descartados, para trazabilidad de auditoría completa.

La librería **nunca genera** `TraceID` ni `CorrelationID`. Son responsabilidad del
orquestador.

---

## 13. Tests implementados

### `engine_test.go` — 7 tests

| Test | Verifica |
|---|---|
| `TestNewEngine_ValidSorterConfig` | Engine creado sin error con config de sorter |
| `TestNewEngine_ValidPresorterConfig` | Engine creado sin error con config de presorter |
| `TestNewEngine_BothConfigured` | Engine con ambas configs |
| `TestNewEngine_NilBothConfigs_ReturnsErrInvalidConfig` | Config vacía → error |
| `TestNewEngine_InvalidSorterConfig_ReturnsErrInvalidConfig` | Config inválida → error |
| `TestNewEngine_IsolatesFromExternalMutation` | Mutación externa no afecta el `exitIndex` |
| `TestNewEngine_ExitIndexBuilt` | `exitIndex` contiene exactamente los exits configurados |

### `sorter_test.go` — 26 tests

| Test | Verifica |
|---|---|
| `TestResolveSorter_DirectMatch_ActionRoute` | Match + exit disponible → `ActionRoute` |
| `TestResolveSorter_HigherPriorityMatchChosen` | Se elige el match de mayor prioridad |
| `TestResolveSorter_BlockedExitFallsToNextMatch` | Exit bloqueado → cae al siguiente match |
| `TestResolveSorter_AllMatchedExitsUnavailable_UsesDefaultExit` | Todos bloqueados → `DefaultExitID` |
| `TestResolveSorter_AllMatchedExitsUnavailable_NoDefault_RejectsNoAvailableExit` | Sin default → `ActionReject` |
| `TestResolveSorter_NoMatch_RejectExitAvailable_ActionReject` | Sin match + `RejectExitID` → `ActionReject` |
| `TestResolveSorter_NoMatch_NoRejectExit_DefaultAvailable_ActionRoute` | Sin match + `DefaultExitID` → `ActionRoute` |
| `TestResolveSorter_NoMatch_NoExits_ActionReject` | Sin match, sin exits → `ActionReject` |
| `TestResolveSorter_ExitFull_NotAvailable` | Estado `Full` → exit no elegible |
| `TestResolveSorter_ExitHasError_NotAvailable` | Estado `HasError` → exit no elegible |
| `TestResolveSorter_ExitInMaintenance_NotAvailable` | Estado `Maintenance` → exit no elegible |
| `TestResolveSorter_ExitAvailableFalse_NotAvailable` | `Available=false` → exit no elegible |
| `TestResolveSorter_MissingExitState_AssumedAvailable` | Sin estado → exit asumido disponible |
| `TestResolveSorter_Ambiguity_PolicyError` | Igual prioridad + exits distintos + error → `ErrAmbiguousAssignment` |
| `TestResolveSorter_Ambiguity_PolicyFirstWins` | Igual prioridad + exits distintos + first_wins → `ActionRoute` |
| `TestResolveSorter_Ambiguity_SameExit_NoError` | Igual prioridad + mismo exit → no ambigüedad |
| `TestResolveSorter_ContextCanceled_ReturnsErrContextCanceled` | Contexto cancelado → `ErrContextCanceled` |
| `TestResolveSorter_InvalidRequest_ReturnsErrInvalidRequest` | Request inválido → error Go |
| `TestResolveSorter_SorterNotConfigured_ReturnsErrInvalidConfig` | Engine sin sorterCfg → `ErrInvalidConfig` |
| `TestResolveSorter_ActionRejectIsNotGoError` | `ActionReject` no produce error Go |
| `TestResolveSorter_TraceIDAndCorrelationIDCopied` | IDs del orquestador propagados a `Trace` |
| `TestResolveSorter_CandidateEvaluationsPopulated` | `Trace.CandidateEvaluations` siempre populado |
| `TestResolveSorter_SorterIDPopulated` | `SorterID` de config copiado a la decisión |
| `TestResolveSorter_EvalTimeZeroUsesNow` | `EvalTime` cero → `time.Now()` en la decisión |
| `TestResolveSorter_EvalTimePropagated` | `EvalTime` no-cero → propagado exacto |
| `TestResolveSorter_DisabledExitInConfig_NotAvailable` | `Enabled=false` en config → exit no elegible |

### `validation_test.go` — +6 tests en Iteración 04

| Test | Verifica |
|---|---|
| `TestValidateSorterConfig_ValidAmbiguityPolicyError` | `"error"` es válido |
| `TestValidateSorterConfig_ValidAmbiguityPolicyFirstWins` | `"first_wins"` es válido |
| `TestValidateSorterConfig_UnknownAmbiguityPolicy` | Valor desconocido → `ErrInvalidConfig` |
| `TestValidateSorterConfig_EmptyAmbiguityPolicyIsValid` | Vacío → válido (default) |
| `TestValidateSorterConfig_RejectExitIDValid` | `RejectExitID` referencia exit existente → ok |
| `TestValidateSorterConfig_RejectExitIDNotFound` | `RejectExitID` sin exit existente → `ErrInvalidConfig` |

---

## 14. Comandos ejecutados

```bash
go fmt ./...
go vet ./...
go test ./... -count=1
```

Resultado:

```
ok      github.com/dantrack/dantrack-sorter     0.711s
```

---

## 15. Resultado de tests

| Iteración | Tests agregados | Total acumulado |
|---|---|---|
| 02 — Foundation | 33 | 33 |
| 03 — Assignment Matching | 37 | 70 |
| 04 — Sorter Decision Engine | +35 (+6 validación, +7 engine, +26 sorter) | **105** |

**105 tests — 0 FAIL — `go vet` sin warnings**

---

## 16. Decisiones técnicas aplicadas

**`exitIndex` en el Engine, no lectura directa de `sorterCfg.Exits`**
Lookup O(1) durante `ResolveSorter`. Con muchos exits, evita iterar el slice en cada
llamada. El índice se construye sobre la copia del struct, por lo que mutaciones
externas al slice original no lo afectan.

**Copia del struct de config en `NewEngine` (`sc := *cfg.Sorter`)**
Aísla al Engine de cambios externos al `SorterConfig` original. El slice de Exits
se referencia en el índice, no directamente, por lo que appends externos no afectan
al Engine.

**`resolveNoMatch` como método separado**
Mantiene `ResolveSorter` lineal y auditable. El caso no-match tiene su propia
lógica de prioridades: `RejectExitID > DefaultExitID > error`.

**`checkAmbiguity` no modifica el slice de matches**
`first_wins` no requiere cambios: `EvaluateAssignments` usa `sort.SliceStable`, por
lo que el primer elemento de `matched` es siempre el candidato correcto en orden
estable.

**`AmbiguityPolicy` vacío = comportamiento `"error"` (default seguro)**
Un campo vacío no fuerza al caller a conocer la constante. El default es seguro: si
hay ambigüedad no resuelta, el sistema devuelve un error explícito en lugar de tomar
una decisión silenciosa potencialmente incorrecta.

**`ActionReject` no es error Go**
El rechazo es una decisión de negocio válida con trazabilidad completa. Convertirlo
en error Go obligaría al caller a usar `errors.Is` para un camino operativo normal,
lo que va contra el patrón idiomático de Go.

**Estado de exit ausente = asumir disponible**
Permite operar con resiliencia parcial cuando el sistema de estados no reporta todos
los exits. La política está documentada explícitamente en el código y en este doc.

---

## 17. Qué NO se implementó todavía

- `(*Engine).ResolvePresorter` — pendiente Iteración 05.
- Estrategias de balanceo (`least_loaded`, `round_robin`, `weighted`) — pendiente Iteración 05.
- `presorter.go` y `balance.go` — pendiente Iteración 05.
- Integración con base de datos — fuera del alcance de la librería.
- Integración con `dantrack-connect`, `dantrack-db`, `dantrack-sim` — fuera del alcance.
- Control de hardware o PLCs — fuera del alcance.
- Persistencia del estado de round-robin entre reinicios — fuera del alcance.
- Ejemplos de uso completos en `examples/` — pendiente Iteración 06.
- `ARCHITECTURE.md` — pendiente Iteración 06.

---

## 18. Riesgos o pendientes

| Riesgo / Pendiente | Severidad | Nota |
|---|---|---|
| `round_robin` requerirá `atomic.Uint64` en `Engine` | Baja | Engine no será completamente inmutable en Iter 05; usar `sync/atomic` correctamente |
| `weighted` puede necesitar evaluación de si es stateless o stateful | Baja | A definir en Iter 05 |
| Sin ejemplos de uso aún | Baja | Pendiente Iter 06 |
| Tests de concurrencia no incluidos | Baja | Agregar test con `-race` en Iter 05 cuando se agregue estado mutable |

---

## 19. Siguiente paso recomendado

**Iteración 05 — Presorter Decision Engine + Balance Strategies**

Implementar `(*Engine).ResolvePresorter(ctx, req) (PresorterDecision, error)` con
las tres estrategias de balanceo: `least_loaded`, `round_robin`, `weighted`.

---

## 20. Next Agent Instructions — Iteración 05

### Objetivo

Implementar `(*Engine).ResolvePresorter` con selección de parks y balanceo de carga.
Misma filosofía que `ResolveSorter`: función pura, sin conexiones, sin hardware,
con trazabilidad completa.

### Archivos a crear

- `presorter.go` — `ResolvePresorter` + helpers de park:
  `isParkAvailable`, `buildParkStateIndex`, `parkMatchesToCandidates`
- `balance.go` — estrategias de balanceo:
  - `selectLeastLoaded(parks []ParkConfig, stateIdx map[string]ParkState) (ParkConfig, bool)`
  - `selectRoundRobin(parks []ParkConfig, stateIdx map[string]ParkState, counter *atomic.Uint64) (ParkConfig, bool)`
  - `selectWeighted(parks []ParkConfig, stateIdx map[string]ParkState) (ParkConfig, bool)`
- `presorter_test.go` — tests de `ResolvePresorter`
- `balance_test.go` — tests de las tres estrategias
- `docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md` — closure doc

### Archivos a modificar

- `engine.go` — agregar `roundRobinIdx atomic.Uint64` para round_robin
- `README.md` — Iter 05 = Completada, Iter 06 = Próxima

### Flujo esperado de `ResolvePresorter`

```
 1. ctx.Err() != nil                          → ErrContextCanceled
 2. ValidatePresorterRequest(req)             → error si inválido
 3. e.presorterCfg == nil                     → ErrInvalidConfig
 4. evalTime = req.EvalTime; si cero → time.Now()
 5. parkStateIdx = buildParkStateIndex(req.ParkStates)
 6. candidateParks = parks habilitados y disponibles (isParkAvailable)
 7. len(candidateParks) == 0                  → DefaultParkID si disponible, si no ErrNoAvailablePark
 8. seleccionar park con la estrategia de la config (least_loaded / round_robin / weighted)
 9. park seleccionado                         → ActionRoute, RulePresorterBalanced
```

### Reglas de disponibilidad de parks

- `ParkConfig.Enabled == false` — no elegible.
- Sin `ParkState` para el park — **asumir disponible** (misma política que exits).
- Si estado existe: `Available && !Blocked && !Full && !HasError && !Maintenance`.

### Estrategias de balanceo

| Estrategia | Descripción | Estado en Engine |
|---|---|---|
| `least_loaded` | Park con menor `ParkState.CurrentLoad`; desempate por `ParkID` (orden estable) | Stateless |
| `round_robin` | Rotación circular sobre parks disponibles; usa `Engine.roundRobinIdx atomic.Uint64` | Stateful |
| `weighted` | Selección proporcional a `ParkConfig.Weight`; desempate por `ParkID` | Stateless |

### Restricciones a respetar en Iteración 05

- NO integrar con base de datos ni `dantrack-db`.
- NO integrar con `dantrack-connect` ni `dantrack-sim`.
- NO controlar hardware ni PLCs.
- NO persistir el estado de round-robin entre reinicios del proceso.
- NO agregar dependencias externas (stdlib only).
- NO tocar `ResolveSorter` ni la lógica de Iteración 04.
- NO implementar ejemplos complejos — eso es Iteración 06.

### Tests mínimos para `presorter_test.go`

- Match directo + park disponible → `ActionRoute`.
- Estrategia `least_loaded`: elige park con menor carga.
- Estrategia `round_robin`: rota entre parks (al menos 2 llamadas consecutivas).
- Estrategia `weighted`: park con mayor weight es favorecido.
- Park `Full` → no elegible.
- Park `Enabled=false` → no elegible.
- Sin estado de park → asumido disponible.
- `DefaultParkID` como fallback cuando todos los candidatos están bloqueados.
- Contexto cancelado → `ErrContextCanceled`.
- Request inválido → error Go.
- Presorter no configurado → `ErrInvalidConfig`.
- `TraceID`/`CorrelationID` copiados a `PresorterDecision.Trace`.

### Tests mínimos para `balance_test.go`

- `selectLeastLoaded` elige el park con menor `CurrentLoad`.
- `selectLeastLoaded` desempata por `ParkID` de forma estable.
- `selectRoundRobin` avanza el contador en cada llamada.
- `selectRoundRobin` es seguro para uso concurrente (test con `-race`).
- `selectWeighted` favorece parks con mayor weight.
- `selectWeighted` con un solo park → lo elige siempre.