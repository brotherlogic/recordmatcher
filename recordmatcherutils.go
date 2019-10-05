package main

import (
	"time"

	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

type getter interface {
	getRecords(ctx context.Context) ([]*pbrc.Record, error)
	update(ctx context.Context, r *pbrc.Record) error
}

func (s *Server) requiresStockCheck(ctx context.Context, r *pbrc.Record) bool {
	if len(r.GetMetadata().CdPath) > 0 {
		return false
	}

	return true
}

func (s *Server) processRecords(ctx context.Context) error {
	count := 0
	recs, err := s.getter.getRecords(ctx)

	if err != nil {
		return err
	}

	matches := make(map[int32][]*pbrc.Record)
	trackNumbers := make(map[int32]int)
	for _, r := range recs {
		if r.GetRelease().MasterId > 0 {
			matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
		} else {
			matches[r.GetRelease().Id] = append(matches[r.GetRelease().Id], r)
		}

		trackNumbers[r.GetRelease().InstanceId] = 0
		for _, track := range r.GetRelease().Tracklist {
			if track.TrackType == pbgd.Track_TRACK {
				trackNumbers[r.GetRelease().InstanceId]++
			}
		}
	}

	for _, records := range matches {
		if len(records) == 2 {
			if trackNumbers[records[0].GetRelease().InstanceId] == trackNumbers[records[1].GetRelease().InstanceId] {
				if records[0].GetMetadata().Match != pbrc.ReleaseMetadata_FULL_MATCH {
					records[0].GetMetadata().Match = pbrc.ReleaseMetadata_FULL_MATCH
					err := s.getter.update(ctx, records[0])
					if err != nil {
						return err
					}
				}
			}
		}

		if len(records) == 1 && !records[0].GetMetadata().NeedsStockCheck && time.Now().Sub(time.Unix(records[0].GetMetadata().LastStockCheck, 0)) > time.Hour*24*30*6 && records[0].GetMetadata().Keep != pbrc.ReleaseMetadata_KEEPER && s.requiresStockCheck(ctx, records[0]) {
			records[0].GetMetadata().NeedsStockCheck = true
			err := s.getter.update(ctx, records[0])
			if err != nil {
				return err
			}
		}
	}

	s.count = count
	s.config.LastRun = time.Now().Unix()
	return nil
}
