package sorter_test

import (
	"context"
	"fmt"
	"time"

	sorter "github.com/danich-informatica/dantrack-sorter"
)

func ExampleEvaluateAssignments() {
	box := sorter.BoxContext{
		BoxID:   "BOX-001",
		QR:      "QR001",
		Variety: "Fuji",
		Caliber: "80",
	}

	set := sorter.AssignmentSet{
		Assignments: []sorter.Assignment{
			{
				ID:       "assign_fuji",
				Enabled:  true,
				Priority: 10,
				Conditions: []sorter.MatchCondition{
					{Field: "variety", Operator: sorter.OpEquals, Values: []string{"Fuji"}},
				},
				TargetType: sorter.TargetTypeExit,
				TargetID:   "exit_A",
			},
			{
				ID:       "assign_gala",
				Enabled:  true,
				Priority: 5,
				Conditions: []sorter.MatchCondition{
					{Field: "variety", Operator: sorter.OpEquals, Values: []string{"Gala"}},
				},
				TargetType: sorter.TargetTypeExit,
				TargetID:   "exit_B",
			},
		},
	}

	results := sorter.EvaluateAssignments(box, set, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	for _, r := range results {
		fmt.Printf("%s matched=%v\n", r.Assignment.ID, r.Matched)
	}
	// Output:
	// assign_fuji matched=true
	// assign_gala matched=false
}

func ExampleEngine_ResolveSorter() {
	engine, _ := sorter.NewEngine(sorter.EngineConfig{
		Sorter: &sorter.SorterConfig{
			SorterID: "sorter_01",
			Exits: []sorter.SorterExit{
				{ExitID: "exit_A", Enabled: true},
				{ExitID: "exit_default", Enabled: true},
			},
			DefaultExitID: "exit_default",
		},
	})

	decision, _ := engine.ResolveSorter(context.Background(), sorter.SorterRequest{
		TraceID: "trace-001",
		Box:     sorter.BoxContext{BoxID: "BOX-001", QR: "QR001", Variety: "Fuji"},
		Assignments: sorter.AssignmentSet{
			Assignments: []sorter.Assignment{
				{
					ID:       "assign_fuji",
					Enabled:  true,
					Priority: 10,
					Conditions: []sorter.MatchCondition{
						{Field: "variety", Operator: sorter.OpEquals, Values: []string{"Fuji"}},
					},
					TargetType: sorter.TargetTypeExit,
					TargetID:   "exit_A",
				},
			},
		},
		ExitStates: []sorter.SorterExitState{
			{ExitID: "exit_A", Available: true},
			{ExitID: "exit_default", Available: true},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	fmt.Printf("Action=%s ExitID=%s\n", decision.Action, decision.ExitID)
	// Output:
	// Action=route ExitID=exit_A
}

func ExampleEngine_ResolvePresorter() {
	engine, _ := sorter.NewEngine(sorter.EngineConfig{
		Presorter: &sorter.PresorterConfig{
			Parks: []sorter.ParkConfig{
				{ParkID: "park_1", Enabled: true, Weight: 3},
				{ParkID: "park_2", Enabled: true, Weight: 5},
			},
			BalanceStrategy: sorter.BalanceLeastLoaded,
			DefaultParkID:   "park_1",
		},
	})

	decision, _ := engine.ResolvePresorter(context.Background(), sorter.PresorterRequest{
		TraceID: "trace-002",
		Box:     sorter.BoxContext{BoxID: "BOX-002", QR: "QR002"},
		ParkStates: []sorter.ParkState{
			{ParkID: "park_1", Available: true, CurrentLoad: 30},
			{ParkID: "park_2", Available: true, CurrentLoad: 10},
		},
		EvalTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	fmt.Printf("Action=%s ParkID=%s\n", decision.Action, decision.ParkID)
	// Output:
	// Action=route ParkID=park_2
}
