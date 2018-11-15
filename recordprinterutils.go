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

		lines, err := s.bridge.resolve(ctx, move)
		if err != nil {
			s.Log(fmt.Sprintf("Error getting move: %v", err))
			return
		}
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
