// Command error_control demonstrates how the presorter prefers parks with
// error control when a box has read errors or quality issues.
package main

import (
	"context"
	"fmt"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func main() {
	// 1. Configure parks: park_normal has no error control, park_error has it.
	parks := []sorter.ParkConfig{
		{ParkID: "park_normal", Enabled: true, HasErrorControl: false, Priority: 1, MaxLoad: 100, Weight: 5},
		{ParkID: "park_error", Enabled: true, HasErrorControl: true, Priority: 2, MaxLoad: 50, Weight: 1},
	}

	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Presorter: &sorter.PresorterConfig{
			Parks:           parks,
			BalanceStrategy: sorter.BalanceLeastLoaded,
			DefaultParkID:   "park_normal",
		},
	})
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		return
	}

	parkStates := []sorter.ParkState{
		{ParkID: "park_normal", Available: true, CurrentLoad: 10},
		{ParkID: "park_error", Available: true, CurrentLoad: 5},
	}

	// 2. Box with read error — should prefer error control park.
	boxNoRead := sorter.BoxContext{
		BoxID:      "BOX-ERR-001",
		QR:         "QR-NOREAD",
		ReadStatus: "no_read",
	}

	decision, err := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID:    "trace-err-001",
		Box:        boxNoRead,
		ParkStates: parkStates,
		EvalTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("=== Error Control: no_read box ===")
	fmt.Printf("ParkID:               %s\n", decision.ParkID)
	fmt.Printf("ErrorControlRequired:  %v\n", decision.ErrorControlRequired)
	fmt.Printf("ErrorControlAvailable: %v\n", decision.ErrorControlAvailable)
	fmt.Printf("FallbackUsed:          %v\n", decision.FallbackUsed)
	fmt.Println()

	// 3. Box with ErrorFlags — should also prefer error control park.
	boxFlags := sorter.BoxContext{
		BoxID:      "BOX-ERR-002",
		QR:         "QR-FLAGS",
		ReadStatus: "ok",
		ErrorFlags: []string{"damaged_label", "weight_mismatch"},
	}

	decision2, err := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID:    "trace-err-002",
		Box:        boxFlags,
		ParkStates: parkStates,
		EvalTime:   time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("=== Error Control: ErrorFlags box ===")
	fmt.Printf("ParkID:               %s\n", decision2.ParkID)
	fmt.Printf("ErrorControlRequired:  %v\n", decision2.ErrorControlRequired)
	fmt.Printf("ErrorControlAvailable: %v\n", decision2.ErrorControlAvailable)
	fmt.Printf("FallbackUsed:          %v\n", decision2.FallbackUsed)
}
