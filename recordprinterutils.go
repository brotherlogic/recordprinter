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

	if move.GetBeforeContext() != nil && move.GetAfterContext() != nil && move.GetBeforeContext().Location != move.GetAfterContext().Location {

		//We don't need to print purgatory or google_play moves
		if (move.GetBeforeContext().Location != "Purgatory" && move.GetAfterContext().Location != "Purgatory") &&
			(move.GetBeforeContext().Location != "Google Play" && move.GetAfterContext().Location != "Google Play") {

			marked := false

			record, err := s.bridge.getRecord(ctx, move.InstanceId)
			if err != nil {
				return err
			}

			// Only print if it's a FRESHMAN record or it's listed to sell
			if record.GetMetadata().Category == pbrc.ReleaseMetadata_FRESHMAN ||
				record.GetMetadata().Category == pbrc.ReleaseMetadata_LISTED_TO_SELL {
				lines := []string{fmt.Sprintf("%v: %v -> %v\n", record.GetRelease().Title, move.GetBeforeContext().Location, move.GetAfterContext().Location)}
				lines = append(lines, fmt.Sprintf(" (Slot %v)\n", move.GetAfterContext().Slot))
				if move.GetAfterContext().GetBefore() != nil {
					bef, err := s.bridge.getRecord(ctx, move.GetAfterContext().GetBeforeInstance())
					if err != nil {
						return err
					}
					lines = append(lines, fmt.Sprintf(" %v\n", bef.GetRelease().Title))
				}
				lines = append(lines, fmt.Sprintf(" %v\n", record.GetRelease().Title))
				if move.GetAfterContext().GetAfter() != nil {
					aft, err := s.bridge.getRecord(ctx, move.GetAfterContext().GetAfterInstance())
					if err != nil {
						return err
					}
					lines = append(lines, fmt.Sprintf(" %v\n", aft.GetRelease().Title))
				}

				err := s.bridge.print(ctx, lines)
				if err != nil {
					return err
				}
				marked = true
			}

			// Only clear SOLD records
			if marked || record.GetMetadata().Category == pbrc.ReleaseMetadata_SOLD ||
				record.GetMetadata().Category == pbrc.ReleaseMetadata_SOLD_ARCHIVE {
				err := s.bridge.clearMove(ctx, move)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil

}
