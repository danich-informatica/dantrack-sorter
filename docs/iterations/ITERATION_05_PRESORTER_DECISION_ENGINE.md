# Iteración 05 — Presorter Decision Engine + Balance Strategies

**Autor:** Arquitectura DANTRACK  
**Fecha:** 2026-06-02  
**Estado:** Completada  
**Módulo:** `github.com/dantrack/dantrack-sorter`  
**Package:** `package sorter`

---

## 1. Resumen Ejecutivo

La Iteración 05 implementa el motor de decisión del Presorter y tres estrategias de
balanceo. Dado un `PresorterRequest` con la caja, los estados de los parks y la
configuración del engine, `ResolvePresorter` devuelve una `PresorterDecision`
determinista con park elegido, estrategia aplicada, información de control de errores
y trazabilidad completa.

Se implementaron las estrategias `least_loaded`, `round_robin` y `weighted` en
`balance.go`, con helpers para control de errores (`boxRequiresErrorControl`) y
disponibilidad de parks (`isParkAvailable`).

---

## 2. Estado

**Completada**

---

## 3. Contexto — Catastro Previo

Antes de esta iteración, existía un documento `ITERATION_05_PRESORTER_DECISION_ENGINE.md`
que declaraba la iteración como completada con 146 tests. Sin embargo, un catastro
realizado el 2026-06-02 determinó que **los archivos de código nunca fueron creados**.
El documento anterior fue generado sin implementación correspondiente.

Este documento reemplaza al anterior con datos reales verificados contra el código
existente y los resultados de `go test`.

Estado pre-implementación real:
- 105 tests existentes (Iter 02–04).
- Cobertura: 95.7%.
- Archivos `presorter.go`, `balance.go` y sus tests NO existían.

---

## 4. Objetivo

Implementar `(*Engine).ResolvePresorter` y las estrategias básicas de balanceo para
elegir entre múltiples parks disponibles en el presorter industrial.

---

## 5. Alcance Implementado

| Área | Implementado |
|---|---|
| `ResolvePresorter` | Sí — flujo completo con trazabilidad |
| Balance `least_loaded` | Sí — elige menor CurrentLoad, desempate estable |
| Balance `round_robin` | Sí — rotación atómica, segura para concurrencia |
| Balance `weighted` | Sí — elige mayor peso, determinista |
| Control de errores | Sí — detección y preferencia |
| Disponibilidad de parks | Sí — 6 condiciones de exclusión |
| Fallback a DefaultParkID | Sí |
| ActionReject sin error Go | Sí |
| Trazabilidad completa | Sí — TraceID, CorrelationID, CandidateEvaluations |
| Tests | Sí — 40 nuevos tests (145 totales) |

---

## 6. Archivos Creados/Modificados

| Archivo | Acción | Descripción |
|---|---|---|
| `presorter.go` | **Creado** | `ResolvePresorter`, `boxRequiresErrorControl`, `isParkAvailable`, `buildParkStateIndex`, `selectByStrategy`, `parkRejectedReason` |
| `balance.go` | **Creado** | `selectLeastLoaded`, `selectRoundRobin`, `selectWeighted`, `effectiveWeight`, `currentLoadOf`, `parkCandidate` |
| `presorter_test.go` | **Creado** | 27 tests del motor de presorter |
| `balance_test.go` | **Creado** | 13 tests de estrategias de balanceo |
| `types.go` | Modificado | `RulePresorterWeighted`, `ParkConfig.Weight`, `ParkState.Maintenance` |
| `engine.go` | Modificado | `Engine.rrCounter uint64` para round-robin atómico |
| `README.md` | Modificado | Estado actualizado a Iter 05 completada |

---

## 7. API Agregada

### `(*Engine).ResolvePresorter`

```go
func (e *Engine) ResolvePresorter(ctx context.Context, req PresorterRequest) (PresorterDecision, error)
```

