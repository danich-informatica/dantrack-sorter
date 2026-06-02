package sorter

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// TestNewEngine — construcción del Engine
// ---------------------------------------------------------------------------

func TestNewEngine_ValidSorterConfig(t *testing.T) {
	cfg := EngineConfig{Sorter: ptr(validSorterConfig())}
	e, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil Engine")
	}
}

func TestNewEngine_ValidPresorterConfig(t *testing.T) {
	cfg := EngineConfig{Presorter: ptr(validPresorterConfig())}
	e, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil Engine")
	}
}

func TestNewEngine_BothConfigured(t *testing.T) {
	cfg := EngineConfig{
		Presorter: ptr(validPresorterConfig()),
		Sorter:    ptr(validSorterConfig()),
	}
	e, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil Engine")
	}
}

func TestNewEngine_NilBothConfigs_ReturnsErrInvalidConfig(t *testing.T) {
	cfg := EngineConfig{}
	_, err := NewEngine(cfg)
	if err == nil {
		t.Fatal("expected error for empty EngineConfig, got nil")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestNewEngine_InvalidSorterConfig_ReturnsErrInvalidConfig(t *testing.T) {
	bad := validSorterConfig()
	bad.SorterID = "" // campo requerido vacío → ErrMissingRequiredField (subclase de ErrInvalidConfig)
	_, err := NewEngine(EngineConfig{Sorter: &bad})
	if err == nil {
		t.Fatal("expected error for invalid SorterConfig, got nil")
	}
	// La validación devuelve ErrMissingRequiredField para campos vacíos,
	// y ErrInvalidConfig para invariantes de config. Ambos son errores de configuración.
	if !errors.Is(err, ErrInvalidConfig) && !errors.Is(err, ErrMissingRequiredField) {
		t.Errorf("expected ErrInvalidConfig or ErrMissingRequiredField, got: %v", err)
	}
}

// TestNewEngine_IsolatesFromExternalMutation verifica que el Engine no se ve afectado
// si el caller modifica la SorterConfig original después de NewEngine.
func TestNewEngine_IsolatesFromExternalMutation(t *testing.T) {
	sc := validSorterConfig()
	e, err := NewEngine(EngineConfig{Sorter: &sc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutar la config original: deshabilitar todos los exits.
	for i := range sc.Exits {
		sc.Exits[i].Enabled = false
	}

	// El índice interno del Engine fue construido antes de la mutación.
	// Los exits deben seguir apareciendo en el exitIndex del engine.
	if len(e.exitIndex) == 0 {
		t.Error("expected engine exitIndex to be non-empty after external mutation of original config")
	}
	for id, ex := range e.exitIndex {
		if !ex.Enabled {
			t.Errorf("exit %q in engine exitIndex was affected by external mutation", id)
		}
	}
}

// TestNewEngine_ExitIndexBuilt verifica que el exitIndex del Engine se construye correctamente.
func TestNewEngine_ExitIndexBuilt(t *testing.T) {
	sc := SorterConfig{
		SorterID: "sorter-01",
		Exits: []SorterExit{
			{ExitID: "exit-A", Enabled: true},
			{ExitID: "exit-B", Enabled: false},
		},
	}
	e, err := NewEngine(EngineConfig{Sorter: &sc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := e.exitIndex["exit-A"]; !ok {
		t.Error("expected exit-A in exitIndex")
	}
	if _, ok := e.exitIndex["exit-B"]; !ok {
		t.Error("expected exit-B in exitIndex")
	}
	if len(e.exitIndex) != 2 {
		t.Errorf("expected 2 exits in exitIndex, got %d", len(e.exitIndex))
	}
}
