# Iteración 01 — Propuesta de Arquitectura: dantrack-sorter

**Autor:** Arquitectura DANTRACK  
**Fecha:** 2026-06-01  
**Estado:** Aprobada — Iteración 02 Foundation implementada (2026-06-01)  
**Módulo:** `github.com/dantrack/dantrack-sorter`  
**Package:** `package sorter`

---

## Iteración 02 — Foundation implementada

**Fecha:** 2026-06-01  
**Resultado:** `go fmt`, `go vet`, `go test ./...` — 33/33 tests PASS

**Archivos creados:**
- `types.go` — todos los modelos de dominio
- `errors.go` — errores sentinel distinguibles
- `validation.go` — `ValidateConfig`, `ValidatePresorterConfig`, `ValidateSorterConfig`, `ValidatePresorterRequest`, `ValidateSorterRequest`
- `validation_test.go` — 33 tests con stdlib pura
- `README.md` — actualizado con estado real

**Decisiones aplicadas en implementación:**

| Decisión | Implementado |
|---|---|
| No generar UUID automáticamente | `TraceID` y `CorrelationID` vienen del orquestador; se copian a la decisión |
| `EvalTime time.Time` (no puntero) | Confirmado; zero value → `time.Now()` en iteraciones futuras |
| `ActionReject` es decisión válida | Los errores Go solo para config/request inválido o ambigüedad |
| `RuleApplied` es tipo `Rule` con constantes | Definidas en `types.go` |
| Trazabilidad estructurada | `DecisionTrace`, `CandidateEvaluation`, `ConditionEvaluation` como structs tipados |
| `slices.Contains` para validar enums | Sin reflect, sin bucles manuales |

**Corrección vs propuesta original:**  
El documento original mencionaba generar `DecisionID` UUID v4. Eso fue descartado. La librería no genera IDs; el orquestador los pasa como `TraceID`/`CorrelationID`.

---

## 1. Resumen Ejecutivo

`dantrack-sorter` es una librería Go de dominio puro que resuelve decisiones de ruteo industrial (Presorter y Sorter) dentro del ecosistema DANTRACK. No controla hardware, no abre conexiones, no persiste datos. Es un motor de decisión: recibe contexto, evalúa reglas y devuelve una decisión trazable.

El orquestador de cada packing es quien integra esta librería con `dantrack-connect`, `dantrack-db` y el resto del ecosistema.

---

## 2. Nombre de la Librería

**`dantrack-sorter`**

Justificación: el nombre comunica inmediatamente que es el componente de decisión de sorting dentro de DANTRACK. "Sorter" en contexto industrial ya implica tanto presorter como sorter final. No se usa "routing" porque podría confundirse con routing HTTP o de red.

---

## 3. Alcance Exacto

Motor de decisión para:

- **Presorter:** decidir a qué parque/camino enviar una caja.
- **Sorter:** decidir a qué salida enviar una caja según asignaciones vigentes.

---

## 4. Qué Queda Dentro

| Responsabilidad | Descripción |
|---|---|
| Resolución Presorter | Decidir parque destino con balanceo y fallback |
| Resolución Sorter | Decidir salida destino según asignaciones |
| Matching de asignaciones | Evaluar condiciones contra contexto de caja |
| Validación de disponibilidad | Usar estados entregados para descartar destinos no disponibles |
| Fallback | Aplicar estrategias alternativas cuando el destino primario no está disponible |
| Balanceo | Round-robin, least-loaded, weighted entre destinos válidos |
| Trazabilidad | Cada decisión incluye razón, regla, candidatos evaluados y descartados |
| Validación de config | Validar configuración antes de operar |
| Modelos de dominio | BoxContext, Assignment, Decision, Config |

---

## 5. Qué Queda Fuera

| Fuera de alcance | Razón |
|---|---|
| Conexiones hardware (PLC, Modbus, OPC-UA, TCP) | Responsabilidad de `dantrack-connect` |
| Conexiones a base de datos | Responsabilidad de `dantrack-db` |
| Lectura de asignaciones desde DB | El orquestador las carga y las pasa |
| Ejecución física del desvío | El orquestador ejecuta vía `dantrack-connect` |
| Lógica específica de un packing | Viene por configuración o request |
| Simulación | Responsabilidad de `dantrack-sim` |
| Persistencia de auditoría | El orquestador decide dónde guardar |
| Cálculo de señales de hardware | Solo consume estados que le entregan |

