package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func (s *Server) processRecords(ctx context.Context) error {
	recs, err := s.getter.getRecordsSince(ctx, 0)

	if err != nil {
		return err
	}

	return s.processRecordList(ctx, recs, "repeat")
}

func (s *Server) processRecordList(ctx context.Context, recs []int32, source string) error {
	for _, r := range recs {
		val, ok := s.lastMap[r]
		if ok && time.Since(val) < time.Hour*24 {
			s.Log(fmt.Sprintf("Skipping match as we have done this recently: %v", val))
			return nil
		}
	}
	for _, r := range recs {
		s.lastMap[r] = time.Now()
	}
	matches := make(map[int32][]*pbrc.Record)
	trackNumbers := make(map[int32]int)
	ll := ""
	for _, id := range recs {
		r, err := s.getter.getRecord(ctx, id)

		// We don't try and match a sold record
		if r.GetMetadata().GetCategory() == pbrc.ReleaseMetadata_STAGED_TO_SELL ||
			r.GetMetadata().GetCategory() == pbrc.ReleaseMetadata_SOLD_ARCHIVE {
			return nil
		}

		if err != nil {
			if status.Convert(err).Code() == codes.OutOfRange {
				return nil
			}
			return err
		}

		if r.GetRelease().MasterId > 0 {
			mrecs, err := s.getter.getRecordsWithMaster(ctx, r.GetRelease().MasterId)
			if err != nil {
				return err
			}

			ll = fmt.Sprintf("MID,%v", len(mrecs))
			if len(mrecs) == 0 {
				s.Log(fmt.Sprintf("Could not find any master ids for %v", r.GetRelease().GetInstanceId()))
			}

			for _, mrec := range mrecs {
				r, err = s.getter.getRecord(ctx, mrec)
				// This is a deleted record
				if status.Convert(err).Code() == codes.OutOfRange {
					continue
				}
				if err != nil {
					return err
				}

				if r.GetMetadata().GetCategory() != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
					r.GetMetadata().GetCategory() != pbrc.ReleaseMetadata_SOLD_ARCHIVE {
					matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
				}
			}
		} else {
			mrecs, err := s.getter.getRecordsWithID(ctx, r.GetRelease().Id)
			if err != nil {
				return err
			}
			ll = fmt.Sprintf("WID,%v", len(mrecs))

			for _, mrec := range mrecs {
				r, err = s.getter.getRecord(ctx, mrec)
				// This is a deleted record
				if status.Convert(err).Code() == codes.OutOfRange {
					continue
				}
				if err != nil {
					return err
				}
				if r.GetMetadata().GetCategory() != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
					r.GetMetadata().GetCategory() != pbrc.ReleaseMetadata_SOLD_ARCHIVE {
					matches[r.GetRelease().MasterId] = append(matches[r.GetRelease().MasterId], r)
				}
			}

		}
	}

	for _, recs := range matches {
		for _, r := range recs {
			trackNumbers[r.GetRelease().InstanceId] = 0
			for _, track := range r.GetRelease().Tracklist {
				if track.TrackType == pbgd.Track_TRACK {
					trackNumbers[r.GetRelease().InstanceId]++
				}
			}
		}
	}

	lens := fmt.Sprintf("%v:%v:", len(matches), ll)
	for i, records := range matches {
		lens += fmt.Sprintf(" %v->%v ", i, len(records))

		for i := 1; i < len(records); i++ {
			lens += fmt.Sprintf(" adding %v and %v from %v", trackNumbers[records[0].GetRelease().InstanceId], trackNumbers[records[i].GetRelease().InstanceId], trackNumbers)
			if trackNumbers[records[0].GetRelease().InstanceId] <= trackNumbers[records[i].GetRelease().InstanceId] {
				if records[0].GetMetadata().Match != pbrc.ReleaseMetadata_FULL_MATCH {
					return s.getter.update(ctx, records[0].GetRelease().InstanceId, pbrc.ReleaseMetadata_FULL_MATCH, records[0].GetMetadata().GetMatch(), source)
				}
				return nil
			}
		}

		if len(records) >= 2 {
			if records[0].GetMetadata().Match != pbrc.ReleaseMetadata_NO_MATCH {
				return s.getter.update(ctx, records[0].GetRelease().InstanceId, pbrc.ReleaseMetadata_NO_MATCH, records[0].GetMetadata().GetMatch(), source)
			}
			return nil
		}

		if len(records) == 1 {
			//No match found
			s.Log(fmt.Sprintf("FOUND NO MATCH %v -> %v", recs, lens))
			return s.getter.update(ctx, records[0].GetRelease().GetInstanceId(), pbrc.ReleaseMetadata_NO_MATCH, records[0].GetMetadata().GetMatch(), source)
		}
	}

	return fmt.Errorf("No match state appropriate for %v: %v", recs, lens)
}
