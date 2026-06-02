# Contratos de Integración — dantrack-sorter

Este documento define cómo un orquestador debe integrar `dantrack-sorter`
para tomar decisiones de clasificación en una planta de packing.

---

## 1. Responsabilidades del Orquestador

| # | Responsabilidad |
|---|---|
| 1 | Cargar configuración desde DB/archivo → `SorterConfig`, `PresorterConfig` |
| 2 | Escuchar eventos de hardware → construir `BoxContext` |
| 3 | Consultar estados de exits/parks → `ExitStates`, `ParkStates` |
| 4 | Generar TraceID y CorrelationID únicos |
| 5 | Llamar `engine.ResolveSorter` o `engine.ResolvePresorter` |
| 6 | Interpretar la decisión (Action, ExitID/ParkID) |
| 7 | Ejecutar acción física vía hardware/PLC |
| 8 | Persistir DecisionTrace para auditoría |
| 9 | Manejar errores Go (contexto, validación, config) |
| 10 | Decidir qué hacer con ActionReject (recircular, alarma, park manual) |

---

## 2. Responsabilidades de dantrack-sorter

| # | Responsabilidad |
|---|---|
| 1 | Evaluar assignments contra box context |
| 2 | Filtrar destinos disponibles |
| 3 | Aplicar estrategia de balanceo |
| 4 | Aplicar preferencia de error control |
| 5 | Aplicar fallback a default exit/park |
| 6 | Devolver decisión determinista con trazabilidad |
| 7 | Validar requests y configs |
| 8 | NO acceder a DB, hardware, red ni filesystem |

---

## 3. Contrato de Entrada — Sorter

### SorterRequest

```go
SorterRequest{
    TraceID:       string,       // Generado por orquestador (UUID recomendado)
    CorrelationID: string,       // Sesión, lote, turno
    Box:           BoxContext,   // Datos de la caja actual
    Assignments:   AssignmentSet,// Cargado desde DB
    ExitStates:    []SorterExitState, // Estado real de cada exit
    EvalTime:      time.Time,    // Momento de evaluación; zero → time.Now()
}
```

**Precondiciones obligatorias:**

- `Box.BoxID` o `Box.QR` no vacío.
- `Assignments` cargado y con al menos una assignment.
- `ExitStates` no debe tener duplicados por ExitID.
- `EvalTime` debe ser explícito para reproducibilidad (no confiar en time.Now()).

---

## 4. Contrato de Salida — Sorter

### SorterDecision

```go
SorterDecision{
    SorterID:     string,       // Copiado de SorterConfig.SorterID
    ExitID:       string,       // Exit seleccionado (vacío si reject sin exit)
    Action:       Action,       // "route" | "reject"
    AssignmentID: string,       // ID del assignment que hizo match (vacío si fallback/reject)
    FallbackUsed: bool,         // true si se usó DefaultExitID
    Rejected:     bool,         // true si Action == ActionReject
    Trace:        DecisionTrace,// Trazabilidad completa
    EvalTime:     time.Time,    // Momento de evaluación usado
}
```

**Postcondiciones garantizadas:**

- Si `Action == ActionRoute`: ExitID no vacío.
- Si `Action == ActionReject`: puede tener ExitID (RejectExitID) o vacío.
- `Trace.RuleApplied` siempre tiene valor.
- `Trace.CandidateEvaluations` contiene evaluación de cada exit.
- El error Go es nil salvo contexto cancelado, request inválido, config ausente o ambigüedad.

---

## 5. Contrato de Entrada — Presorter

### PresorterRequest

```go
PresorterRequest{
    TraceID:       string,       // UUID del orquestador
    CorrelationID: string,       // Correlación
    Box:           BoxContext,   // Datos de la caja
    ParkStates:    []ParkState,  // Estado actual de cada park
    EvalTime:      time.Time,    // Momento de evaluación
}
```

**Precondiciones obligatorias:**

- `Box.BoxID` o `Box.QR` no vacío.
- `ParkStates` no debe tener duplicados por ParkID.
- Proporcionar estados para todos los parks configurados (parks sin estado se asumen disponibles).

---

## 6. Contrato de Salida — Presorter

### PresorterDecision

```go
PresorterDecision{
    ParkID:                string,          // Park seleccionado (vacío si reject)
    Action:                Action,          // "route" | "reject"
    FallbackUsed:          bool,            // true si se usó DefaultParkID o park sin EC
    ErrorControlRequired:  bool,            // true si la caja necesita error control
    ErrorControlAvailable: bool,            // true si se encontró park EC disponible
    BalanceStrategy:       BalanceStrategy, // Estrategia aplicada
    Trace:                 DecisionTrace,   // Trazabilidad completa
    EvalTime:              time.Time,       // Momento de evaluación
}
```

**Postcondiciones garantizadas:**

