# Integration Smoke Test

Este directorio simula un consumidor externo de `dantrack-sorter`,
como un orquestador futuro que usa la librería para tomar decisiones
de clasificación.

## Qué hace

1. Carga configuración hardcodeada (simula datos de DB).
2. Simula eventos de caja (scanner/cámara).
3. Simula estados de hardware (PLC/sensors).
4. Llama a `EvaluateAssignments`, `ResolveSorter` y `ResolvePresorter`.
5. Valida programáticamente que los resultados son correctos.
6. Imprime salida detallada de cada decisión.

## Cómo ejecutar

```bash
cd integration_smoke
go run .
```

## Qué NO hace

- No conecta a DB real.
- No conecta a hardware real.
- No usa dantrack-connect, dantrack-db ni dantrack-sim.
- No es un servidor ni un CLI productivo.

## Resultado esperado

Si todo es correcto, la salida termina con:

```
=== ALL SMOKE TESTS PASSED ===
```

Si alguna validación falla, el programa termina con `log.Fatalf`.
