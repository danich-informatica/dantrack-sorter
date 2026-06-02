// Command basic_presorter demonstrates creating an Engine with a PresorterConfig,
// defining parks with balance strategy, and resolving a presorter decision.
package main

import (
	"context"
	"fmt"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func main() {
	// 1. Define park configs.
	parks := []sorter.ParkConfig{
		{ParkID: "park_1", Enabled: true, HasErrorControl: false, Priority: 1, MaxLoad: 100, Weight: 5},
		{ParkID: "park_2", Enabled: true, HasErrorControl: false, Priority: 2, MaxLoad: 100, Weight: 3},
		{ParkID: "park_3", Enabled: true, HasErrorControl: true, Priority: 3, MaxLoad: 50, Weight: 1},
	}

	// 2. Create Engine with PresorterConfig using least_loaded strategy.
	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Presorter: &sorter.PresorterConfig{
			Parks:           parks,
			BalanceStrategy: sorter.BalanceLeastLoaded,
			DefaultParkID:   "park_1",
		},
	})
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		return
	}

	// 3. Define park states (park_2 has lowest load).
	parkStates := []sorter.ParkState{
		{ParkID: "park_1", Available: true, CurrentLoad: 45},
		{ParkID: "park_2", Available: true, CurrentLoad: 12},
		{ParkID: "park_3", Available: true, CurrentLoad: 30},
	}

	// 4. Build box context (normal box, no errors).
	box := sorter.BoxContext{
		BoxID:         "BOX-100",
		QR:            "QR-NORMAL",
		SKU:           "SKU-APPLE",
		Variety:       "Gala",
		Caliber:       "75",
		ReadStatus:    "ok",
		QualityStatus: "approved",
	}

	// 5. Resolve presorter decision.
	decision, err := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID:       "trace-presorter-001",
		CorrelationID: "batch-99",
		Box:           box,
		ParkStates:    parkStates,
		EvalTime:      time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 6. Print results.
	fmt.Println("=== Presorter Decision ===")
	fmt.Printf("Action:          %s\n", decision.Action)
	fmt.Printf("ParkID:          %s\n", decision.ParkID)
	fmt.Printf("BalanceStrategy: %s\n", decision.BalanceStrategy)
	fmt.Printf("RuleApplied:     %s\n", decision.Trace.RuleApplied)
	fmt.Printf("Reason:          %s\n", decision.Trace.Reason)
	fmt.Printf("FallbackUsed:    %v\n", decision.FallbackUsed)
}
