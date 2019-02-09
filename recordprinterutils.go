package main

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/context"
)

func (s *Server) moveLoop(ctx context.Context) {
	s.count++
	s.lastCount = time.Now()
	moves, err := s.bridge.getMoves(ctx)

	if err != nil {
		s.lastIssue = fmt.Sprintf("%v", err)
		s.Log(fmt.Sprintf("Error getting moves: %v", err))
		return
	}

	//Sort moves by date
	sort.SliceStable(moves, func(i, j int) bool {
		return moves[i].MoveDate < moves[j].MoveDate
	})

	for _, move := range moves {
		if move.GetBeforeContext() != nil && move.GetAfterContext() != nil && move.GetBeforeContext().Location != move.GetAfterContext().Location {
			s.Log(fmt.Sprintf("MOVE: %v", move.InstanceId))

			//Raise an alarm if the move has no record
			if move.Record == nil {
				s.lastIssue = "Record is missing from the move"
				s.RaiseIssue(ctx, "Record is missing from move", fmt.Sprintf("Move regarding %v is missing the record information", move.InstanceId), false)
				return
			}

			//We don't need to print purgatory or google_play moves
			if (move.GetBeforeContext().Location != "Purgatory" && move.GetAfterContext().Location != "Purgatory") &&
				(move.GetBeforeContext().Location != "Google Play" && move.GetAfterContext().Location != "Google Play") {

				//Raise an alarm if the move context is incomplete
				if (move.GetBeforeContext() == nil || move.GetAfterContext() == nil) ||
					(move.GetBeforeContext().Before == nil && move.GetBeforeContext().After == nil) ||
					(move.GetAfterContext().Before == nil || move.GetAfterContext().After == nil) {
					s.lastIssue = "No Context"
					s.RaiseIssue(ctx, "Context is missing from move", fmt.Sprintf("Move regarding %v is missing the full context %v -> %v", move.InstanceId, move.BeforeContext, move.AfterContext), false)
					return
				}

				lines := []string{fmt.Sprintf("%v: %v -> %v\n", move.Record.GetRelease().Title, move.GetBeforeContext().Location, move.GetAfterContext().Location)}
				lines = append(lines, fmt.Sprintf(" (Slot %v)\n", move.GetAfterContext().Slot))
				if move.GetAfterContext().GetBefore() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetBefore().GetRelease().Title))
				}
				lines = append(lines, fmt.Sprintf(" %v\n", move.Record.GetRelease().Title))
				if move.GetAfterContext().GetAfter() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetAfter().GetRelease().Title))
				}

				s.Log(fmt.Sprintf("PRINTING: %v", lines))

				err = s.bridge.print(ctx, lines)
				if err != nil {
					s.Log(fmt.Sprintf("Error printing move: %v", err))
					return
				}
			}

			err = s.bridge.clearMove(ctx, move)
			if err != nil {
				s.lastIssue = fmt.Sprintf("%v", err)
				s.Log(fmt.Sprintf("Error clearing move: %v", err))
			}
		}
	}

	s.lastIssue = "No issues"
}