---

## 6. Principios de Diseño

1. **Puro:** sin side effects, sin I/O, sin estado global.
2. **Determinista:** mismos inputs + mismo estado interno del Engine → misma decisión. La única excepción es `round_robin`, que mantiene un contador interno. Estrategias como `least_loaded` son deterministas dado el mismo `ParkState`/`ExitState`.
3. **Testeable:** 100% testeable sin mocks de infraestructura.
4. **Trazable:** toda decisión explica su razonamiento completo con estructuras tipadas.
5. **Configurable:** todo lo específico viene por config o request.
6. **Mínimo:** no sobreingenierizar; agregar complejidad solo cuando se justifique.
7. **Extensible sin modificar:** nuevas estrategias de balanceo o matching vía interfaces, no flags.
8. **Context-aware:** `context.Context` en toda API pública.
9. **Sin generación implícita:** la librería no genera UUIDs, timestamps ni datos no solicitados por defecto.

---

## 7. Modelos Principales

### 7.1 BoxContext

Representa el contexto de una caja para decisión de routing. No es una entidad de persistencia.

```go
type BoxContext struct {
    BoxID         string
    QR            string
    SKU           string
    Variety       string
    Caliber       string
    Category      string
    Client        string
    PackageType   string
    Destination   string
    Flow          string
    Source        string
    Tags          map[string]string
    ReadStatus    string
    QualityStatus string
    ErrorFlags    []string
    Timestamp     time.Time
}
```

**Nota:** se usa `ErrorFlags []string` en lugar de un solo string para permitir múltiples flags sin parseo.

### 7.2 Presorter

```go
type PresorterConfig struct {
    Parks           []ParkConfig
    BalanceStrategy BalanceStrategy
    DefaultParkID   string // fallback final si existe
}

type ParkConfig struct {
    ParkID              string
    Enabled             bool
    HasErrorControl     bool
    Priority            int
    MaxLoad             int
    Tags                map[string]string
}

type ParkState struct {
    ParkID      string
    Available   bool
    Blocked     bool
    Full        bool
    HasError    bool
    CurrentLoad int
}

type PresorterRequest struct {
    TraceID    string // entregado por el orquestador para correlación
    Box        BoxContext
    ParkStates []ParkState
    EvalTime   time.Time // si zero, usa time.Now()
}

type PresorterDecision struct {
    TraceID               string
    ParkID                string
    Action                Action // Route, Reject, Error
    Reason                string
    RuleApplied           Rule
    BalanceStrategy       BalanceStrategy
    FallbackUsed          bool
    ErrorControlRequired  bool
    ErrorControlAvailable bool
    Candidates            []CandidateEvaluation
    EvalTime              time.Time
}

type CandidateEvaluation struct {
    TargetType     string // "park", "exit"
    TargetID       string
    Eligible       bool
    RejectedReason string
    Rule           Rule
    AssignmentID   string
    Conditions     []ConditionEvaluation
}

type ConditionEvaluation struct {
    Field    string
    Operator MatchOperator
    Expected []string
    Actual   string
    Matched  bool
}
```

### 7.3 Sorter

```go
type SorterConfig struct {
    SorterID    string
    Exits       []ExitConfig
    DefaultExit string // salida de rechazo/default
}

type ExitConfig struct {
    ExitID   string
    Enabled  bool
    Priority int
    Capacity int
    Tags     map[string]string
}

type ExitState struct {
    ExitID      string
    Available   bool
    Blocked     bool
    Full        bool
    HasError    bool
    Maintenance bool
    CurrentLoad int
}

type SorterRequest struct {
    TraceID     string // entregado por el orquestador para correlación
    Box         BoxContext
    Assignments []Assignment
    ExitStates  []ExitState
    EvalTime    time.Time // si zero, usa time.Now()
}

type SorterDecision struct {
    TraceID           string
    SorterID          string
    ExitID            string
    Action            Action
    AssignmentID      string
    Reason            string
    RuleApplied       Rule
    FallbackUsed      bool
    Rejected          bool
    DiagnosticMessage string
    Candidates        []CandidateEvaluation
    EvalTime          time.Time
}
```

