# dantrack-sorter

Motor de decisión de ruteo industrial para el ecosistema DANTRACK.

## Qué hace

Resuelve decisiones de **Presorter** (a qué parque enviar una caja) y **Sorter**
(a qué salida enviar una caja según asignaciones vigentes). Devuelve decisiones
trazables con razón, regla aplicada y candidatos evaluados.

**Funciones principales:**

- `EvaluateAssignments` — evalúa asignaciones contra un BoxContext.
- `NewEngine` — crea un Engine con configuración de sorter y/o presorter.
- `engine.ResolveSorter` — decide exit de sorter.
- `engine.ResolvePresorter` — decide park de presorter con balanceo.

## Qué NO hace

- No controla hardware (PLC, Modbus, OPC-UA, TCP, cámaras, scanners).
- No abre conexiones a base de datos.
- No lee asignaciones desde DB — las recibe del orquestador.
- No ejecuta físicamente el desvío — solo decide.
- No genera UUIDs ni IDs.
- No depende de `dantrack-connect`, `dantrack-db` ni `dantrack-sim`.

## Estado

**v0.1.0-rc1** — Release Candidate.  
Motor completo: `ResolveSorter` + `ResolvePresorter` con estrategias de balanceo.  
156 tests · 95.3% cobertura · race-free · sin dependencias externas.

Ver [CHANGELOG.md](CHANGELOG.md) para historial de cambios.

## Instalación

```bash
go get github.com/dantrack/dantrack-sorter
```

## Ejemplo Mínimo — Sorter

```go
package main

import (
    "context"
    "fmt"
    "time"

    sorter "github.com/dantrack/dantrack-sorter"
)

func main() {
    engine, _ := sorter.NewEngine(sorter.EngineConfig{
        Sorter: &sorter.SorterConfig{
            SorterID:      "sorter-01",
            Exits:         []sorter.SorterExit{{ExitID: "exit_A", Enabled: true}},
            DefaultExitID: "exit_A",
        },
    })

    decision, _ := engine.ResolveSorter(context.Background(), sorter.SorterRequest{
        TraceID: "trace-001",
        Box:     sorter.BoxContext{BoxID: "BOX-001", QR: "QR001", Variety: "Fuji"},
        Assignments: sorter.AssignmentSet{
            Assignments: []sorter.Assignment{{
                ID: "assign_fuji", Enabled: true, Priority: 10,
                Conditions: []sorter.MatchCondition{
                    {Field: "variety", Operator: sorter.OpEquals, Values: []string{"Fuji"}},
                },
                TargetType: sorter.TargetTypeExit, TargetID: "exit_A",
            }},
        },
        ExitStates: []sorter.SorterExitState{{ExitID: "exit_A", Available: true}},
        EvalTime:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
    })

    fmt.Printf("Action=%s ExitID=%s\n", decision.Action, decision.ExitID)
    // Output: Action=route ExitID=exit_A
}
```

## Ejemplo Mínimo — Presorter

```go
engine, _ := sorter.NewEngine(sorter.EngineConfig{
    Presorter: &sorter.PresorterConfig{
        Parks: []sorter.ParkConfig{
            {ParkID: "park_1", Enabled: true, Weight: 3},
            {ParkID: "park_2", Enabled: true, Weight: 5},
        },
        BalanceStrategy: sorter.BalanceLeastLoaded,
        DefaultParkID:   "park_1",
    },
})

decision, _ := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
    TraceID: "trace-002",
    Box:     sorter.BoxContext{BoxID: "BOX-002", QR: "QR002"},
    ParkStates: []sorter.ParkState{
        {ParkID: "park_1", Available: true, CurrentLoad: 30},
        {ParkID: "park_2", Available: true, CurrentLoad: 10},
    },
    EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
})

fmt.Printf("Action=%s ParkID=%s\n", decision.Action, decision.ParkID)
// Output: Action=route ParkID=park_2
```

## Tests

```bash
go test ./... -count=1        # ejecutar tests
go test -cover ./...          # ver cobertura
go test -run "^Example" -v    # ejecutar godoc examples
```

## Examples

