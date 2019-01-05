package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
)

func (s *Server) moveLoop(ctx context.Context) {
	s.count++
	s.lastCount = time.Now()
	moves, err := s.bridge.getMoves(ctx)

	if err != nil {
		s.Log(fmt.Sprintf("Error getting moves: %v", err))
		return
	}

	for _, move := range moves {
		s.Log(fmt.Sprintf("MOVE: %v", move.InstanceId))

		//Raise an alarm if the move has no record
		if move.Record == nil {
			s.RaiseIssue(ctx, "Record is missing from move", fmt.Sprintf("Move regarding %v is missing the record information", move.InstanceId), false)
			return
		}

		//Raise an alarm if the move context is incomplete
		if (move.GetBeforeContext() == nil || move.GetAfterContext() == nil) || (move.GetBeforeContext().Before == nil && move.GetBeforeContext().After == nil) || (move.GetAfterContext().Before == nil && move.GetAfterContext().After == nil) {
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

		err = s.bridge.clearMove(ctx, move)
		if err != nil {
			s.Log(fmt.Sprintf("Error clearing move: %v", err))
		}
	}
}