### 7.4 Assignments

```go
type Assignment struct {
    ID         string
    Enabled    bool
    Priority   int
    Conditions []MatchCondition
    TargetType string // "exit", "park"
    TargetID   string
    ValidFrom  time.Time
    ValidTo    time.Time
    Metadata   map[string]string
}

type MatchCondition struct {
    Field    string        // "sku", "variety", "caliber", "tags.lote", etc.
    Operator MatchOperator // Equals, NotEquals, In, NotIn, Exists
    Values   []string
}

type MatchOperator string

const (
    OpEquals    MatchOperator = "equals"
    OpNotEquals MatchOperator = "not_equals"
    OpIn        MatchOperator = "in"
    OpNotIn     MatchOperator = "not_in"
    OpExists    MatchOperator = "exists"
)
```

### 7.5 Tipos Compartidos

```go
type Action string

const (
    ActionRoute  Action = "route"
    ActionReject Action = "reject"
    ActionError  Action = "error"
)

type BalanceStrategy string

const (
    BalanceLeastLoaded BalanceStrategy = "least_loaded" // recomendada para primera integración
    BalanceRoundRobin  BalanceStrategy = "round_robin"  // requiere estado interno, safe para concurrencia
    BalanceWeighted    BalanceStrategy = "weighted"
)

type Rule string

const (
    // Presorter rules
    RulePresorterErrorControlPreferred Rule = "presorter_error_control_preferred"
    RulePresorterLeastLoaded           Rule = "presorter_least_loaded"
    RulePresorterRoundRobin            Rule = "presorter_round_robin"
    RulePresorterFallbackDefault       Rule = "presorter_fallback_default"
    RulePresorterRejectNoPark          Rule = "presorter_reject_no_park"

    // Sorter rules
    RuleSorterAssignmentMatched    Rule = "sorter_assignment_matched"
    RuleSorterFallbackDefaultExit  Rule = "sorter_fallback_default_exit"
    RuleSorterRejectNoMatch        Rule = "sorter_reject_no_match"
    RuleSorterRejectNoAvailableExit Rule = "sorter_reject_no_available_exit"
)
```

---

## 8. API Pública Propuesta

```go
package sorter

// Engine es el motor de decisión principal.
type Engine struct { ... }

// NewEngine crea un motor con la configuración dada.
func NewEngine(cfg EngineConfig) (*Engine, error)

// EngineConfig agrupa configuraciones de presorter y sorter.
type EngineConfig struct {
    Presorter *PresorterConfig
    Sorter    *SorterConfig
}

// ResolvePresorter decide a qué parque enviar la caja.
func (e *Engine) ResolvePresorter(ctx context.Context, req PresorterRequest) (PresorterDecision, error)

// ResolveSorter decide a qué salida enviar la caja.
func (e *Engine) ResolveSorter(ctx context.Context, req SorterRequest) (SorterDecision, error)

// ValidateConfig valida la configuración sin crear un engine.
func ValidateConfig(cfg EngineConfig) error

// EvaluateAssignments evalúa qué asignaciones aplican a una caja (útil para diagnóstico).
func EvaluateAssignments(box BoxContext, assignments []Assignment) []AssignmentMatch

// AssignmentMatch resultado de evaluación de una asignación contra una caja.
type AssignmentMatch struct {
    Assignment Assignment
    Matched    bool
    Reason     string
}
```

**Decisión de diseño:** un solo `Engine` que agrupa presorter y sorter. Si un packing solo necesita sorter, pasa `Presorter: nil`. Esto evita proliferación de constructores sin agregar complejidad.

---

## 9. Estructura de Repositorio Recomendada

