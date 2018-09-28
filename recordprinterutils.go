package main

import (
	"fmt"

	"golang.org/x/net/context"
)

func (s *Server) moveLoop(ctx context.Context) {
	moves, err := s.bridge.getMoves(ctx)

	if err != nil {
		s.Log(fmt.Sprintf("Error getting moves: %v", err))
		return
	}

	for _, move := range moves {
		s.Log(fmt.Sprintf("MOVE: %v", move.InstanceId))
		/*err := s.bridge.print(ctx, fmt.Sprintf("MOVE: %v", move.InstanceId))

		if err != nil {
			s.Log(fmt.Sprintf("Error printing move: %v", err))
			return
		}*/
	}
}
