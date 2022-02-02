package main

import (
	"golang.org/x/net/context"

	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordmatcher/proto"
)

// Match - force matches a record
func (s *Server) Match(ctx context.Context, req *pb.MatchRequest) (*pb.MatchResponse, error) {
	return &pb.MatchResponse{}, s.processRecordList(ctx, []int32{req.GetInstanceId()}, "api", req.GetForce())
}

//ClientUpdate on an updated record
func (s *Server) ClientUpdate(ctx context.Context, req *rcpb.ClientUpdateRequest) (*rcpb.ClientUpdateResponse, error) {
	return &rcpb.ClientUpdateResponse{}, s.processRecordList(ctx, []int32{req.GetInstanceId()}, "capi", false)
}
