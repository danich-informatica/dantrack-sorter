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

**v0.3.0** — Stable release.  
Motor completo: `ResolveSorter` + `ResolvePresorter` con balanceo multi-target,
error park fijo, balanceo global por ventana, vistas de disponibilidad,
y helpers/builders para reducción de boilerplate del consumidor.  
310 tests · 95.1% cobertura · race-free · sin dependencias externas.

Ver [CHANGELOG.md](CHANGELOG.md) para historial de cambios.

## Instalación

```bash
go get github.com/danich-informatica/dantrack-sorter
```

## Ejemplo Mínimo — Sorter

```go
package main

import (
    "context"
    "fmt"
    "time"

    sorter "github.com/danich-informatica/dantrack-sorter"
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

## Balanceo Global (AccumulatedLoad)

El presorter soporta balanceo por ventana de carga: si `ParkState.AccumulatedLoad > 0`
en al menos un candidato, la selección prioriza el park con menor carga acumulada.

**Importante:** `AccumulatedLoad` no es un total histórico de jornada.
Representa la carga acumulada dentro de una **ventana de balance** que el orquestador
define y calcula (ej: últimos 60 segundos, últimos 2 minutos).

```go
ParkStates: []sorter.ParkState{
    {ParkID: "park_1", Available: true, CurrentLoad: 5, AccumulatedLoad: 120},
    {ParkID: "park_2", Available: true, CurrentLoad: 8, AccumulatedLoad: 95},
}
// → elige park_2 (menor AccumulatedLoad en ventana)
```

La librería no calcula ni resetea la ventana. El orquestador es responsable de:
1. Definir el tamaño de ventana (30s, 1min, 5min, etc).
2. Calcular cuántas cajas fueron enrutadas a cada park en esa ventana.
3. Entregar el snapshot en `ParkState.AccumulatedLoad`.

Si `AccumulatedLoad == 0` en todos los parks, se usa `CurrentLoad` (least_loaded).

## Helpers y Builders

La librería incluye helpers para reducir el boilerplate del consumidor.
Todas las funciones son aditivas y compatibles con la API v0.2.0.

**Match helpers** — shortcuts para `MatchCondition`:

```go
sorter.MatchEquals("variety", "Fuji")
sorter.MatchIn("caliber", "80", "85", "90")
sorter.MatchExists("tags.lote")
```

**State helpers** — constructores semánticos para estados:

```go
sorter.ExitAvailable("exit_A", 30)
sorter.ExitBlocked("exit_B")
sorter.ParkAvailable("park_1", 45)
sorter.ParkAvailableWithWindow("park_2", 10, 120)
```

**Box builder** — construcción fluida de `BoxContext`:

```go
box := sorter.NewBox("BOX-001").Variety("Fuji").Caliber("80").Build()
errBox := sorter.ErrorBox("BOX-ERR", "no_read")
```

**Assignment builder** — construcción fluida de `Assignment`:

```go
a := sorter.NewAssignment("a1", 10).
    MatchEquals("variety", "Fuji").
    MatchIn("caliber", "80", "85").
    ToExits("exit_A", "exit_B").
    Build()
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

## Repository Layout

```text
dantrack-sorter/
├── *.go                    # Package principal (package sorter)
├── *_test.go               # Tests unitarios
├── examples/               # Ejemplos ejecutables (módulos independientes)
│   ├── basic_sorter/
│   ├── basic_presorter/
│   ├── error_control/
│   └── fallbacks/
├── integration_smoke/      # Smoke test externo (módulo independiente)
├── README.md
├── CHANGELOG.md
├── go.mod
└── .gitignore
```

El package principal vive en la raíz para mantener el import limpio:

```go
import sorter "github.com/danich-informatica/dantrack-sorter"
```

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

El motor de decisión está completo y estable (v0.2.0).

Posibles extensiones para v0.3.0:
- Vista inversa: dado un SKU, qué exits/parks lo aceptan.
- Prioridad por tipo de producto en Presorter.
- Overflow rules entre parks.
- Nuevos operadores: regex, contains, range numérico.
- Adapters: interfaces `ConfigLoader`, `StateProvider`, `AuditWriter`.
