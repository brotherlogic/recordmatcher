package main

import (
	"context"
	"testing"

	pb "github.com/brotherlogic/recordmatcher/proto"
)

func TestBasicAPI(t *testing.T) {
	s := InitTest()
	_, err := s.Match(context.Background(), &pb.MatchRequest{InstanceId: 12})
	if err != nil {
		t.Errorf("Bad Match: %v", err)
	}
}
