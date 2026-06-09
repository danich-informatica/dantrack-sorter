// Command fallbacks demonstrates fallback behavior when exits/parks are unavailable.
package main

import (
	"context"
	"fmt"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func main() {
	sorterFallback()
	fmt.Println()
	presorterFallback()
}

func sorterFallback() {
	fmt.Println("=== Sorter: Matched exit blocked → fallback to default ===")

	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Sorter: &sorter.SorterConfig{
			SorterID: "sorter_fallback",
			Exits: []sorter.SorterExit{
				{ExitID: "exit_A", Enabled: true},
				{ExitID: "exit_default", Enabled: true},
			},
			DefaultExitID: "exit_default",
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Assignment matches exit_A, but exit_A is blocked.
	decision, err := engine.ResolveSorter(context.Background(), sorter.SorterRequest{
		TraceID: "trace-fallback-sorter",
		Box:     sorter.NewBox("BOX-F01").QR("QR-F01").Variety("Fuji").Build(),
		Assignments: sorter.AssignmentSet{
			Assignments: []sorter.Assignment{
				sorter.NewAssignment("assign_fuji", 10).
					MatchEquals("variety", "Fuji").
					ToExit("exit_A").
					Build(),
			},
		},
		ExitStates: sorter.ExitStates(
			sorter.ExitBlocked("exit_A"),
			sorter.ExitAvailable("exit_default", 0),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Action:       %s\n", decision.Action)
	fmt.Printf("ExitID:       %s\n", decision.ExitID)
	fmt.Printf("FallbackUsed: %v\n", decision.FallbackUsed)
	fmt.Printf("RuleApplied:  %s\n", decision.Trace.RuleApplied)
}

func presorterFallback() {
	fmt.Println("=== Presorter: Other parks full → only DefaultParkID remains ===")

	engine, err := sorter.NewEngine(sorter.EngineConfig{
		Presorter: &sorter.PresorterConfig{
			Parks: []sorter.ParkConfig{
				{ParkID: "park_A", Enabled: true},
				{ParkID: "park_B", Enabled: true},
				{ParkID: "park_fallback", Enabled: true},
			},
			BalanceStrategy: sorter.BalanceLeastLoaded,
			DefaultParkID:   "park_fallback",
		},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// park_A and park_B are full; park_fallback (default) is the only one available.
	decision, err := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID: "trace-fallback-presorter-1",
		Box:     sorter.NewBox("BOX-F02").QR("QR-F02").Build(),
		ParkStates: sorter.ParkStates(
			sorter.ParkFull("park_A"),
			sorter.ParkFull("park_B"),
			sorter.ParkAvailable("park_fallback", 0),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Action:       %s\n", decision.Action)
	fmt.Printf("ParkID:       %s\n", decision.ParkID)
	fmt.Printf("FallbackUsed: %v\n", decision.FallbackUsed)
	fmt.Printf("RuleApplied:  %s\n", decision.Trace.RuleApplied)
	fmt.Println()

	// All parks including default are blocked → ActionReject.
	fmt.Println("=== Presorter: ALL parks blocked → ActionReject ===")
	decision2, err := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID: "trace-fallback-presorter-2",
		Box:     sorter.NewBox("BOX-F03").QR("QR-F03").Build(),
		ParkStates: sorter.ParkStates(
			sorter.ParkBlocked("park_A"),
			sorter.ParkBlocked("park_B"),
			sorter.ParkBlocked("park_fallback"),
		),
		EvalTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Action:       %s\n", decision2.Action)
	fmt.Printf("Rejected:     %v\n", decision2.Action == sorter.ActionReject)
	fmt.Printf("RuleApplied:  %s\n", decision2.Trace.RuleApplied)
	fmt.Printf("Reason:       %s\n", decision2.Trace.Reason)
}
