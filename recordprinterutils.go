package main

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/context"

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

	s.Log(fmt.Sprintf("Processing %v moves", len(moves)))
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

	if move.GetBeforeContext().GetLocation() != "" && move.GetAfterContext().GetLocation() != "" {

		// Short circuit if this is a within folder move
		if move.GetBeforeContext().GetLocation() == move.GetAfterContext().GetLocation() {
			if move.GetToFolder() == move.GetFromFolder() {
				return nil
			}
			s.RaiseIssue(ctx, "Weird Move", fmt.Sprintf("%v is a weird move", move), false)
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

		if move.GetAfterContext().GetLocation() == "12s" {
			lines = append(lines, fmt.Sprintf("Before: %v", move.GetAfterContext().GetBeforeInstance()))
			lines = append(lines, fmt.Sprintf("After: %v", move.GetAfterContext().GetAfterInstance()))
		}

		err = s.bridge.print(ctx, lines, move, true)
		s.config.LastPrint = time.Now().Unix()
		if err != nil {
			return err
		}

		err = s.bridge.clearMove(ctx, move)
		if err != nil {
			return err
		}

		s.save(ctx)
	} else {
		s.RaiseIssue(ctx, "Record Print Issue", fmt.Sprintf("Move for %v is not able to be printed", move.GetInstanceId()), false)
	}

	return nil
}
