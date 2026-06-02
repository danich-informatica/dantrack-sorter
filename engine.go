package sorter

// Engine es el motor de decisión de dantrack-sorter.
// Es seguro para uso concurrente. El único estado mutable es rrCounter (round-robin),
// que se sincroniza con sync/atomic.
//
// El round-robin counter NO se persiste entre reinicios del proceso.
// Es determinista respecto al estado del engine dentro de un mismo ciclo de vida.
type Engine struct {
	sorterCfg    *SorterConfig         // copia aislada; nil si el sorter no está configurado
	presorterCfg *PresorterConfig      // copia aislada; nil si el presorter no está configurado
	exitIndex    map[string]SorterExit // índice por ExitID; construido en NewEngine para O(1) lookup
	rrCounter    uint64                // round-robin counter; sincronizado con sync/atomic
}

// NewEngine crea un Engine con la configuración dada.
// Llama ValidateConfig internamente; devuelve error compatible con ErrInvalidConfig si falla.
// No abre conexiones, no inicializa hardware, no genera IDs.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	e := &Engine{}

	if cfg.Sorter != nil {
		// Copiar el valor del struct para aislar el Engine de cambios externos al cfg.Sorter.
		// El slice de Exits también se indexa, por lo que mutaciones externas del slice
		// no afectan el índice del Engine.
		sc := *cfg.Sorter
		e.sorterCfg = &sc
		e.exitIndex = make(map[string]SorterExit, len(sc.Exits))
		for _, ex := range sc.Exits {
			e.exitIndex[ex.ExitID] = ex
		}
	}

	if cfg.Presorter != nil {
		pc := *cfg.Presorter
		e.presorterCfg = &pc
	}

	return e, nil
}