```
dantrack-sorter/
├── go.mod
├── README.md
├── docs/
│   ├── ARCHITECTURE.md
│   └── ITERATION_01_PROPOSAL.md
├── engine.go            // Engine, NewEngine, EngineConfig
├── presorter.go         // ResolvePresorter, lógica de presorter
├── sorter.go            // ResolveSorter, lógica de sorter
├── assignment.go        // Matching de asignaciones, EvaluateAssignments
├── balance.go           // Estrategias de balanceo
├── types.go             // Todos los modelos/tipos
├── errors.go            // Errores de dominio
├── validation.go        // ValidateConfig y validaciones internas
├── engine_test.go
├── presorter_test.go
├── sorter_test.go
├── assignment_test.go
├── balance_test.go
├── validation_test.go
└── examples/
    ├── basic_presorter/
    │   └── main.go
    └── basic_sorter/
        └── main.go
```

**Decisión:** paquete plano en raíz (sin `pkg/routing/`). Razón:

- Es una librería de dominio con un solo paquete público.
- `pkg/` no agrega valor en una librería pura.
- Import path queda limpio: `import "github.com/dantrack/dantrack-sorter"`.
- Si en el futuro se necesitan sub-paquetes (ej: `strategy`), se agregan sin romper la API.

Si se prefiere un nombre de paquete más corto que `dantrack-sorter` para el import, el paquete Go se declara como `package sorter`.

---

## 10. Estrategia de Matching/Asignaciones

### Flujo de evaluación:

1. Filtrar asignaciones habilitadas (`Enabled == true`).
2. Filtrar por vigencia temporal (`ValidFrom` / `ValidTo` contra `EvalTime` del request; si zero, `time.Now()`).
3. Evaluar condiciones de cada asignación contra `BoxContext`.
4. Ordenar matches por prioridad (`Priority`).
5. Verificar disponibilidad del target.
6. Si target no disponible, intentar siguiente match.
7. Si no hay match, aplicar asignación default/fallback si existe.
8. Si no hay nada, devolver `ActionReject` o `ActionError`.

### Resolución de campos:

Los campos de `BoxContext` se acceden por nombre string en `MatchCondition.Field`:

- Campos directos: `"sku"`, `"variety"`, `"caliber"`, `"category"`, `"client"`, `"package_type"`, `"destination"`, `"flow"`, `"source"`, `"read_status"`, `"quality_status"`.
- Tags: `"tags.nombre_del_tag"`.

Implementación interna: un resolver simple con switch o map de accessors. No reflection.

### Prioridad vs ambigüedad:

- Si dos asignaciones con **misma prioridad** hacen match al mismo target → no hay ambigüedad, es el mismo resultado.
- Si dos asignaciones con **misma prioridad** hacen match a **targets diferentes** → `ErrAmbiguousAssignment` (configurable: puede ser error o elegir primera).

---

## 11. Estrategia de Fallback

### Presorter:

1. Evaluar parques candidatos (habilitados, disponibles).
2. Si la caja requiere error control → filtrar parques con error control.
3. Si hay múltiples candidatos → aplicar balance strategy.
4. Si el candidato primario no está disponible → siguiente candidato.
5. Si ningún parque disponible → `DefaultParkID` si existe.
6. Si no hay default → `ActionReject` con `ErrNoAvailablePark`.

### Sorter:

1. Evaluar asignaciones → obtener target exit.
2. Verificar disponibilidad del exit.
3. Si exit no disponible → siguiente asignación por prioridad.
4. Si no hay exit disponible → `DefaultExit` (reject/recirculación).
5. Si no hay default → `ActionReject` con `ErrNoAvailableExit`.

**Principio:** fallback es explícito en la decisión (`FallbackUsed: true`) para trazabilidad.

---

## 12. Estrategia de Trazabilidad/Auditoría

Cada decisión incluye:

| Campo | Propósito |
|---|---|
| `TraceID` | ID de correlación entregado por el orquestador |
| `Action` | Qué decidió (route/reject/error) |
| `Reason` | Texto humano explicando por qué |
| `RuleApplied` | Constante tipada de la regla que produjo la decisión |
| `AssignmentID` | Qué asignación se aplicó (sorter) |
| `FallbackUsed` | Si la decisión es resultado de fallback |
| `Candidates` | Lista tipada de todos los candidatos evaluados con condiciones |
| `EvalTime` | Momento usado para evaluación temporal |

