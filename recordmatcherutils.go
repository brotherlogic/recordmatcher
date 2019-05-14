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
	_, err := s.getter.getRecords(ctx)

	if err != nil {
		return err
	}

	s.Log(fmt.Sprintf("Processed in %v", time.Now().Sub(startTime)))
	return nil
}
