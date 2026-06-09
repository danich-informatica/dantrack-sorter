// Command basic_sorter demonstrates creating an Engine with a SorterConfig,
// evaluating assignments, and resolving a sorter decision for a box.
package main

import (
	"context"
	"fmt"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func main() {
	// 1. Define sorter exits.
	exits := []sorter.SorterExit{
		{ExitID: "exit_A", Enabled: true, Priority: 1, Capacity: 50},
		{ExitID: "exit_B", Enabled: true, Priority: 2, Capacity: 50},
		{ExitID: "exit_default", Enabled: true, Priority: 0, Capacity: 100},
	}

	// 2. Create Engine with SorterConfig.
	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Sorter: &sorter.SorterConfig{
			SorterID:      "sorter_01",
			Exits:         exits,
			DefaultExitID: "exit_default",
			RejectExitID:  "",
		},
	})
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		return
	}

	// 3. Define assignments using builders.
	assignments := sorter.AssignmentSet{
		Assignments: []sorter.Assignment{
			sorter.NewAssignment("assign_apples_A", 10).
				MatchEquals("variety", "Fuji").
				MatchIn("caliber", "80", "85", "90").
				ToExit("exit_A").
				Build(),
			sorter.NewAssignment("assign_oranges_B", 5).
				MatchEquals("variety", "Navel").
				ToExit("exit_B").
				Build(),
		},
	}

	// 4. Build box context.
	box := sorter.NewBox("BOX-001").
		QR("QR123456").
		SKU("SKU-FUJI-80").
		Variety("Fuji").
		Caliber("80").
		Build()

	// 5. Build exit states (all available).
	exitStates := sorter.ExitStates(
		sorter.ExitAvailable("exit_A", 0),
		sorter.ExitAvailable("exit_B", 0),
		sorter.ExitAvailable("exit_default", 0),
	)

	// 6. Resolve sorter decision.
	decision, err := engine.ResolveSorter(context.Background(), sorter.SorterRequest{
		TraceID:       "trace-example-001",
		CorrelationID: "session-42",
		Box:           box,
		Assignments:   assignments,
		ExitStates:    exitStates,
		EvalTime:      time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 7. Print results.
	fmt.Println("=== Sorter Decision ===")
	fmt.Printf("Action:       %s\n", decision.Action)
	fmt.Printf("ExitID:       %s\n", decision.ExitID)
	fmt.Printf("AssignmentID: %s\n", decision.AssignmentID)
	fmt.Printf("RuleApplied:  %s\n", decision.Trace.RuleApplied)
	fmt.Printf("Reason:       %s\n", decision.Trace.Reason)
	fmt.Printf("FallbackUsed: %v\n", decision.FallbackUsed)
	fmt.Printf("Rejected:     %v\n", decision.Rejected)
}
