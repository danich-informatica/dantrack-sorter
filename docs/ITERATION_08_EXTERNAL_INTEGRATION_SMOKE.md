# Iteración 08 — External Integration Smoke Test

**Autor:** Arquitectura DANTRACK  
**Fecha:** 2026-06-02  
**Estado:** Completada  
**Módulo:** `github.com/dantrack/dantrack-sorter`  
**Package:** `package sorter`

---

## 1. Resumen Ejecutivo

Se creó un smoke test externo que simula un orquestador consumiendo `dantrack-sorter`
como módulo Go independiente. El smoke test valida programáticamente 7 escenarios
de integración cubriendo sorter, presorter, fallback, error control, trazabilidad
y CandidateEvaluations. Todos pasan sin errores.

La librería está lista para ser taggeada como v0.1.0-rc1.

---

## 2. Estado

**Completada** — smoke test externo exitoso.

---

## 3. Objetivo

Validar que `dantrack-sorter` puede ser consumida correctamente desde un proyecto
externo, simulando un orquestador futuro, sin dependencias reales.

---

## 4. Alcance Implementado

| Área | Implementado |
|---|---|
| Módulo externo con `replace` directive | Sí |
| Simulación de config desde DB | Sí |
| Simulación de eventos desde scanner/cámara | Sí |
| Simulación de estados desde PLC | Sí |
| Llamadas a EvaluateAssignments, ResolveSorter, ResolvePresorter | Sí |
| Validaciones programáticas (log.Fatalf on failure) | Sí |
| 7 escenarios de integración | Sí |
| Salida impresa detallada | Sí |

---

## 5. Archivos Creados/Modificados

| Archivo | Acción | Descripción |
|---|---|---|
| `integration_smoke/go.mod` | **Creado** | Módulo externo con replace hacia `..` |
| `integration_smoke/main.go` | **Creado** | Smoke test con 7 validaciones |
| `integration_smoke/README.md` | **Creado** | Instrucciones de uso |
| `docs/ITERATION_08_EXTERNAL_INTEGRATION_SMOKE.md` | **Creado** | Este documento |
| `docs/RELEASE_CHECKLIST.md` | Modificado | Evidencia de smoke test |

---

## 6. Qué Simula el Smoke Test

| Componente simulado | En producción sería |
|---|---|
| Config hardcodeada | `dantrack-db` → SorterConfig, PresorterConfig |
| Assignments hardcodeados | `dantrack-db` → AssignmentSet |
| BoxContext construido inline | `dantrack-connect` → evento de scanner/cámara |
| ExitStates hardcodeados | `dantrack-connect` → lectura de PLC |
| ParkStates hardcodeados | `dantrack-connect` → lectura de PLC |
| TraceID generado inline | Orquestador → UUID generator |
| Validación con log.Fatalf | Orquestador → error handling, métricas, alarmas |

---

## 7. Qué NO Simula

- Conexión real a DB.
- Conexión real a hardware/PLC.
- Generación de UUIDs.
- Persistencia de auditoría.
- Retry/timeout de hardware.
- Concurrencia real de múltiples cajas.
- Streaming de eventos.
- Servidor HTTP/gRPC.

---

## 8. Flujo del Orquestador Fake

```text
1. Crear EngineConfig (SorterConfig + PresorterConfig)
2. NewEngine(cfg) → *Engine
3. Para cada escenario:
   a. Construir BoxContext
   b. Construir ExitStates o ParkStates
   c. Construir Assignments (si sorter)
   d. Llamar ResolveSorter o ResolvePresorter
   e. Validar resultado programáticamente
   f. Imprimir salida
4. Si todo pasa: "ALL SMOKE TESTS PASSED"
5. Si algo falla: log.Fatalf termina con exit code 1
```

---

## 9. Validaciones Realizadas

