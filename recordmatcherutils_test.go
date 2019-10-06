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

func (t *testGetter) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	return t.rec[0], nil
}

func (t *testGetter) getRecordsSince(ctx context.Context, ti int64) ([]int32, error) {
	if t.fail {
		return []int32{}, fmt.Errorf("Built to fail")
	}
	return []int32{t.rec[0].GetRelease().InstanceId}, nil
}

func (t *testGetter) getRecordsWithMaster(ctx context.Context, m int32) ([]int32, error) {
	return []int32{t.rec[0].GetRelease().InstanceId}, nil
}
func (t *testGetter) getRecordsWithId(ctx context.Context, i int32) ([]int32, error) {
	return []int32{t.rec[0].GetRelease().InstanceId}, nil
}

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123}}}}
	s.GoServer.KSclient = *keystoreclient.GetTestClient(".testing")

	return s
}

func TestFailStock(t *testing.T) {
	s := InitTest()
	if s.requiresStockCheck(context.Background(), &pbrc.Record{Metadata: &pbrc.ReleaseMetadata{CdPath: "blah"}}) {
		t.Errorf("Ripped record needed stock check")
	}
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
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123, FolderId: 242017, Tracklist: []*pbgd.Track{&pbgd.Track{Title: "Test", TrackType: pbgd.Track_TRACK}}}}}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}
func TestBasicNoMasterTest(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{Id: 123}}, &pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{Id: 123, FolderId: 242017, Tracklist: []*pbgd.Track{&pbgd.Track{Title: "Test", TrackType: pbgd.Track_TRACK}}}}}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestBasicTestWithFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{updateFail: true, rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123, FolderId: 242017}}}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
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
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123}}}}
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
