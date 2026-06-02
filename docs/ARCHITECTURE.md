# Arquitectura — dantrack-sorter

## 1. Propósito

`dantrack-sorter` es el motor de decisión puro del ecosistema DANTRACK.
Dado el contexto de una caja y la configuración de destinos, decide a qué
salida (sorter) o parque (presorter) debe ir la caja.

Es una librería Go sin efectos secundarios: no accede a base de datos,
no controla hardware, no genera UUIDs, no abre conexiones.

---

## 2. Qué Problema Resuelve

En una planta de packing industrial, las cajas deben ser clasificadas
y enviadas a destinos físicos (salidas de un sorter o parques de un presorter)
en base a reglas dinámicas configuradas por el negocio.

`dantrack-sorter` encapsula la lógica de clasificación de forma
determinista, trazable y testeable, separándola completamente de:

- La infraestructura de hardware (PLCs, scanners, cámaras).
- La persistencia (base de datos, archivos).
- El transporte (TCP, Modbus, OPC-UA, gRPC).

Esto permite:
- Replay de decisiones para auditoría.
- Testing sin hardware.
- Cambio de reglas sin reinicio de hardware.
- Simulación offline (dantrack-sim).

---

## 3. Qué NO Hace

| No hace | Responsable |
|---|---|
| Conectar con PLCs/hardware | `dantrack-connect` |
| Persistir datos/auditoría | `dantrack-db` |
| Generar IDs únicos | El orquestador |
| Abrir conexiones de red | El orquestador |
| Manejar concurrencia de hardware | El orquestador |
| Simular planta | `dantrack-sim` |
| Servir HTTP/gRPC | El orquestador |
| Manejar sesiones de usuario | El orquestador |

---

## 4. Arquitectura Conceptual

```text
┌─────────────────────────────────────────────────┐
│                  ORQUESTADOR                     │
│                                                  │
│  ┌────────────┐  ┌──────────────┐  ┌─────────┐ │
│  │dantrack-db │  │dantrack-sort │  │dantrack- │ │
│  │(persistenc)│  │(decisiones)  │  │connect   │ │
│  └─────┬──────┘  └──────┬───────┘  └─────┬───┘ │
│        │                 │                │      │
│    cargar cfg        decidir          ejecutar   │
│    guardar audit.    evaluar          leer HW    │
│                                                  │
└─────────────────────────────────────────────────┘
```

`dantrack-sorter` vive en el centro: recibe datos, devuelve decisiones.

---

## 5. Flujo General de Integración

```text
1. El orquestador carga configuración usando dantrack-db.
   → SorterConfig, PresorterConfig, Assignments.

2. El orquestador recibe un evento de caja usando dantrack-connect.
   → Scanner detecta QR, PLC reporta posición, cámara envía calidad.

3. El orquestador construye BoxContext con datos del evento.
   → Mapea datos hardware a campos semánticos.

4. El orquestador consulta estados actuales de exits/parks.
   → ExitStates, ParkStates desde dantrack-connect o dantrack-db.

5. El orquestador llama dantrack-sorter.
   → engine.ResolveSorter(ctx, req) o engine.ResolvePresorter(ctx, req).

6. dantrack-sorter devuelve una decisión con trazabilidad.
   → SorterDecision o PresorterDecision.

7. El orquestador ejecuta la decisión física usando dantrack-connect.
   → Envía comando al PLC: "enviar caja a exit_A".

8. El orquestador guarda auditoría usando dantrack-db.
   → Persiste DecisionTrace para replay y reporting.
```

---

## 6. Componentes Internos

### BoxContext

Snapshot de una caja en el momento de decisión. No es una entidad de
persistencia. Contiene: ID, QR, SKU, variedad, calibre, categoría,
cliente, tipo de paquete, destino, flujo, origen, tags, estados de
lectura/calidad, y flags de error.

### AssignmentSet

Conjunto de reglas de asignación que mapean condiciones de caja a
destinos. La librería recibe assignments ya cargados; no accede a DB.
Cada Assignment tiene: condiciones, prioridad, vigencia temporal,
target type y target ID.

### Engine

Struct central que contiene la configuración del sorter y/o presorter.
Se crea con `NewEngine(EngineConfig)`. Es seguro para uso concurrente.
El único estado mutable es el contador round-robin (sync/atomic).

### Sorter Decision Engine (`ResolveSorter`)

Evalúa assignments contra el BoxContext, encuentra el exit del match
con mayor prioridad que esté disponible, aplica fallback si es necesario,
y devuelve `SorterDecision`.

### Presorter Decision Engine (`ResolvePresorter`)

Filtra parks disponibles, aplica preferencia de error control si la caja
lo requiere, selecciona park por estrategia de balanceo, aplica fallback
a DefaultParkID si no hay candidatos, y devuelve `PresorterDecision`.

### Balance Strategies

- **least_loaded**: elige park con menor `CurrentLoad`. Desempata por index.
- **round_robin**: rotación atómica (sync/atomic). Sin mutex.
- **weighted**: elige park con mayor `Weight`. Desempata por index.

### DecisionTrace

Rastro completo de cómo se llegó a una decisión. Incluye TraceID,
CorrelationID, RuleApplied, Reason, DiagnosticMessage y
CandidateEvaluations (evaluación individual de cada candidato).

---

## 7. Diagrama — ResolveSorter

