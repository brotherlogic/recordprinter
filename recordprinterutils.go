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

	if move.GetBeforeContext() != nil && move.GetAfterContext() != nil {
		record, err := s.bridge.getRecord(ctx, move.InstanceId)
		if err != nil {
			return err
		}

		lines := []string{fmt.Sprintf("%v: %v -> %v\n", record.GetRelease().Title, move.GetFromFolder(), move.GetToFolder())}

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
