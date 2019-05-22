package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
)

type getter interface {
	getRecords(ctx context.Context) ([]*pbrc.Record, error)
	update(ctx context.Context, r *pbrc.Record) error
}

func (s *Server) processRecords(ctx context.Context) error {
	count := 0
	startTime := time.Now()
	recs, err := s.getter.getRecords(ctx)

	if err != nil {
		return err
	}

	matches := make(map[int32][]*pbrc.Record)
	for _, r := range recs {
		if r.GetRelease().MasterId > 0 {
			matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
		}
	}

	for parent, records := range matches {
		if len(records) > 2 {
			s.Log(fmt.Sprintf("Found super match: %v", parent))
		}

		if len(records) == 2 {

			if len(records[0].GetRelease().Tracklist) == len(records[1].GetRelease().Tracklist) {
				if records[0].GetMetadata().Match != pbrc.ReleaseMetadata_FULL_MATCH {
					records[0].GetMetadata().Match = pbrc.ReleaseMetadata_FULL_MATCH
					err := s.getter.update(ctx, records[0])
					if err != nil {
						return err
					}
				}
			}
		}

		if len(records) == 1 && !records[0].GetMetadata().NeedsStockCheck && time.Now().Sub(time.Unix(records[0].GetMetadata().LastStockCheck, 0)) > time.Hour*24*30*6 && records[0].GetMetadata().Keep != pbrc.ReleaseMetadata_KEEPER {
			records[0].GetMetadata().NeedsStockCheck = true
			err := s.getter.update(ctx, records[0])
			if err != nil {
				return err
			}
		}
	}

	s.count = count
	s.Log(fmt.Sprintf("Processed in %v", time.Now().Sub(startTime)))
	return nil
}
