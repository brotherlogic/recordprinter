package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbrm "github.com/brotherlogic/recordmover/proto"
)

func (s *Server) moveLoop(ctx context.Context, id int32) error {
	moves, err := s.bridge.getMoves(ctx, id)
	if err == nil {
		for _, move := range moves {
			err = s.move(ctx, move)
			if err != nil {
				break
			}
		}
	}

	return err
}

func (s *Server) buildMove(ctx context.Context, record *pbrc.Record, move *pbrm.RecordMove) []string {
	lines := []string{}
	surrounds := move.GetAfterContext()

	// Don't show after context for sales
	if move.GetAfterContext().GetLocation() == "Sell" || move.GetAfterContext().GetLocation() == "Sold" {
		surrounds = nil
	}

	// If we're moving into the LP, use the before context
	if move.GetAfterContext().GetLocation() == "Listening Pile" {
		surrounds = move.GetBeforeContext()
	}

	if surrounds != nil {
		lines = append(lines, fmt.Sprintf("Slot %v", surrounds.GetSlot()))
		if surrounds.GetBeforeInstance() != 0 {
			bef, _ := s.bridge.getRecord(ctx, move.GetAfterContext().GetBeforeInstance())
			lines = append(lines, fmt.Sprintf(" %v", bef.GetRelease().GetTitle()))
		}
		lines = append(lines, fmt.Sprintf(" %v", record.GetRelease().Title))
		if move.GetAfterContext().GetAfterInstance() != 0 {
			aft, _ := s.bridge.getRecord(ctx, move.GetAfterContext().GetAfterInstance())
			lines = append(lines, fmt.Sprintf(" %v", aft.GetRelease().GetTitle()))
		}
	}

	return lines
}

func (s *Server) move(ctx context.Context, move *pbrm.RecordMove) error {
	s.currMove = move.InstanceId

	if move.GetBeforeContext().GetLocation() != "" && move.GetAfterContext().GetLocation() != "" {

		// Short circuit if this is a within folder move
		if move.GetBeforeContext().GetLocation() == move.GetAfterContext().GetLocation() {
			if move.GetToFolder() == move.GetFromFolder() {
				return nil
			}
			s.RaiseIssue("Weird Move", fmt.Sprintf("%v is a weird move", move))
			return nil
		}

		record, err := s.bridge.getRecord(ctx, move.InstanceId)
		if err != nil {
			return err
		}

		artistName := "Unknown Artist"
		if len(record.GetRelease().GetArtists()) > 0 {
			artistName = record.GetRelease().GetArtists()[0].GetName()
		}
		lines := []string{fmt.Sprintf("%v - %v: %v -> %v\n", artistName, record.GetRelease().Title, move.GetBeforeContext().GetLocation(), move.GetAfterContext().GetLocation())}

		// Also add in the after surrounds
		surrounds := move.GetAfterContext()

		if move.GetAfterContext().GetLocation() == "Listening Pile" {
			surrounds = move.GetBeforeContext()
			if surrounds.GetBeforeInstance() == 0 && surrounds.GetAfterInstance() == 0 {
				s.RaiseIssue("Weird Move", fmt.Sprintf("%v has not before context", move))
				return nil
			}
		}

		addlines := s.buildMove(ctx, record, move)
		lines = append(lines, addlines...)

		// Don't print bandcamp moves, unless they're into digital
		// Don't print moves to stale sales
		// Don't print 12 inch moves (handled by STO)
		// Don't print moves into library records
		if record.GetMetadata().GetGoalFolder() != 1782105 &&
			record.GetMetadata().GetGoalFolder() != 2274270 &&
			move.GetToFolder() != 268147 &&
			move.GetToFolder() != 1708299 &&
			move.GetToFolder() != 242017 &&
			move.GetAfterContext().GetLocation() != "Library Records" &&
			move.GetAfterContext().GetLocation() != "Keepers" &&
			!strings.Contains(move.GetAfterContext().GetLocation(), "Boxed") {
			err = s.bridge.print(ctx, lines, move, true)
			if err != nil {
				return err
			}
		}

		err = s.bridge.clearMove(ctx, move)
		if err != nil {
			return err
		}
	} else {
		s.RaiseIssue("Record Print Issue", fmt.Sprintf("Move for %v is not able to be printed", move.GetInstanceId()))
	}

	return nil
}
