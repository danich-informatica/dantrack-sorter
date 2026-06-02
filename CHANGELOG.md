# Changelog

Todos los cambios notables de `dantrack-sorter` se documentan aquí.

Formato basado en [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).  
Este proyecto usa [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