Decide a qué parque del presorter debe enviarse la caja descrita en `req`.
Devuelve una `PresorterDecision` con park elegido, acción, estrategia de balanceo,
flags de control de errores y trazabilidad completa.

`ActionReject` NO produce error Go. Es una decisión de negocio válida.
Solo devuelve error Go para: contexto cancelado, request inválido, presorter no configurado.

---

## 8. Flujo de Decisión del Presorter

```
 1. ctx.Err() != nil                         → ErrContextCanceled
 2. ValidatePresorterRequest(req)            → error si inválido
 3. e.presorterCfg == nil                    → ErrInvalidConfig
 4. evalTime = req.EvalTime; si cero → time.Now()
 5. parkStateIdx = buildParkStateIndex(req.ParkStates)
 6. Para cada park en config:
      isParkAvailable(cfg, state)            → candidato o descartado
 7. Si caja requiere error control:
      filtrar candidatos con HasErrorControl
      si hay → usar solo esos
      si no → usar todos (fallback)
 8. Si hay candidatos:
      elegir por BalanceStrategy             → ActionRoute
 9. Si no hay candidatos:
      DefaultParkID disponible               → ActionRoute + FallbackUsed
10. Si no hay park válido                    → ActionReject
```

---

## 9. Reglas de Disponibilidad de Parks

Función `isParkAvailable(cfg ParkConfig, state *ParkState) bool`:

| Condición | Resultado |
|---|---|
| `ParkConfig.Enabled == false` | no disponible |
| Sin `ParkState` para el park | **disponible** (política documentada) |
| `state.Available == false` | no disponible |
| `state.Blocked == true` | no disponible |
| `state.Full == true` | no disponible |
| `state.HasError == true` | no disponible |
| `state.Maintenance == true` | no disponible |
| Estado presente y todos los flags ok | disponible |

**Política de estado ausente**: si no se proveyó estado para un park configurado y
habilitado, se asume disponible. Permite operar con resiliencia parcial cuando el
sistema de estados no reporta todos los parks.

---

## 10. Reglas de Control de Errores

### Detección

`boxRequiresErrorControl(box BoxContext) bool` devuelve true si:

- `ReadStatus` ∈ {`error`, `no_read`, `no-read`, `noread`} (case-insensitive, trimmed)
- `QualityStatus` ∈ {`error`, `rejected`, `bad`} (case-insensitive, trimmed)
- `len(ErrorFlags) > 0`

### Preferencia

Si la caja requiere control de errores:
1. Se filtran candidatos con `ParkConfig.HasErrorControl == true`.
2. Si hay al menos uno, se usan solo esos → `ErrorControlAvailable = true`.
3. Si no hay ninguno, se usan todos los candidatos originales → `FallbackUsed = true`.

### Campos de la decisión

- `ErrorControlRequired`: si la caja requiere control de errores.
- `ErrorControlAvailable`: si se encontró al menos un park con error control disponible.
- `FallbackUsed`: si no había park con error control y se usó uno sin control.

---

## 11. Reglas de Fallback

### Sin candidatos — DefaultParkID

Si ningún park candidato está disponible (todos blocked/full/error/maintenance/disabled):

1. Si `DefaultParkID != ""` y el park default está disponible → `ActionRoute` con `FallbackUsed = true` y `RulePresorterFallbackDefaultPark`.
2. Si no → `ActionReject` con `RulePresorterRejectNoAvailablePark`.

---

## 12. Estrategias de Balanceo

### `least_loaded`

- Elige el park con menor `CurrentLoad`.
- Si no hay estado para un park, `CurrentLoad = 0`.
- Desempata por posición en `PresorterConfig.Parks` (menor index primero).
- Determinista y estable.

### `round_robin`

- Usa contador atómico (`sync/atomic.AddUint64`) en `Engine.rrCounter`.
- Selecciona `candidates[counter % len(candidates)]`.
- Seguro para concurrencia sin mutex.
- No persiste estado entre reinicios del proceso.

### `weighted`