```text
ResolveSorter(ctx, SorterRequest)
│
├── ctx cancelado? ──────────────────────── → error: ErrContextCanceled
│
├── ValidateSorterRequest(req)? ─────────── → error: ErrMissingRequiredField
│
├── sorterCfg == nil? ──────────────────── → error: ErrInvalidConfig
│
├── EvaluateAssignments(box, assignments)
│   └── resultados ordenados por Priority desc
│
├── ¿Hay match?
│   ├── NO → resolveNoMatch:
│   │        ├── RejectExitID disponible? → ActionReject en exit físico
│   │        ├── DefaultExitID disponible? → ActionRoute con fallback
│   │        └── Ninguno → ActionReject sin exit
│   │
│   └── SÍ → checkAmbiguity:
│            ├── Ambigüedad irresoluble? → error: ErrAmbiguousAssignment
│            └── OK → iterar matches por prioridad:
│                     ├── Exit disponible? → ActionRoute
│                     ├── Siguiente match...
│                     ├── DefaultExitID disponible? → ActionRoute + FallbackUsed
│                     └── Ninguno → ActionReject
│
└── return SorterDecision
```

---

## 8. Diagrama — ResolvePresorter

```text
ResolvePresorter(ctx, PresorterRequest)
│
├── ctx cancelado? ──────────────────────── → error: ErrContextCanceled
│
├── ValidatePresorterRequest(req)? ──────── → error: ErrMissingRequiredField
│
├── presorterCfg == nil? ───────────────── → error: ErrInvalidConfig
│
├── buildParkStateIndex(req.ParkStates)
│
├── Para cada park en config:
│   └── isParkAvailable(cfg, state)?
│       ├── SÍ → candidato
│       └── NO → descartado con razón
│
├── ¿Caja requiere error control?
│   └── SÍ → filtrar candidatos con HasErrorControl
│            ├── Hay parks EC → usar solo esos
│            └── No hay parks EC → usar todos (FallbackUsed)
│
├── ¿Hay candidatos?
│   └── SÍ → selectByStrategy:
│            ├── least_loaded → menor CurrentLoad
│            ├── round_robin → rotación atómica
│            └── weighted → mayor Weight
│            └── return ActionRoute
│
├── DefaultParkID disponible? → ActionRoute + FallbackUsed
│
└── Ningún park → ActionReject
```

---

## 9. Trazabilidad

Toda decisión incluye `DecisionTrace` con:

- **TraceID**: copiado del request. El orquestador lo genera (UUID, request ID).
- **CorrelationID**: ID de correlación (sesión, lote, turno).
- **RuleApplied**: constante tipada que identifica la regla que produjo la decisión.
- **Reason**: descripción legible para humanos.
- **DiagnosticMessage**: información extra para debugging (solo en rejects/errores).
- **CandidateEvaluations**: evaluación de cada candidato (elegible/descartado/razón).

El orquestador debe persistir `DecisionTrace` completo para:
- Auditoría post-operacional.
- Replay de decisiones.
- Debugging de incidentes.
- Reporting de eficiencia.

---

## 10. Errores vs Decisiones Válidas

| Situación | Retorno |
|---|---|
| Context cancelado | `error` (ErrContextCanceled) |
| Request inválido | `error` (ErrMissingRequiredField) |
| Motor no configurado | `error` (ErrInvalidConfig) |
| Ambigüedad irresoluble | `error` (ErrAmbiguousAssignment) |
| No hay match → reject | `ActionReject`, error=nil |
| Exit no disponible → reject | `ActionReject`, error=nil |
| Park no disponible → reject | `ActionReject`, error=nil |

**Regla fundamental**: `ActionReject` NO es un error Go.
Es una decisión de negocio válida con trazabilidad completa.
El orquestador no debe tratar rejects como bugs.

---

## 11. Determinismo

`dantrack-sorter` es determinista dada la misma entrada:

- `EvaluateAssignments` es puro: misma box + assignments + evalTime = mismo resultado.
- `ResolveSorter` es puro: misma config + request = misma decisión.
- `ResolvePresorter` con `least_loaded` o `weighted` es determinista.
- `ResolvePresorter` con `round_robin` depende del estado del contador atómico.

Para replay exacto, el orquestador debe guardar:
1. La configuración exacta en el momento de la decisión.
2. El request completo (box, states, assignments, evalTime).
3. Para round_robin: el valor del contador (o usar `least_loaded` para replay).

---

## 12. Concurrencia

- `Engine` es seguro para uso concurrente desde múltiples goroutines.
- `ResolveSorter` es thread-safe (no muta estado).
- `ResolvePresorter` es thread-safe (solo `rrCounter` muta, via sync/atomic).
- `EvaluateAssignments` es una función pura sin estado.

El orquestador puede compartir un `*Engine` entre goroutines sin mutex.

---

## 13. Extensibilidad Futura

| Extensión | Cómo agregar |
|---|---|
| Nueva BalanceStrategy | Agregar constante + case en `selectByStrategy` |
| Nuevo operador de condición | Agregar constante + case en `evaluateCondition` |
| Nuevos campos en BoxContext | Agregar field + registrar en `resolveFieldValue` |
| Nueva Action | Agregar constante (ej: ActionRecirculate ya existe) |
| Nuevas Rules | Agregar constantes tipadas |
| Métricas | El orquestador instrumenta; la librería no importa Prometheus |
| Eventos | El orquestador emite eventos a partir de DecisionTrace |
| Persistencia | El orquestador serializa Decision/Trace a DB |

La librería crece sin romper contratos existentes gracias a:
- Tipos struct con campos nombrados (backward compatible).
- Constantes tipadas (compile-time safety).
- Sin interfaces implícitas que requieran implementar nuevos métodos.
