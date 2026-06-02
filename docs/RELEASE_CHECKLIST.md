# Release Checklist — dantrack-sorter v0.1.0

Checklist para liberar la versión v0.1.0 de `dantrack-sorter`.

---

## Pre-Release Checks

### Code Quality

- [ ] `go fmt ./...` — sin archivos reformateados.
- [ ] `go vet ./...` — sin warnings.
- [ ] `go test ./... -count=1` — 0 FAIL.
- [ ] `go test -race ./...` — 0 data races.
- [ ] `go test -cover ./...` — cobertura ≥ 90%.
- [ ] Godoc examples pasan (`go test -run Example -v`).

### Examples

- [ ] `go run ./examples/basic_sorter` — salida esperada.
- [ ] `go run ./examples/basic_presorter` — salida esperada.
- [ ] `go run ./examples/error_control` — salida esperada.
- [ ] `go run ./examples/fallbacks` — salida esperada.

### Documentation

- [ ] `README.md` actualizado con estado y versión.
- [ ] `docs/ARCHITECTURE.md` — arquitectura vigente.
- [ ] `docs/INTEGRATION_CONTRACTS.md` — contratos vigentes.
- [ ] `docs/PUBLIC_API_REVIEW.md` — API revisada.
- [ ] `docs/VERSIONING.md` — política de versiones documentada.
- [ ] `CHANGELOG.md` — entrada para la versión.
- [ ] Godoc comments en todas las funciones públicas.

### Integration Smoke Test

- [x] `cd integration_smoke && go run .` — 7/7 tests PASS.
- [x] Sorter: match, no-match→reject, blocked→fallback.
- [x] Presorter: error control, least_loaded.
- [x] Trace: TraceID y CorrelationID preservados.
- [x] CandidateEvaluations: no vacío en ambos motores.
- [x] Repo principal no afectado por módulo anidado.

### API Stability

- [ ] No hay TODO/FIXME en código publicado que afecte funcionalidad.
- [ ] Tipos exportados son forward-compatible (structs pueden crecer).
- [ ] Errores sentinel son estables y compatibles con `errors.Is`.
- [ ] No hay dependencias externas.

### Version & Tag

- [ ] `go.mod` con module path correcto.
- [ ] Tag `v0.1.0` creado en commit limpio (tests verdes, fmt OK).
- [ ] CHANGELOG.md refleja la versión taggeada.

---

## Known Limitations

Documentar antes de release:

- [ ] Round-robin counter no persiste entre reinicios.
- [ ] `weighted` siempre elige el mismo park si pesos no cambian.
- [ ] No hay validación cruzada ParkState vs ParkConfig.
- [ ] `PresorterConfig.ErrorControlFlag` definido pero no usado en la implementación.
- [ ] `ActionRecirculate`, `ActionError`, `ActionNoop` definidos pero nunca producidos.
- [ ] No hay adapters de integración (responsabilidad del orquestador).
- [ ] No hay métricas ni observabilidad integrada.

---

## Post-Release

- [ ] Verificar que `go get github.com/dantrack/dantrack-sorter@v0.1.0` funciona.
- [ ] Verificar que godoc renderiza correctamente.
- [ ] Comunicar release a equipos dependientes.