La trazabilidad **no es solo texto**. `CandidateEvaluation` y `ConditionEvaluation` son estructuras tipadas que permiten reconstruir la lógica completa de la decisión programáticamente.

La librería **no persiste** auditoría. Devuelve la decisión completa al orquestador, quien decide si guardarla en DB, log, o ambos.

**TraceID:** lo entrega el orquestador en el request. La librería lo copia a la decisión. Si viene vacío, la decisión se devuelve sin ID — no se genera UUID automáticamente.

---

## 13. Estrategia de Errores

```go
var (
    ErrNoAssignmentMatch   = errors.New("no assignment matches box context")
    ErrNoAvailableExit     = errors.New("no available exit for matched assignment")
    ErrNoAvailablePark     = errors.New("no available park for presorter")
    ErrInvalidConfig       = errors.New("invalid engine configuration")
    ErrInvalidRequest      = errors.New("invalid request")
    ErrAmbiguousAssignment = errors.New("ambiguous assignment: multiple matches with same priority")
    ErrMissingRequiredField = errors.New("missing required field")
)
```

**Patrón:** errores sentinel + wrapping con `fmt.Errorf("...: %w", err)` para contexto adicional. El caller usa `errors.Is()` para distinguir.

**Regla crítica: errores vs decisiones válidas:**

- `ActionReject` es una **decisión válida**, no un error técnico. Significa "no hay destino disponible" pero la librería funcionó correctamente.
- Los errores Go (`error` return) se reservan exclusivamente para:
  - Config inválida (`ErrInvalidConfig`)
  - Request inválido (`ErrInvalidRequest`, `ErrMissingRequiredField`)
  - Contexto cancelado (`ctx.Err()`)
  - Ambigüedad no resoluble (`ErrAmbiguousAssignment`)
- No tener salida válida produce `ActionReject` con razón trazable, **no** un `error`.
- Solo se devuelve `error` si la configuración no permite tomar ninguna decisión válida (ni siquiera reject).

---

## 14. Estrategia de Tests

### Niveles:

1. **Unit tests por componente:**
   - `assignment_test.go` — matching de condiciones individuales.
   - `balance_test.go` — estrategias de balanceo.
   - `presorter_test.go` — decisiones de presorter con table-driven tests.
   - `sorter_test.go` — decisiones de sorter con table-driven tests.
   - `validation_test.go` — validación de config.

2. **Integration tests (en el mismo paquete):**
   - Flujos completos request → decision.
   - Escenarios de fallback encadenados.
   - Edge cases: config vacía, sin asignaciones, todos bloqueados.

3. **Example tests:**
   - `Example_presorterBasic`
   - `Example_sorterWithFallback`

### Principios:

- Table-driven tests para todas las combinaciones.
- No mocks necesarios (librería pura).
- Fixtures como funciones helper que construyen configs/requests de test.
- Zero dependencias externas para tests (ni testify si no se justifica).
- Si se usa testify, solo `assert` y `require`.

### Cobertura objetivo:

- 90%+ en matching y decisión.
- Todos los paths de fallback cubiertos.
- Todos los errores sentinel testeados.

---

## 15. Riesgos Técnicos y Conceptuales

| # | Riesgo | Impacto | Mitigación |
|---|---|---|---|
| 1 | **Balanceo round-robin requiere estado mutable** | El Engine deja de ser puramente stateless | Estado interno mínimo con `atomic.Uint64`; documentar que RR no es determinista entre llamadas; recomendar `least_loaded` como default |
| 2 | **Matching genérico puede volverse lento con muchas asignaciones** | Latencia en packing de alto volumen | Pre-indexar asignaciones por campo cuando se cargan; benchmark temprano |
| 3 | **El nombre "Sorter" puede confundirse con sorting algorithms** | Confusión en devs nuevos | README claro; el package statement `package sorter` es industrial |
| 4 | **Temporal validity de asignaciones depende de clock** | En tests y replay, `time.Now()` no es controlable | `EvalTime time.Time` en request; si zero → `time.Now()`; en tests siempre pasar valor explícito |
| 5 | **Scope creep:** tentación de agregar routing de otros equipos | Librería se infla | Mantener scope estricto: presorter + sorter. Si aparece otro equipo, evaluar si es un nuevo paquete |
| 6 | **Ambigüedad en prioridad de asignaciones** | Comportamiento impredecible | Política explícita configurable: error o first-wins |
| 7 | **Concurrencia en round-robin** | Data race si Engine se comparte | `atomic.Uint64` para contador; tests con `-race` obligatorios |

