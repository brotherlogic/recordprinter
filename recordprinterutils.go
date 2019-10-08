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

	if move.GetBeforeContext() != nil && move.GetAfterContext() != nil && move.GetBeforeContext().Location != move.GetAfterContext().Location && move.GetAfterContext().After != nil {

		//Raise an alarm if the move has no record
		if move.Record == nil {
			s.lastIssue = "Record is missing from the move"
			return fmt.Errorf("Move regarding %v is missing the record information", move.InstanceId)
		}

		//We don't need to print purgatory or google_play moves
		if (move.GetBeforeContext().Location != "Purgatory" && move.GetAfterContext().Location != "Purgatory") &&
			(move.GetBeforeContext().Location != "Google Play" && move.GetAfterContext().Location != "Google Play") {

			//Raise an alarm if the move context is incomplete
			if (move.GetBeforeContext() == nil || move.GetAfterContext() == nil) ||
				(move.GetBeforeContext().Before == nil && move.GetBeforeContext().After == nil) ||

				(move.GetAfterContext().Before == nil || move.GetAfterContext().After == nil) {
				if move.GetBeforeContext().Location != "Bandcamp" {
					s.lastIssue = "No Context"
					return fmt.Errorf("Move regarding %v is missing context", move.InstanceId)
				}
			}

			if move.Record.GetMetadata() == nil {
				return fmt.Errorf("Record has no metadata")
			}

			marked := false
			// Only print if it's a FRESHMAN record or it's listed to sell
			if move.Record.GetMetadata().Category == pbrc.ReleaseMetadata_FRESHMAN ||
				move.Record.GetMetadata().Category == pbrc.ReleaseMetadata_LISTED_TO_SELL {
				lines := []string{fmt.Sprintf("%v: %v -> %v\n", move.Record.GetRelease().Title, move.GetBeforeContext().Location, move.GetAfterContext().Location)}
				lines = append(lines, fmt.Sprintf(" (Slot %v)\n", move.GetAfterContext().Slot))
				if move.GetAfterContext().GetBefore() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetBefore().GetRelease().Title))
				}
				lines = append(lines, fmt.Sprintf(" %v\n", move.Record.GetRelease().Title))
				if move.GetAfterContext().GetAfter() != nil {
					lines = append(lines, fmt.Sprintf(" %v\n", move.GetAfterContext().GetAfter().GetRelease().Title))
				}

				err := s.bridge.print(ctx, lines)
				if err != nil {
					return err
				}
				marked = true
			}

			// Only clear SOLD records
			if marked || move.Record.GetMetadata().Category == pbrc.ReleaseMetadata_SOLD ||
				move.Record.GetMetadata().Category == pbrc.ReleaseMetadata_SOLD_ARCHIVE {
				err := s.bridge.clearMove(ctx, move)
				if err != nil {
					return err
				}
			}

		}
	} else {
		return fmt.Errorf("Cannot process move: %v", move.InstanceId)
	}

	tv := time.Now().Sub(time.Unix(move.MoveDate, 0))
	if move.GetBeforeContext() != nil && move.GetAfterContext() != nil && tv > time.Hour*2 && (move.GetBeforeContext().Location == move.GetAfterContext().Location || move.GetBeforeContext().Location == "Purgatory") {
		err := s.bridge.clearMove(ctx, move)
		if err != nil {
			return err
		}
	}

	return nil

}