- Si `Action == ActionRoute`: ParkID no vacío.
- Si `Action == ActionReject`: ParkID vacío.
- `Trace.RuleApplied` siempre tiene valor.
- El error Go es nil salvo contexto cancelado, request inválido o config ausente.

---

## 7. Mapear Datos desde DB

La librería no depende de DB. El orquestador es responsable de mapear.

### Ejemplo conceptual (no importar, solo referencia):

```go
// En el orquestador (NO en dantrack-sorter):
func loadSorterConfig(db *dantrackdb.Client) (*sorter.SorterConfig, error) {
    rows, err := db.QueryExits(ctx, sorterID)
    if err != nil {
        return nil, err
    }

    exits := make([]sorter.SorterExit, 0, len(rows))
    for _, row := range rows {
        exits = append(exits, sorter.SorterExit{
            ExitID:   row.ExitID,
            Enabled:  row.Active,
            Priority: row.Priority,
            Capacity: row.MaxCapacity,
        })
    }

    return &sorter.SorterConfig{
        SorterID:      sorterID,
        Exits:         exits,
        DefaultExitID: findDefault(rows),
    }, nil
}
```

**Regla**: la conversión DB → struct de sorter vive en el orquestador o en un adapter layer, nunca en la librería.

---

## 8. Mapear Estados desde Hardware

La librería no depende de hardware. El orquestador traduce señales PLC.

### Ejemplo conceptual:

```go
// En el orquestador (NO en dantrack-sorter):
func buildExitStates(plc *dantrackconnect.PLCClient) ([]sorter.SorterExitState, error) {
    signals, err := plc.ReadAllExitSignals(ctx)
    if err != nil {
        return nil, err
    }

    states := make([]sorter.SorterExitState, 0, len(signals))
    for _, sig := range signals {
        states = append(states, sorter.SorterExitState{
            ExitID:      sig.ExitID,
            Available:   sig.SensorOK && !sig.Fault,
            Blocked:     sig.PhotocellBlocked,
            Full:        sig.CounterFull,
            HasError:    sig.Fault,
            Maintenance: sig.MaintenanceMode,
            CurrentLoad: sig.BoxCount,
        })
    }
    return states, nil
}
```

**Regla**: la traducción hardware → ExitState/ParkState es responsabilidad exclusiva del orquestador.

---

## 9. Guardar Auditoría usando DecisionTrace

```go
// En el orquestador (NO en dantrack-sorter):
func saveAudit(db *dantrackdb.Client, decision sorter.SorterDecision) error {
    return db.InsertDecisionAudit(ctx, dantrackdb.AuditRecord{
        TraceID:       decision.Trace.TraceID,
        CorrelationID: decision.Trace.CorrelationID,
        Timestamp:     decision.EvalTime,
        SorterID:      decision.SorterID,
        ExitID:        decision.ExitID,
        Action:        string(decision.Action),
        RuleApplied:   string(decision.Trace.RuleApplied),
        Reason:        decision.Trace.Reason,
        FallbackUsed:  decision.FallbackUsed,
        Rejected:      decision.Rejected,
        AssignmentID:  decision.AssignmentID,
        // Serializar CandidateEvaluations como JSON si se necesita detalle
        CandidatesJSON: marshalCandidates(decision.Trace.CandidateEvaluations),
    })
}
```

---

## 10. Recomendaciones de TraceID / CorrelationID

| Campo | Recomendación |
|---|---|
| TraceID | UUID v4 generado por el orquestador al recibir el evento de caja. Único por decisión. |
| CorrelationID | ID compartido por un grupo lógico: sesión de lote, turno, o request del cliente. |

**Uso en logging:**

```go
slog.Info("decision resolved",
    "trace_id", decision.Trace.TraceID,
    "correlation_id", decision.Trace.CorrelationID,
    "action", decision.Action,
    "exit_id", decision.ExitID,
)
```

**Propagación**: si el orquestador usa OpenTelemetry, el TraceID puede ser
el span ID de OTel para correlación directa en Grafana/Jaeger.

---

## 11. Manejo de Errores Go

```go
decision, err := engine.ResolveSorter(ctx, req)
if err != nil {
    switch {
    case errors.Is(err, sorter.ErrContextCanceled):
        // Operación cancelada por timeout o shutdown. Recircular la caja.
        return recirculate(box)
    case errors.Is(err, sorter.ErrMissingRequiredField):
        // Bug del orquestador: request mal construido. Loggear y alarmar.
        slog.Error("invalid request", "err", err, "trace_id", req.TraceID)
        return reject(box, "invalid_request")
    case errors.Is(err, sorter.ErrInvalidConfig):
        // Config corrupta. No debería pasar post-init. Alarma crítica.
        slog.Error("engine misconfigured", "err", err)
        return reject(box, "config_error")
    case errors.Is(err, sorter.ErrAmbiguousAssignment):
        // Asignaciones contradictorias de igual prioridad. Revisar config.
        slog.Warn("ambiguous assignment", "err", err, "trace_id", req.TraceID)
        return reject(box, "ambiguous")
    default:
        slog.Error("unexpected sorter error", "err", err)
        return reject(box, "unknown_error")
    }
}
// err == nil → usar decision normalmente.
```

