# Changelog

Todos los cambios notables de `dantrack-sorter` se documentan aquí.

Formato basado en [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).  
Este proyecto usa [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **Match helpers**: `Match`, `MatchEquals`, `MatchNotEquals`, `MatchIn`, `MatchNotIn`, `MatchExists`, `MatchNotExists` — shortcuts para construir `MatchCondition`.
- **Exit state helpers**: `ExitStates`, `ExitAvailable`, `ExitFull`, `ExitBlocked`, `ExitMaintenance`, `ExitError`, `ExitUnavailable`.
- **Park state helpers**: `ParkStates`, `ParkAvailable`, `ParkAvailableWithWindow`, `ParkFull`, `ParkBlocked`, `ParkMaintenance`, `ParkError`, `ParkUnavailable`.
- **BoxBuilder**: `NewBox(boxID).Variety("Fuji").Caliber("80").Build()` — builder fluido inmutable para `BoxContext`.
- **ErrorBox**: `ErrorBox(boxID, readStatus)` — shortcut para cajas con error.
- **AssignmentBuilder**: `NewAssignment(id, priority).MatchEquals(...).ToExits(...).Build()` — builder fluido inmutable para `Assignment`.
- 48 tests nuevos para helpers y builders.

---

## [v0.2.0] — 2026-06-03

### Added

- `AvailableSKUsPerExit` — vista pura de SKUs disponibles por salida de sorter.
- `AvailableSKUsPerPark` — vista pura de SKUs disponibles por parque de presorter.
- `ParkState.AccumulatedLoad` — carga acumulada en ventana de balance (no histórico total).
- `RulePresorterGlobalBalance` — regla cuando se aplica balanceo por carga acumulada.
- `selectGlobalBalanced` — selección de park por menor `AccumulatedLoad` con tie-break `CurrentLoad`.
- `hasAccumulatedLoad` — detección automática de presencia de carga acumulada.
- `PresorterConfig.ErrorParkID` — park fijo exclusivo para cajas error/no-read.
- `RulePresorterErrorFixedPark` — regla para routing a error park fijo.
- `RulePresorterPassErrorParkUnavailable` — regla para ActionPass cuando error park no disponible.
- `ActionPass` en Presorter: cuando error park no disponible, no accionar desvío.
- `SorterConfig.BalanceStrategy` — campo para seleccionar estrategia de balanceo multi-target.
- `BalanceLeastLoaded` soportado en sorter: elige el exit con menor `CurrentLoad`.
- `RuleSorterBalancedMultiTarget` — regla cuando se usa balanceo por carga.
- `ActionPass` se produce activamente cuando todos los targets del match están `Full`.
- `resolveBalancedMultiTarget` — método interno para balanceo least_loaded.
- `pickLeastLoaded`, `exitLoad`, `allConfiguredTargetsFull`, `exitUnavailableReason` helpers.
- `Assignment.TargetIDs []string` — soporte multi-target por asignación.
- `ActionPass` — acción para "no accionar desvío" cuando hay destino lógico pero sin capacidad.
- `RuleSorterPassAllTargetsFull` — regla para cuando todos los targets del match están llenos.
- `assignmentTargetIDs()` helper interno para resolución de targets efectivos.
- `ResolveSorter` ahora itera todos los `TargetIDs` de un match buscando el primero disponible.

### Changed

- Documentación de `AccumulatedLoad`: aclarada como ventana de balance, no histórico total.
- `selectByStrategy` default: si hay `AccumulatedLoad > 0`, usa balanceo global; sino `least_loaded`.
- Trazabilidad Presorter: `Reason` incluye tipo de balanceo y carga del park elegido.
- `ResolvePresorter`: si `ErrorParkID` configurado y caja es error/no-read, enruta exclusivamente sin fallback.
- Validación de `PresorterConfig`: `ErrorParkID` debe referenciar park existente si no está vacío.
- Validación de `SorterConfig`: solo acepta `""` y `BalanceLeastLoaded` como `BalanceStrategy`.
- `ResolveSorter` bifurca en step 12: first-available (legacy) vs least_loaded (balanced).
- Trazabilidad: `CandidateEvaluations` incluye carga de cada exit evaluado.
- Validación de `Assignment` acepta `TargetID` (legacy) o `TargetIDs` (multi-target).
- `checkAmbiguity` usa targets efectivos para comparación.

### Deprecated

- `Assignment.TargetID` — usar `TargetIDs` para asignaciones nuevas.

### Verified

- Integración externa aprobada (`prueba-librería`): 40 tests PASS, race-free, sin bugs.
- Import directo sin `replace` directive.

---

## [v0.1.0-rc2] — 2026-06-02

### Fixed

- Module path corregido de `github.com/dantrack/dantrack-sorter` a `github.com/danich-informatica/dantrack-sorter`.
- Consumidores ya no necesitan `replace` directive para usar la librería.

---

## [v0.1.0-rc1] — 2026-06-02

Primera versión release candidate de `dantrack-sorter`.

### Added

**Foundation (Iter 02)**
- Tipos de dominio: `BoxContext`, `Action`, `Rule`, `BalanceStrategy`, `MatchOperator`.
- Errores sentinel compatibles con `errors.Is`.
- Validación de configs y requests.

**Assignment Matching (Iter 03)**
- `EvaluateAssignments` — evaluación pura de asignaciones contra box context.
- Operadores: `equals`, `not_equals`, `in`, `not_in`, `exists`, `not_exists`.
- Soporte de tags (`tags.<key>`), vigencia temporal, prioridades.
- Trazabilidad completa con `ConditionEvaluation`.

**Sorter Decision Engine (Iter 04)**
- `NewEngine` — constructor con validación.
- `Engine.ResolveSorter` — decisión de exit con assignments, fallback y reject.
- `AmbiguityPolicy` — manejo de asignaciones ambiguas (`error` / `first_wins`).
- Fallback a `DefaultExitID` cuando exits matched no están disponibles.
- `ActionReject` como decisión válida (no error Go).

**Presorter Decision Engine (Iter 05)**
- `Engine.ResolvePresorter` — decisión de park con balanceo.
- Balance strategies: `least_loaded`, `round_robin`, `weighted`.
- Detección de error control (`ReadStatus`, `QualityStatus`, `ErrorFlags`).
- Preferencia de parks con `HasErrorControl`.
- Fallback a `DefaultParkID`.

**Examples & Documentation (Iter 06)**
- 4 examples ejecutables (`basic_sorter`, `basic_presorter`, `error_control`, `fallbacks`).
- 3 godoc examples validados por `go test`.
- `docs/ARCHITECTURE.md` — arquitectura completa.
- `docs/INTEGRATION_CONTRACTS.md` — contratos de integración.

**Release Candidate (Iter 07)**
- API review (`docs/PUBLIC_API_REVIEW.md`).
- Edge case tests: nil maps, nil slices, empty strategies, unknown parks/exits.
- Release documentation (`CHANGELOG.md`, `docs/VERSIONING.md`, `docs/RELEASE_CHECKLIST.md`).

### Known Limitations

- Round-robin counter no persiste entre reinicios del proceso.
- `weighted` strategy es determinista (siempre elige el de mayor peso); no usa distribución proporcional.
- `PresorterConfig.ErrorControlFlag` definido pero no usado internamente (reservado).
- `ActionRecirculate`, `ActionError`, `ActionNoop` definidos pero no producidos por el motor.
- Sin adapters de integración — la responsabilidad es del orquestador.
- Sin métricas/observabilidad integrada.
- Sin validación cruzada entre ParkStates y Parks configurados.

### Stats

- **Tests**: 156 (148 + 8 edge cases)
- **Coverage**: 95.3%
- **Race detector**: clean
- **Dependencies**: none (stdlib only)
