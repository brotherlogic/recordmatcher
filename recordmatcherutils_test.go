package main

import (
	"fmt"
	"testing"

	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

type testGetter struct {
	fail        bool
	rec         []*pbrc.Record
	lastUpdated int32
	updateFail  bool
}

func (t *testGetter) getRecords(ctx context.Context) ([]*pbrc.Record, error) {
	if t.fail {
		return nil, fmt.Errorf("Build to fail")
	}
	return t.rec, nil
}

func (t *testGetter) update(ctx context.Context, r *pbrc.Record) error {
	if t.updateFail {
		return fmt.Errorf("Built to fail")
	}
	t.lastUpdated = r.GetRelease().Id
	return nil
}

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.getter = &testGetter{}
	s.GoServer.KSclient = *keystoreclient.GetTestClient(".testing")

	return s
}

func TestVeryBasicTest(t *testing.T) {
	s := InitTest()
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestBasicTest(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123, FolderId: 242017}}}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestNeedsStockCheck(t *testing.T) {
	s := InitTest()
	tu := &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{Id: 123, MasterId: 123}, Metadata: &pbrc.ReleaseMetadata{}}}}
	s.getter = tu
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}

	if tu.lastUpdated != 123 {
		t.Errorf("Update did not occur: %v", tu.lastUpdated)
	}
}

func TestNeedsStockCheckWithUpdateFail(t *testing.T) {
	s := InitTest()
	tu := &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{Id: 123, MasterId: 123}, Metadata: &pbrc.ReleaseMetadata{}}}, updateFail: true}
	s.getter = tu
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Processing did not fail")
	}
}

func TestBasicTestSuper(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123}}}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestGetFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{fail: true}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}
