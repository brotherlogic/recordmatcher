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
	fail bool
	rec  []*pbrc.Record
}

func (t *testGetter) getRecords(ctx context.Context) ([]*pbrc.Record, error) {
	if t.fail {
		return nil, fmt.Errorf("Build to fail")
	}
	return t.rec, nil
}

func (t *testGetter) update(ctx context.Context, r *pbrc.Record) error {
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
	s.getter = &testGetter{rec: []*pbrc.Record{&pbrc.Record{Release: &pbgd.Release{MasterId: 123}}, &pbrc.Record{Release: &pbgd.Release{MasterId: 123}}}}
	err := s.processRecords(context.Background())
	if err != nil {
		t.Errorf("Failed: %v", err)
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