---

## 12. Manejo de ActionReject

`ActionReject` con `error == nil` es una decisión operacional válida.

```go
decision, err := engine.ResolveSorter(ctx, req)
if err != nil {
    // ... manejar error técnico
}

if decision.Action == sorter.ActionReject {
    // La caja no tiene destino válido. Opciones del orquestador:
    // 1. Enviar a salida física de rechazo (ya hecho si RejectExitID está configurado).
    // 2. Recircular.
    // 3. Alarmar al operador.
    // 4. Guardar en auditoría para análisis posterior.
    slog.Info("box rejected",
        "box_id", req.Box.BoxID,
        "rule", decision.Trace.RuleApplied,
        "reason", decision.Trace.Reason,
    )
    return handleRejection(box, decision)
}

// Action == ActionRoute → ejecutar envío físico.
err = plc.SendToExit(ctx, decision.ExitID, box.BoxID)
```

---

## 13. Reglas para Replay/Debug

Para reproducir una decisión exacta offline:

1. **Guardar el snapshot completo**: request + config en el momento de decisión.
2. **EvalTime explícito**: nunca usar zero (que resuelve a `time.Now()`).
3. **Para round_robin**: guardar el valor del counter, o usar `least_loaded` para replay.
4. **Recrear Engine con misma config**: `NewEngine(savedConfig)`.
5. **Llamar con mismo request**: `engine.ResolveSorter(ctx, savedRequest)`.
6. **Verificar resultado**: debe coincidir con decisión original.

```go
// Replay conceptual:
engine, _ := sorter.NewEngine(savedConfig)
replayed, err := engine.ResolveSorter(context.Background(), savedRequest)
if replayed.ExitID != original.ExitID {
    slog.Warn("replay mismatch", "original", original.ExitID, "replayed", replayed.ExitID)
}
```

---

## 14. Ejemplo Conceptual de Integración Completa

Este ejemplo muestra cómo un orquestador de packing conecta las piezas.
**No es código ejecutable** — es un contrato de diseño.

```go
// orchestrator/handler.go (NO en dantrack-sorter)
package orchestrator

import (
    "context"
    "log/slog"

    sorter "github.com/dantrack/dantrack-sorter"
    "github.com/dantrack/dantrack-connect"
    "github.com/dantrack/dantrack-db"
)

type BoxHandler struct {
    engine *sorter.Engine
    plc    *dantrackconnect.Client
    db     *dantrackdb.Client
}

func (h *BoxHandler) HandleBoxEvent(ctx context.Context, event dantrackconnect.BoxEvent) error {
    // 1. Construir BoxContext desde evento hardware.
    box := sorter.BoxContext{
        BoxID:         event.BoxID,
        QR:            event.QR,
        SKU:           event.SKU,
        Variety:       event.Variety,
        Caliber:       event.Caliber,
        ReadStatus:    event.ScanResult,
        QualityStatus: event.QualityGrade,
        ErrorFlags:    event.ErrorFlags,
    }

    // 2. Leer estados actuales de exits.
    exitStates, err := h.plc.ReadExitStates(ctx)
    if err != nil {
        return fmt.Errorf("reading exit states: %w", err)
    }

    // 3. Cargar assignments vigentes.
    assignments, err := h.db.LoadActiveAssignments(ctx, event.SorterID)
    if err != nil {
        return fmt.Errorf("loading assignments: %w", err)
    }

    // 4. Resolver decisión.
    decision, err := h.engine.ResolveSorter(ctx, sorter.SorterRequest{
        TraceID:       generateTraceID(),
        CorrelationID: event.SessionID,
        Box:           box,
        Assignments:   mapAssignments(assignments),
        ExitStates:    mapExitStates(exitStates),
        EvalTime:      time.Now(),
    })
    if err != nil {
        slog.Error("sorter decision failed", "err", err, "box", box.BoxID)
        return h.plc.Recirculate(ctx, event.BoxID)
    }

    // 5. Ejecutar decisión.
    if decision.Action == sorter.ActionRoute {
        if err := h.plc.RouteToExit(ctx, decision.ExitID, event.BoxID); err != nil {
            return fmt.Errorf("routing to exit: %w", err)
        }
    }

    // 6. Guardar auditoría.
    h.db.SaveDecisionAudit(ctx, decision)

    return nil
}
```

**Nota**: `dantrack-connect` y `dantrack-db` son paquetes separados.
`dantrack-sorter` NO los importa. La integración ocurre en el orquestador.
