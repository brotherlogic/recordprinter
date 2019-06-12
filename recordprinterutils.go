package main

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbrm "github.com/brotherlogic/recordmover/proto"
)

func (s *Server) moveLoop(ctx context.Context) error {
	s.count++
	s.lastCount = time.Now()
	moves, err := s.bridge.getMoves(ctx)

	if err != nil {
		s.lastIssue = fmt.Sprintf("%v", err)
		s.Log(fmt.Sprintf("Error getting moves: %v", err))
		return err
	}

	//Sort moves by date
	sort.SliceStable(moves, func(i, j int) bool {
		return moves[i].MoveDate < moves[j].MoveDate
	})

	for _, move := range moves {
		err := s.move(ctx, move)
		if err != nil {
			return err
		}
	}

	s.lastIssue = "No issues"
	return nil
}

func (s *Server) move(ctx context.Context, move *pbrm.RecordMove) error {
	s.currMove = move.InstanceId
	s.Log(fmt.Sprintf("Trying to move %v", s.currMove))
	if move.GetBeforeContext() != nil && move.GetAfterContext() != nil && move.GetBeforeContext().Location != move.GetAfterContext().Location && move.GetAfterContext().After != nil {
		s.Log(fmt.Sprintf("MOVE: %v", move.InstanceId))

		//Raise an alarm if the move has no record
		if move.Record == nil {
			s.Log(fmt.Sprintf("Record missing"))
			s.lastIssue = "Record is missing from the move"
			s.RaiseIssue(ctx, "Record is missing from move", fmt.Sprintf("Move regarding %v is missing the record information", move.InstanceId), false)
			return nil
		}

		s.Log(fmt.Sprintf("%v and %v", move.GetBeforeContext().Location, move.GetAfterContext().Location))

		//We don't need to print purgatory or google_play moves
		if (move.GetBeforeContext().Location != "Purgatory" && move.GetAfterContext().Location != "Purgatory") &&
			(move.GetBeforeContext().Location != "Google Play" && move.GetAfterContext().Location != "Google Play") {

			//Raise an alarm if the move context is incomplete
			if (move.GetBeforeContext() == nil || move.GetAfterContext() == nil) ||
				(move.GetBeforeContext().Before == nil && move.GetBeforeContext().After == nil) ||
				(move.GetAfterContext().Before == nil || move.GetAfterContext().After == nil) {
				s.Log(fmt.Sprintf("No context"))
				s.lastIssue = "No Context"
				s.RaiseIssue(ctx, "Context is missing from move", fmt.Sprintf("Move regarding %v is missing the full context %v -> %v", move.InstanceId, move.BeforeContext, move.AfterContext), false)
				return nil
			}

			if move.GetAfterContext().After.GetMetadata() == nil {
				return fmt.Errorf("Record has no metadata")
			}

			// Only print if it's a FRESHMAN record
			if move.GetAfterContext().After.GetMetadata().Category == pbrc.ReleaseMetadata_FRESHMAN {
				lines := []string{fmt.Sprintf("%v: %v -> %v\n", move.Record.GetRelease().Title, move.GetBeforeContext().Location, move.GetAfterContext().Location)}
				lines = append(lines, fmt.Sprintf(" (Slot %v)\n", move.GetAfterContext().Slot))
				if move.GetAfterContext().GetBefore() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetBefore().GetRelease().Title))
				}
				lines = append(lines, fmt.Sprintf(" %v\n", move.Record.GetRelease().Title))
				if move.GetAfterContext().GetAfter() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetAfter().GetRelease().Title))
				}

				s.Log(fmt.Sprintf("DELIVER: %v", move.InstanceId))

				err := s.bridge.print(ctx, lines)
				if err != nil {
					s.Log(fmt.Sprintf("Error printing move: %v", err))
					return nil
				}
			}

			err := s.bridge.clearMove(ctx, move)
			if err != nil {
				s.lastIssue = fmt.Sprintf("%v", err)
				s.Log(fmt.Sprintf("Error clearing move: %v", err))
			}

		}
	}

	tv := time.Now().Sub(time.Unix(move.MoveDate, 0))
	if tv > time.Hour*2 && (move.GetBeforeContext().Location == move.GetAfterContext().Location || move.GetBeforeContext().Location == "Purgatory") {
		s.Log(fmt.Sprintf("CLearning move (matching location %v [%v -> %v])", move.InstanceId, move.GetBeforeContext().Location))
		err := s.bridge.clearMove(ctx, move)
		if err != nil {
			s.lastIssue = fmt.Sprintf("%v", err)
			s.Log(fmt.Sprintf("Error clearing move: %v", err))
		}
	}

	return nil

}
