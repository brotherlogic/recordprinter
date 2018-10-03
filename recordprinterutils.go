package main

import (
	"fmt"

	"golang.org/x/net/context"
)

func (s *Server) moveLoop(ctx context.Context) {
	s.count++
	moves, err := s.bridge.getMoves(ctx)

	if err != nil {
		s.Log(fmt.Sprintf("Error getting moves: %v", err))
		return
	}

	for _, move := range moves {
		s.Log(fmt.Sprintf("MOVE: %v", move.InstanceId))
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
