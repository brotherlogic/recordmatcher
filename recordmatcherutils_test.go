package main

import (
	"testing"

	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordprocess/proto"
)

type testGetter struct {
	lastCategory pbrc.ReleaseMetadata_Category
	rec          *pbrc.Record
	sold         *pbrc.Record
}

func (t *testGetter) getRecords(ctx context.Context) ([]*pbrc.Record, error) {
	return []*pbrc.Record{t.rec}, nil
}

func (t *testGetter) update(ctx context.Context, r *pbrc.Record) error {
	t.lastCategory = r.GetMetadata().Category
	return nil
}

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.getter = &testGetter{}
	s.scores = &pb.Scores{}
	s.GoServer.KSclient = *keystoreclient.GetTestClient(".testing")

	return s
}

func TestBasicTest(t *testing.T) {
	s := InitTest()
	s.processRecords(context.Background())
}