Cada example es un módulo independiente con `replace` directive hacia el módulo principal.

```bash
cd examples/basic_sorter   && go run .
cd examples/basic_presorter && go run .
cd examples/error_control  && go run .
cd examples/fallbacks      && go run .
```

| Example | Demuestra |
|---|---|
| `basic_sorter` | Crear engine, evaluar assignments, resolver sorter |
| `basic_presorter` | Presorter con least_loaded, parks con carga variable |
| `error_control` | Preferencia de parks con error control para cajas con errores |
| `fallbacks` | Fallback a default exit/park y ActionReject |

## Arquitectura Resumida

```text
Orquestador
  → carga config (dantrack-db)
  → recibe evento (dantrack-connect)
  → llama dantrack-sorter (decisión pura)
  → ejecuta acción física (dantrack-connect)
  → guarda auditoría (dantrack-db)
```

`dantrack-sorter` es una librería pura: recibe datos, devuelve decisiones.
No tiene I/O, no tiene estado global, no tiene dependencias externas.

Ver [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) para arquitectura completa.  
Ver [docs/INTEGRATION_CONTRACTS.md](docs/INTEGRATION_CONTRACTS.md) para contratos de integración.  
Ver [docs/PUBLIC_API_REVIEW.md](docs/PUBLIC_API_REVIEW.md) para revisión de API.  
Ver [docs/RELEASE_CHECKLIST.md](docs/RELEASE_CHECKLIST.md) para checklist de release.

## Iteration Status

| Iteración | Estado | Descripción |
|---|---|---|
| 01 | Completada | Propuesta técnica y arquitectura — [ITERATION_01_PROPOSAL.md](docs/ITERATION_01_PROPOSAL.md) |
| 02 | Completada | Foundation: modelos, errores, validaciones, tests — [ITERATION_02_FOUNDATION.md](docs/ITERATION_02_FOUNDATION.md) |
| 03 | Completada | Assignment matching: `EvaluateAssignments`, operadores, tags — [ITERATION_03_ASSIGNMENT_MATCHING.md](docs/ITERATION_03_ASSIGNMENT_MATCHING.md) |
| 04 | Completada | Sorter decision engine: `ResolveSorter`, `NewEngine` — [ITERATION_04_SORTER_DECISION_ENGINE.md](docs/ITERATION_04_SORTER_DECISION_ENGINE.md) |
| 05 | Completada | Presorter decision engine: `ResolvePresorter`, balanceo — [ITERATION_05_PRESORTER_DECISION_ENGINE.md](docs/ITERATION_05_PRESORTER_DECISION_ENGINE.md) |
| 06 | Completada | Examples, documentación, contratos — [ITERATION_06_EXAMPLES_AND_DOCS.md](docs/ITERATION_06_EXAMPLES_AND_DOCS.md) |
| 07 | Completada | Release Candidate, API hardening — [ITERATION_07_RELEASE_CANDIDATE.md](docs/ITERATION_07_RELEASE_CANDIDATE.md) |

## Principios

- **Puro:** sin I/O, sin side effects, sin estado global.
- **Determinista:** mismos inputs + mismo estado del engine → misma decisión.
- **Trazable:** toda decisión incluye razón, regla (`Rule`), candidatos evaluados.
- **Testeable:** sin mocks de infraestructura; tests con stdlib pura.
- **Configurable:** nada quemado; todo viene por config o request.
- **Sin dependencias:** solo stdlib de Go.

## Limitaciones Conocidas

- Round-robin counter no persiste entre reinicios del proceso.
- `weighted` strategy es determinista (siempre elige mayor peso).
- Sin adapters de integración (responsabilidad del orquestador).
- Sin métricas ni observabilidad integrada.

## Próximos Pasos

El motor de decisión está completo y listo para uso en producción (v0.1.0-rc1).

Posibles extensiones para v0.2.0:
- Adapters: interfaces `ConfigLoader`, `StateProvider`, `AuditWriter`.
- Métricas: hooks para Prometheus/OpenTelemetry.
- Nuevos operadores: regex, contains, range numérico.
- Weighted-random: distribución proporcional por peso.