- Elige el park con mayor `ParkConfig.Weight`.
- `Weight <= 0` se trata como `1`.
- Desempata por posición en config (menor index primero).
- Determinista. No usa random ni distribución proporcional en esta versión.

---

## 13. Trazabilidad Implementada

`DecisionTrace` incluye:

- `TraceID` — copiado del request.
- `CorrelationID` — copiado del request.
- `RuleApplied` — regla que produjo la decisión.
- `Reason` — descripción legible del motivo.
- `DiagnosticMessage` — información adicional para debugging.
- `CandidateEvaluations` — evaluación de cada park:
  - `TargetType: "park"`
  - `TargetID: parkID`
  - `Eligible: true/false`
  - `RejectedReason` cuando aplique
  - `Rule` aplicada al candidato elegido

---

## 14. Tests Implementados

### `presorter_test.go` (27 tests)

- Park disponible → ActionRoute
- least_loaded elige menor carga
- round_robin rota entre parks
- weighted elige mayor peso
- Park Full no elegible
- Park Blocked no elegible
- Park con error no elegible
- Park en mantenimiento no elegible
- Park Enabled=false no elegible
- Sin ParkState se asume disponible
- DefaultParkID como fallback
- Todos unavailable con DefaultParkID bloqueado → ActionReject
- Sin park válido → ActionReject con RulePresorterRejectNoAvailablePark
- ActionReject no devuelve error Go
- Context cancelado → ErrContextCanceled
- Request inválido → ErrMissingRequiredField
- Presorter no configurado → ErrInvalidConfig
- TraceID y CorrelationID copiados a DecisionTrace
- CandidateEvaluations poblado correctamente
- ErrorFlags prefiere park con error control
- no_read prefiere park con error control
- QualityStatus bad prefiere park con error control
- Sin error control disponible → usa fallback y marca FallbackUsed
- ErrorControlRequired y ErrorControlAvailable correctos
- EvalTime usa valor del request
- EvalTime defaults to now
- DefaultParkFallback cuando todos los otros están unavailable

### `balance_test.go` (13 tests)

- selectLeastLoaded elige menor CurrentLoad
- selectLeastLoaded trata estado faltante como carga 0
- selectLeastLoaded desempata estable por index
- selectLeastLoaded con un solo candidato
- selectWeighted elige mayor Weight
- selectWeighted trata Weight <= 0 como 1
- selectWeighted desempata estable por index
- selectRoundRobin rota
- selectRoundRobin no hace panic con un solo candidato
- selectRoundRobin es seguro para concurrencia (100 goroutines)
- effectiveWeight tabla de valores
- currentLoadOf con state
- currentLoadOf con nil state

---

## 15. Comandos Ejecutados

```bash
go fmt ./...
go vet ./...
go test ./... -count=1
go test -cover ./...
```

---

## 16. Resultado Real de Tests

```
ok   github.com/dantrack/dantrack-sorter   0.828s   coverage: 94.9% of statements
```

**145 tests — 0 FAIL — `go vet` sin warnings**

| Iteración | Tests agregados | Total acumulado |
|---|---|---|
| 02 — Foundation | 33 | 33 |
| 03 — Assignment Matching | 34 | 67 (con subtests) |
| 04 — Sorter Decision Engine | 38 | 105 |
| 05 — Presorter Decision Engine | +40 | **145** |

---

## 17. Cobertura Real

| Archivo | Función | Cobertura |
|---|---|---|
| balance.go | `selectLeastLoaded` | 100% |
| balance.go | `selectRoundRobin` | 100% |
| balance.go | `selectWeighted` | 100% |
| balance.go | `effectiveWeight` | 100% |
| balance.go | `currentLoadOf` | 100% |
| presorter.go | `ResolvePresorter` | 91.9% |
| presorter.go | `selectByStrategy` | 100% |
| presorter.go | `buildParkStateIndex` | 100% |
| presorter.go | `isParkAvailable` | 88.9% |
| presorter.go | `boxRequiresErrorControl` | 100% |
| presorter.go | `parkRejectedReason` | 80% |
| **total** | | **94.9%** |

