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
	startTime := time.Now()
	recs, err := s.getter.getRecords(ctx)

	if err != nil {
		return err
	}

	matches := make(map[int32][]*pbrc.Record)
	for _, r := range recs {
		matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
	}

	for parent, records := range matches {
		if len(records) > 2 {
			s.Log(fmt.Sprintf("Found super match: %v", parent))
		}

		if len(records) == 2 {
			if len(records[0].GetRelease().Tracklist) == len(records[1].GetRelease().Tracklist) && (records[0].GetRelease().FolderId == 242017 || records[1].GetRelease().FolderId == 242017) {
				s.Log(fmt.Sprintf("Found equal match: %v", parent))
			}
		}
	}

	s.Log(fmt.Sprintf("Processed in %v", time.Now().Sub(startTime)))
	return nil
}
