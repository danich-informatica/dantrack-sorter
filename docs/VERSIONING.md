# Versioning — dantrack-sorter

## Versión Actual

**v0.1.0-rc1** — Release Candidate.

---

## Qué se Considera API Estable

Los siguientes elementos son estables a partir de v0.1.0 y no cambiarán
de forma incompatible sin un bump de major version:

### Estable (no romper)

- `NewEngine(EngineConfig) (*Engine, error)` — firma del constructor.
- `Engine.ResolveSorter(ctx, SorterRequest) (SorterDecision, error)` — firma.
- `Engine.ResolvePresorter(ctx, PresorterRequest) (PresorterDecision, error)` — firma.
- `EvaluateAssignments(BoxContext, AssignmentSet, time.Time) []AssignmentMatch` — firma.
- Todas las funciones `Validate*` — firmas.
- Errores sentinel — nombres y semántica con `errors.Is`.
- Constantes de tipo `Action`, `Rule`, `BalanceStrategy`, `MatchOperator`, `TargetType`.
- Structs de request/response — campos existentes no se remueven.
- Comportamiento de `ActionReject` como decisión válida (no error Go).

### Semi-estable (puede crecer, no reducir)

- Structs: pueden agregar campos nuevos (backward-compatible en Go).
- Constantes: pueden agregarse nuevas (no removerse).
- Nuevos errores sentinel pueden agregarse.
- Nuevas funciones/métodos pueden agregarse.

### No estable (puede cambiar)

- Campos no usados: `PresorterConfig.ErrorControlFlag`, `Assignment.Metadata`.
- Constantes no producidas: `ActionRecirculate`, `ActionError`, `ActionNoop`.
- Errores no producidos: `ErrNoAssignmentMatch`, `ErrNoAvailableExit`, `ErrNoAvailablePark`.
- Formato exacto de mensajes de error (usar `errors.Is`, no string matching).
- Orden de `CandidateEvaluations` (estable hoy, pero no garantizado formalmente).

---

## Cómo Romper Cambios en Futuras Versiones

Siguiendo Semantic Versioning:

1. **Patch (v0.1.x)**: solo bug fixes. Sin cambios de API.
2. **Minor (v0.x.0)**: nuevas features backward-compatible. Nuevos campos, constantes, funciones.
3. **Major (v1.0.0)**: cambios breaking. Renombrar tipos, cambiar firmas, remover campos.

Antes de v1.0.0, minor versions (v0.x.0) pueden incluir breaking changes
documentados en CHANGELOG.md, siguiendo la convención de Go modules pre-v1.

---

## Cuándo Pasar a v0.1.0 Final

Requisitos para promover rc1 → v0.1.0:

1. Todos los checks del release checklist en verde.
2. Al menos una integración exitosa con un orquestador (aunque sea prototipo).
3. No se descubren bugs en la API durante período de prueba.
4. CHANGELOG actualizado con la fecha final.
5. Tag `v0.1.0` en commit limpio.

---

## Cuándo Pasar a v0.2.0

Triggers para v0.2.0:

- Nuevas balance strategies (ej: weighted-random).
- Nuevos operadores de matching (ej: regex, contains, range).
- Adapters de integración (interfaces `ConfigLoader`, `StateProvider`).
- Lógica de recirculación.
- Remover campos/constantes marcados como "no estable".
- Métricas hooks.

---

## Política de Deprecation

1. Marcar con `// Deprecated:` en el godoc del elemento.
2. Mantener funcionando durante al menos una minor version.
3. Remover en la siguiente minor version con entrada en CHANGELOG.
4. Nunca remover en patch.
