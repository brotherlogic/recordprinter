package main

import "golang.org/x/net/context"

import rcpb "github.com/brotherlogic/recordcollection/proto"

//ClientUpdate on an updated record
func (s *Server) ClientUpdate(ctx context.Context, req *rcpb.ClientUpdateRequest) (*rcpb.ClientUpdateResponse, error) {
	return &rcpb.ClientUpdateResponse{}, s.moveLoop(ctx, req.GetInstanceId())
}