---

## 16. Decisiones a Tomar Antes de Implementar

| # | Decisión | Opciones | Resolución |
|---|---|---|---|
| 1 | ¿Paquete plano o `pkg/sorter/`? | Plano en raíz / subdirectorio | **Plano en raíz** — aprobado |
| 2 | ¿`package sorter` o `package dantracksorter`? | `sorter` / `dantracksorter` | **`package sorter`** — aprobado |
| 3 | ¿Ambigüedad de asignaciones es error o first-wins? | Error / First-wins / Configurable | **Configurable** con default error |
| 4 | ¿Round-robin con estado en Engine o stateless? | Interno / Externo | **Interno**, safe para concurrencia, documentar reset en restart |
| 5 | ¿`time.Now()` interno o clock inyectable? | Interno / Interface / Campo en request | **`EvalTime time.Time` en request**; si zero value → `time.Now()` |
| 6 | ¿Testify o stdlib pura? | testify / stdlib | **stdlib + helpers propios** para empezar |
| 7 | ¿DecisionID/UUID generado por la librería? | Sí / No / Opcional | **No** — el orquestador pasa `TraceID`; la librería lo copia, no genera |
| 8 | ¿Tags en BoxContext como `map[string]string` o tipo propio? | map / tipo | **`map[string]string`** — simple, suficiente |
| 9 | ¿Estrategia de balanceo default? | round_robin / least_loaded | **`least_loaded`** como recomendada para primera integración |

---

## 17. Plan de Implementación por Iteraciones

### Iteración 02 — Foundation

- `types.go` — todos los modelos.
- `errors.go` — errores sentinel.
- `validation.go` — validación de config y request.
- Tests de validación.

### Iteración 03 — Assignment Matching

- `assignment.go` — evaluación de condiciones.
- Resolver de campos de BoxContext.
- Tests exhaustivos de matching (table-driven con todas las combinaciones de operadores).

### Iteración 04 — Sorter

- `sorter.go` — `ResolveSorter`.
- `engine.go` — estructura Engine + NewEngine.
- Tests de decisión de sorter: match, fallback, reject, error.

### Iteración 05 — Presorter

- `presorter.go` — `ResolvePresorter`.
- `balance.go` — estrategias de balanceo.
- Tests de decisión de presorter: balanceo, fallback, error control.

### Iteración 06 — Polish & Examples

- `examples/` — ejemplos ejecutables.
- `docs/ARCHITECTURE.md` — documentación final.
- README completo.
- Benchmark básico de matching con N asignaciones.
- Review de API pública, locks de contrato.

### Iteración 07 — Integración

- Integración con orquestador real.
- Ajustes de API según feedback de integración.
- Tests de integración con datos reales (anonimizados).

---

## Observaciones Críticas del Arquitecto

1. **El diseño es sólido.** La separación motor/orquestador es correcta y escala bien.

2. **`BoxContext` no debería llamarse `Box`.** "Box" es demasiado genérico y podría chocar con otros modelos. `BoxContext` es correcto — comunica que es el contexto de una caja para decisión, no la entidad caja completa.

3. **No agregar `confidence` en la decisión de presorter.** No hay un modelo probabilístico aquí — las decisiones son deterministas basadas en reglas. Un campo "confidence" sugiere incertidumbre que no existe. Se reemplaza por `Action` que es explícito: `route`, `reject`, `error`.

4. **No crear interfaz `Router` prematuramente.** No hay segundo implementador hoy. Si aparece, se extrae la interfaz entonces. Devolver struct concreto.

