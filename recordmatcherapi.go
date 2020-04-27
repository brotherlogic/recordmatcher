package main

import (
	"golang.org/x/net/context"

	pb "github.com/brotherlogic/recordmatcher/proto"
)

// Match - force matches a record
func (s *Server) Match(ctx context.Context, req *pb.MatchRequest) (*pb.MatchResponse, error) {
	return &pb.MatchResponse{}, s.processRecordList(ctx, []int32{req.GetInstanceId()}, "api")
}
