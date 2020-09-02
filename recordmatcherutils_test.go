package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

type testGetter struct {
	fail          bool
	rec           []*pbrc.Record
	lastUpdated   int32
	updateFail    bool
	getFail       bool
	getMasterFail bool
	failNum       int32
	idFail        bool
	noMatch       bool
}

func (t *testGetter) getRecords(ctx context.Context) ([]*pbrc.Record, error) {
	if t.fail {
		return nil, fmt.Errorf("Build to fail")
	}
	return t.rec, nil
}

func (t *testGetter) update(ctx context.Context, i int32, match, existing pbrc.ReleaseMetadata_MatchState, source string) error {
	if t.updateFail {
		return fmt.Errorf("Built to fail")
	}
	t.lastUpdated = i
	return nil
}

func (t *testGetter) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	if t.getFail {
		return nil, fmt.Errorf("Built to fail")
	}
	if t.failNum == instanceID && t.failNum != 0 {
		return nil, fmt.Errorf("FAIL")
	}

	for _, rec := range t.rec {
		if rec.GetRelease().GetInstanceId() == instanceID {
			return rec, nil
		}
	}

	return t.rec[0], nil
}

func (t *testGetter) getRecordsSince(ctx context.Context, ti int64) ([]int32, error) {
	if t.fail {
		return []int32{}, fmt.Errorf("Built to fail")
	}
	return []int32{t.rec[0].GetRelease().InstanceId}, nil
}

func (t *testGetter) getRecordsWithMaster(ctx context.Context, m int32) ([]int32, error) {
	if t.getMasterFail {
		return []int32{}, fmt.Errorf("Built to fail")
	}
	if t.noMatch {
		return []int32{}, nil
	}

	if len(t.rec) > 1 {
		vals := []int32{}
		for _, id := range t.rec[1:] {
			vals = append(vals, id.GetRelease().InstanceId)
		}
		return vals, nil
	}
	return []int32{t.rec[0].GetRelease().InstanceId}, nil
}
func (t *testGetter) getRecordsWithID(ctx context.Context, i int32) ([]int32, error) {
	if t.idFail {
		return []int32{}, fmt.Errorf("Built to fail")
	}
	if len(t.rec) > 1 {
		return []int32{t.rec[1].GetRelease().InstanceId}, nil
	}
	return []int32{t.rec[0].GetRelease().InstanceId}, nil

}

func InitTest() *Server {
	s := Init()
	s.SkipLog = true
	s.getter = &testGetter{failNum: 1, rec: []*pbrc.Record{&pbrc.Record{
		Metadata: &pbrc.ReleaseMetadata{},
		Release:  &pbgd.Release{InstanceId: 123, MasterId: 12, Tracklist: []*pbgd.Track{&pbgd.Track{TrackType: pbgd.Track_TRACK}}},
	},
		&pbrc.Record{Release: &pbgd.Release{InstanceId: 125}}}}
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

func TestVeryBasicTestTrackMatch(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 11, MasterId: 123, FolderId: 242017}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 12, MasterId: 123, FolderId: 242017}},
	}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}
func TestVeryBasicTestTrackMatchWithTwo(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 15, MasterId: 123, Tracklist: []*pbgd.Track{&pbgd.Track{Title: "Test", TrackType: pbgd.Track_TRACK}}}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 11, MasterId: 123, FolderId: 242017, Tracklist: []*pbgd.Track{&pbgd.Track{Title: "Test", TrackType: pbgd.Track_TRACK}}}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 12, MasterId: 123, FolderId: 242017}},
	}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestVeryBasicTestTrackMatchNoMatch(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{noMatch: true, rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{MasterId: 123}},
	}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Failed: %v", err)
	}
}

func TestVeryBasicTestTrackMatchSingle(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{LastStockCheck: time.Now().Unix()}, Release: &pbgd.Release{MasterId: 123}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{LastStockCheck: time.Now().Unix()}, Release: &pbgd.Release{InstanceId: 11, MasterId: 123, FolderId: 242017}},
	}}
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
	tu := &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{Id: 123, InstanceId: 123, MasterId: 123}, Metadata: &pbrc.ReleaseMetadata{}}}}
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

func TestGetRecFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{getFail: true, rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123}}}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}
func TestGetRecMasterFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{getMasterFail: true, rec: []*pbrc.Record{&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123, MasterId: 123}}}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}
func TestGetRecMasterGetFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{failNum: 125, rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123, MasterId: 123}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 125, MasterId: 123}},
	}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}
func TestGetRecWithIdFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{idFail: true, rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123}},
	}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}

func TestGetRecWithIdGetFail(t *testing.T) {
	s := InitTest()
	s.getter = &testGetter{failNum: 125, rec: []*pbrc.Record{
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 123}},
		&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{}, Release: &pbgd.Release{InstanceId: 125}},
	}}
	err := s.processRecords(context.Background())
	if err == nil {
		t.Errorf("Did not fail")
	}
}