5. **El campo `RuleApplied` es tipo `Rule` con constantes.** Esto permite filtrar/agrupar en auditoría sin parsear strings.

6. **El nombre `dantrack-sorter` es correcto.** Más específico que "routing", más claro en contexto industrial, se entiende al tiro.

---

## Iteration 01.1 Adjustments

Ajustes aplicados sobre la propuesta original aprobada:

### A. TraceID — no generar UUID internamente

- **Antes:** la librería generaba `DecisionID` UUID v4 automáticamente.
- **Ahora:** el orquestador pasa `TraceID string` en el request. La librería lo copia a la decisión.
- Si viene vacío, la decisión se devuelve sin ID. No hay generación implícita.
- **Razón:** mantener determinismo; la librería no debe tener side effects ni dependencias de generación de IDs.

### B. Determinismo reforzado

- La librería es determinista siempre que: inputs iguales + estado interno del Engine igual → misma decisión.
- `least_loaded` es determinista dado el mismo `ParkState`/`ExitState`.
- `round_robin` introduce estado mutable (contador). Se documenta explícitamente como excepción.
- `EvalTime time.Time` en requests permite controlar el tiempo de evaluación en tests y replay.

### C. Round-robin con cuidado

- `least_loaded` es la estrategia recomendada para primera integración real.
- `round_robin` soportado con estado interno mínimo (`atomic.Uint64`).
- Si el Engine es compartido entre goroutines, round-robin es safe para concurrencia.
- El contador se reinicia si el proceso reinicia (no persiste estado). Si se requiere persistencia, el orquestador debe manejarlo.
- Tests de concurrencia obligatorios si se implementa round-robin.

### D. EvalTime como value type

- `EvalTime time.Time` (no puntero).
- Si zero value (`time.Time{}`), la lógica interna usa `time.Now()`.
- En tests y replay, enviar siempre un `EvalTime` explícito para determinismo.

### E. Errores vs decisiones

- `ActionReject` = decisión válida, no error Go.
- `error` return = config/request inválido, contexto cancelado, ambigüedad no resoluble.
- "No hay salida disponible" → `ActionReject` con razón trazable.
- Solo `error` si la librería no puede tomar ninguna decisión (ni siquiera reject).

### F. RuleApplied tipado

- Tipo `Rule string` con constantes predefinidas.
- Permite filtrar/agrupar decisiones en auditoría sin parsear strings.
- Extensible: nuevas reglas se agregan como constantes.

### G. Trazabilidad estructurada

- `CandidateEvaluation` incluye `TargetType`, `TargetID`, `Eligible`, `RejectedReason`, `Rule`, `AssignmentID`, `Conditions`.
- `ConditionEvaluation` incluye `Field`, `Operator`, `Expected`, `Actual`, `Matched`.
- No es solo texto — es reconstruible programáticamente.

### H. Alcance primera versión confirmado

Solo: Presorter, Sorter, Assignment matching, Fallback, Balanceo básico, Trazabilidad, Validación, Tests.
No: DB, hardware, simulador, orquestador, examples complejos.

---

## Next Agent Instructions

Propuesta aprobada con ajustes 01.1. La siguiente iteración implementa Foundation.

### Iteración 02: Foundation

**Objetivo:** modelos, errores y validaciones listos. Sin lógica de decisión.

**Archivos a crear:**

1. `types.go` — todos los modelos definidos en sección 7 (actualizada con ajustes 01.1):
   - `BoxContext`, `PresorterConfig`, `ParkConfig`, `ParkState`
   - `PresorterRequest` (con `TraceID` y `EvalTime time.Time`)
   - `PresorterDecision` (con `TraceID`, `RuleApplied Rule`, `EvalTime`)
   - `SorterConfig`, `ExitConfig`, `ExitState`
   - `SorterRequest` (con `TraceID` y `EvalTime time.Time`)
   - `SorterDecision` (con `TraceID`, `RuleApplied Rule`, `EvalTime`)
   - `Assignment`, `MatchCondition`, `MatchOperator`
   - `CandidateEvaluation` (con `TargetType`, `TargetID`, `Conditions`)
   - `ConditionEvaluation` (con `Field`, `Operator`, `Expected`, `Actual`, `Matched`)
   - `Action`, `BalanceStrategy`, `Rule` con todas las constantes
   - `EngineConfig`