| # | Test | Validación |
|---|---|---|
| 1 | Sorter match | Fuji/80 → exit_fuji, assignment_matched, no fallback |
| 2 | Sorter no match | Honeycrisp → ActionReject, exit_reject, sorter_reject_no_match |
| 3 | Sorter blocked fallback | Fuji match pero exit_fuji blocked → exit_default, FallbackUsed=true |
| 4 | Presorter error control | no_read → park_error, ErrorControlRequired=true, Available=true |
| 5 | Presorter least_loaded | Normal box → park con menor carga (park_error, load=3) |
| 6 | Trace preserved | TraceID y CorrelationID copiados en sorter y presorter |
| 7 | CandidateEvaluations | No vacío en ambos motores (4 entries sorter, 3 presorter) |

---

## 10. Comandos Ejecutados

```bash
# Repo base
go fmt ./...
go vet ./...
go test ./... -count=1
go test -race ./... -count=1
go test -cover ./... -count=1

# Smoke test
cd integration_smoke
go run .
cd ..

# Verificar repo no afectado
go test ./... -count=1
```

---

## 11. Resultado Real

### Repo base

```
ok   github.com/dantrack/dantrack-sorter   0.908s   coverage: 95.3% of statements
0 race conditions
156 tests PASS
```

### Smoke test

```
=== DANTRACK-SORTER INTEGRATION SMOKE TEST ===
[OK] Engine created with Sorter + Presorter config
[OK] 3 assignments loaded
[OK] TEST 1 PASSED
[OK] TEST 2 PASSED
[OK] TEST 3 PASSED
[OK] TEST 4 PASSED
[OK] TEST 5 PASSED
[OK] TEST 6 PASSED
[OK] TEST 7 PASSED
=== ALL SMOKE TESTS PASSED ===
```

### Interferencia con repo principal

Ninguna. El módulo anidado con su propio `go.mod` no afecta `go test ./...`
del repo principal.

---

## 12. Problemas Detectados

**Ninguno.**

- La librería se integra sin problemas desde módulo externo.
- El `replace` directive funciona correctamente.
- No hay panics ni errores inesperados.
- Trazabilidad funciona end-to-end.
- Balance strategies funcionan correctamente.
- Error control funciona correctamente.
- Fallback funciona correctamente.

---

## 13. Decisiones Técnicas

| Decisión | Justificación |
|---|---|
| Smoke test como `go run .` (no `go test`) | Simula un binario real de orquestador; más realista que test unitario |
| `log.Fatalf` en vez de `t.Fatal` | No es un test Go convencional; es validación de integración |
| `replace` directive hacia `..` | Consume la librería local sin publicar en proxy |
| 7 escenarios independientes | Cubren los paths principales sin ser exhaustivos |
| TEST 5 elige park_error (load=3) | Correcto: least_loaded elige el de menor carga de TODOS los parks disponibles para normal box |
| No se usa `go test ./...` en smoke | Es un programa main; no tiene test files |

---

## 14. Recomendación

**Promover a v0.1.0-rc1 — listo para taggear.**

Evidencia:
- 156 tests internos PASS.
- 95.3% coverage.
- 0 data races.
- 7/7 smoke tests de integración PASS.
- API funciona correctamente desde consumidor externo.
- Sin problemas detectados.

Para promover a v0.1.0 final: deploy en orquestador real de desarrollo.

---

## 15. Next Agent Instructions

```
Iteración 09 — Por definir

Contexto:
- Iteraciones 02–08 completadas.
- v0.1.0-rc1 validado con smoke test externo.
- 156 tests + 7 smoke tests.
- 95.3% coverage, race-free.
- API estable, documentada, integrable.

Estado:
- La librería puede ser taggeada como v0.1.0-rc1.
- Para v0.1.0 final: requiere uso en orquestador real de desarrollo.

Opciones para Iteración 09:
1. Tag v0.1.0-rc1 en git.
2. Prototipo de orquestador (proyecto separado).
3. Adapters: interfaces ConfigLoader, StateProvider, AuditWriter.
4. Métricas hooks (callback pattern sin dependencias externas).
5. Nuevos operadores de matching.

No implementar en dantrack-sorter:
- DB, hardware, connect, sim.
- Servidor HTTP/gRPC.
- CLI productiva.
```
