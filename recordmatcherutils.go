package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

type getter interface {
	getRecord(ctx context.Context, id int32) (*pbrc.Record, error)
	getRecordsWithMaster(ctx context.Context, masterID int32) ([]int32, error)
	getRecordsWithID(ctx context.Context, id int32) ([]int32, error)
	getRecordsSince(ctx context.Context, t int64) ([]int32, error)
	update(ctx context.Context, i int32, match, existing pbrc.ReleaseMetadata_MatchState, source string) error
}

func (s *Server) requiresStockCheck(ctx context.Context, r *pbrc.Record) bool {
	if len(r.GetMetadata().CdPath) > 0 {
		return false
	}

	return true
}

func (s *Server) processRecords(ctx context.Context) error {
	recs, err := s.getter.getRecordsSince(ctx, s.config.LastRun)

	if err != nil {
		return err
	}

	return s.processRecordList(ctx, recs, "repeat")
}

func (s *Server) processRecordList(ctx context.Context, recs []int32, source string) error {
	matches := make(map[int32][]*pbrc.Record)
	trackNumbers := make(map[int32]int)
	for _, id := range recs {
		r, err := s.getter.getRecord(ctx, id)
		if err != nil {
			return err
		}

		if r.GetRelease().MasterId > 0 {
			mrecs, err := s.getter.getRecordsWithMaster(ctx, r.GetRelease().MasterId)
			if err != nil {
				return err
			}
			for _, mrec := range mrecs {
				r, err = s.getter.getRecord(ctx, mrec)
				if err != nil {
					return err
				}
				matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
			}
		} else {
			mrecs, err := s.getter.getRecordsWithID(ctx, r.GetRelease().Id)
			if err != nil {
				return err
			}
			for _, mrec := range mrecs {
				r, err = s.getter.getRecord(ctx, mrec)
				if err != nil {
					return err
				}
				matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
			}

		}

		trackNumbers[r.GetRelease().InstanceId] = 0
		for _, track := range r.GetRelease().Tracklist {
			if track.TrackType == pbgd.Track_TRACK {
				trackNumbers[r.GetRelease().InstanceId]++
			}
		}

	}

	lens := ""
	for i, records := range matches {
		lens += fmt.Sprintf(" %v->%v ", i, len(records))
		if len(records) == 2 {
			if trackNumbers[records[0].GetRelease().InstanceId] == trackNumbers[records[1].GetRelease().InstanceId] {
				if records[0].GetMetadata().Match != pbrc.ReleaseMetadata_FULL_MATCH {
					return s.getter.update(ctx, records[0].GetRelease().InstanceId, pbrc.ReleaseMetadata_FULL_MATCH, records[0].GetMetadata().GetMatch(), source)
				}
			}
		}

		if len(records) == 1 && !records[0].GetMetadata().NeedsStockCheck && time.Now().Sub(time.Unix(records[0].GetMetadata().LastStockCheck, 0)) > time.Hour*24*30*6 && records[0].GetMetadata().Keep != pbrc.ReleaseMetadata_KEEPER && s.requiresStockCheck(ctx, records[0]) {
			s.Log(fmt.Sprintf("%v needs stock check: %v", records[0].GetRelease().GetInstanceId(), time.Unix(records[0].GetMetadata().GetLastStockCheck(), 0)))
			return s.getter.update(ctx, records[0].GetRelease().InstanceId, pbrc.ReleaseMetadata_MATCH_UNKNOWN, records[0].GetMetadata().GetMatch(), source)
		}

		if len(records) == 1 {
			//No match found
			return s.getter.update(ctx, records[0].GetRelease().InstanceId, pbrc.ReleaseMetadata_NO_MATCH, records[0].GetMetadata().GetMatch(), source)
		}
	}

	s.config.LastRun = time.Now().Unix()
	return fmt.Errorf("No match state appropriate for %v: %v", recs, lens)
}