2. `errors.go` — errores sentinel:
   - `ErrNoAssignmentMatch`, `ErrNoAvailableExit`, `ErrNoAvailablePark`
   - `ErrInvalidConfig`, `ErrInvalidRequest`
   - `ErrAmbiguousAssignment`, `ErrMissingRequiredField`

3. `validation.go` — `ValidateConfig(cfg EngineConfig) error` y helpers internos:
   - Validar que al menos una config (Presorter o Sorter) esté presente.
   - Validar que parks/exits tengan IDs no vacíos y únicos.
   - Validar que BalanceStrategy sea valor conocido.
   - Validar que DefaultParkID/DefaultExit referencien IDs existentes (si se especifican).

4. `validation_test.go` — table-driven tests para validación:
   - Config válida.
   - Config nil ambas.
   - Parks con IDs duplicados.
   - Exit con ID vacío.
   - BalanceStrategy inválida.
   - Default apuntando a ID inexistente.

**NO implementar todavía:**
- `engine.go` (solo placeholder mínimo si se necesita para compilar)
- `ResolveSorter` / `ResolvePresorter`
- `assignment.go` / matching completo
- `balance.go` / balanceo real
- `examples/`

**Reglas de implementación:**
- `package sorter`
- `EvalTime time.Time` (no puntero); si zero value, la lógica usará `time.Now()` (pero eso se implementa en iteración posterior).
- `TraceID string` en requests; se copia a decisions. Si vacío, se deja vacío.
- `RuleApplied` es tipo `Rule` con constantes definidas.
- `CandidateEvaluation` y `ConditionEvaluation` son structs tipados.
- Zero dependencias externas (solo stdlib).
- `go build ./...` y `go test ./...` deben pasar.
- `go vet ./...` sin warnings.

**Criterio de éxito:** compilación limpia, tests verdes, `go vet` limpio, modelos claros y listos para ser usados por las iteraciones siguientes.

---

## Open Questions

1. **¿`TraceID` o `CorrelationID` como nombre final del campo de correlación?**
   - Propuesta actual usa `TraceID`. Alternativa: `CorrelationID`.
   - `TraceID` es más corto y alineado con OpenTelemetry.
   - Decidir antes de implementar tipos.

2. **¿Round-robin se acepta con estado interno o se prefiere manejar estado fuera?**
   - Propuesta actual: estado interno en Engine, safe para concurrencia (`atomic.Uint64`).
   - Alternativa: el orquestador pasa un counter en el request.
   - Implicación: si interno, round-robin se reinicia cuando el proceso reinicia.

3. **¿"No available exit" debe ser siempre `ActionReject` o error configurable?**
   - Propuesta actual: `ActionReject` con razón trazable (no es error Go).
   - Alternativa: configurable para devolver `error` en ciertos contextos.

4. **¿Existe `ActionRecirculate` como concepto distinto de `ActionReject`?**
   - Si la caja puede volver a pasar por el sorter, necesita una acción diferenciada.
   - Si se confirma, agregar `ActionRecirculate Action = "recirculate"`.

5. **¿Una asignación puede apuntar a múltiples exits?**
   - Si sí: `TargetID` pasa a ser `TargetIDs []string` + estrategia de selección.
   - Si no: se mantiene `TargetID string` simple.

6. **¿Qué campo exacto define que una caja requiere control de errores?**
   - ¿Flag específico en `ErrorFlags`? ¿Tag? ¿`QualityStatus`?
   - Propuesta: config del Engine define qué flag/tag implica "requiere error control".

7. **¿Cuántas asignaciones simultáneas se esperan en un sorter típico?**
   - Si < 100: evaluación lineal es suficiente.
   - Si > 1000: conviene pre-indexar en iteración futura.

8. **¿Se necesita soporte para "asignación temporal" (override manual por operador)?**
   - Se resuelve con prioridad alta en la asignación. No requiere lógica especial.