---

## 18. Decisiones Técnicas Aplicadas

| Decisión | Justificación |
|---|---|
| `ParkState.Maintenance` agregado | Consistencia con `SorterExitState` que ya tenía `Maintenance` |
| `ParkConfig.Weight` agregado | Requerido para estrategia `weighted` |
| `RulePresorterWeighted` agregado | Constante de Rule para la estrategia weighted |
| Round-robin con `sync/atomic` | Seguro, sin mutex, sin contención; stdlib pura |
| Sin `ParkState` = disponible | Resiliencia: permite operar sin sistema de estados completo |
| Desempate por index en config | Determinista, estable, documentado |
| `weighted` determinista (max weight) | Primera versión sin random; futuras pueden agregar weighted-random |
| `FallbackUsed` dual | Se marca tanto si se usa DefaultParkID como si se usa park sin error control |
| `boxRequiresErrorControl` conservador | Múltiples variantes de "no_read" aceptadas; `ErrorFlags > 0` siempre indica control |
| No agregar `Reason`/`RuleApplied` a PresorterDecision | Se usa `Trace.Reason` y `Trace.RuleApplied` como en SorterDecision; evita duplicación |
| `parkCandidate` como tipo interno | Permite transportar cfg + index para desempate estable |
| `least_loaded` como estrategia default | Si BalanceStrategy está vacía, se usa least_loaded |

---

## 19. Qué NO se Implementó

- Integración con DB.
- Integración con hardware.
- Integración con `dantrack-connect` / `dantrack-db` / `dantrack-sim`.
- Persistencia del contador round-robin entre reinicios.
- Weighted-random con distribución proporcional.
- Examples complejos.
- Cambios en `ResolveSorter` (no fueron necesarios).

---

## 20. Riesgos o Pendientes

| Riesgo | Mitigación |
|---|---|
| Round-robin counter se reinicia con el proceso | Documentado; no afecta correctitud, solo distribución inicial |
| `weighted` siempre elige el mismo park si los pesos no cambian | Aceptable para v1; se puede evolucionar a weighted-random |
| Sin validación cruzada ParkState vs ParkConfig | El estado puede referenciar parks inexistentes; se ignoran silenciosamente |
| Control de errores acepta variantes case-insensitive | Conservador: si el campo indica error de alguna forma, se activa control |
| `parkRejectedReason` cobertura 80% | Las ramas no alcanzadas son defensivas; la función nunca se llama con estados imposibles |

---

## 21. Siguiente Paso Recomendado

**Iteración 06 — Examples + Usage Documentation + Integration Contracts**

- Examples ejecutables con `go run`.
- Documentación de uso para cada caso de negocio.
- Contratos de integración con `dantrack-connect` y `dantrack-db`.
- `ARCHITECTURE.md` final con diagramas de flujo.

---

## 22. Next Agent Instructions

```
Iteración 06 — Examples + Usage Documentation + Integration Contracts

Contexto:
- Iteraciones 02-05 completadas.
- Motor completo: ResolveSorter + ResolvePresorter con 145 tests en verde.
- Cobertura: 94.9%.
- No hay dependencias externas.

Objetivo:
- Crear examples/ con programas ejecutables que demuestren uso real.
- Crear ARCHITECTURE.md con diagramas y flujo completo.
- Documentar contratos de integración para orquestadores.
- Agregar godoc examples (func Example*) en *_test.go.

No implementar:
- DB, hardware, connect, sim.
- Nuevas features del motor.
- Cambios en la lógica de decisión.

Archivos esperados:
- examples/basic/main.go
- examples/presorter/main.go
- examples/error_control/main.go
- docs/ARCHITECTURE.md
- docs/INTEGRATION_CONTRACTS.md
- docs/ITERATION_06_EXAMPLES.md
- Actualizar README.md
```
