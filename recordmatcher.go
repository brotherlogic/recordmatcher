package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbgd "github.com/brotherlogic/godiscogs"
	pbg "github.com/brotherlogic/goserver/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordmatcher/proto"
)

const (
	// KEY used to save scores
	KEY = "github.com/brotherlogic/recordmatcher/config"
)

//Server main server type
type Server struct {
	*goserver.GoServer
	getter  getter
	lastMap map[int32]time.Time
}

type prodGetter struct {
	dial func(ctx context.Context, server string) (*grpc.ClientConn, error)
	log  func(log string)
}

func (p prodGetter) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	// Validate this record so that we at least get the right cache details
	req := &pbrc.GetRecordRequest{InstanceId: instanceID, Validate: true}
	resp, err := client.GetRecord(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetRecord(), nil
}

func (p prodGetter) getRecordsSince(ctx context.Context, t int64) ([]int32, error) {
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Origin: "recordmatcher-since", Query: &pbrc.QueryRecordsRequest_UpdateTime{t}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) getRecordsWithMaster(ctx context.Context, m int32) ([]int32, error) {
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Origin: "recordmatcher-master", Query: &pbrc.QueryRecordsRequest_MasterId{m}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) getRecordsWithID(ctx context.Context, i int32) ([]int32, error) {
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Origin: "recordmatcher-withid", Query: &pbrc.QueryRecordsRequest_ReleaseId{i}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) update(ctx context.Context, i int32, match pbrc.ReleaseMetadata_MatchState, existing pbrc.ReleaseMetadata_MatchState, source string) error {
	p.log(fmt.Sprintf("%v: %v -> %v", i, match, existing))
	if match == existing {
		return nil
	}
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	_, err = client.UpdateRecord(ctx, &pbrc.UpdateRecordRequest{Reason: "recordmatch", Requestor: "recordmatcher-" + source, Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: i}, Metadata: &pbrc.ReleaseMetadata{Match: match}}})
	if err != nil {
		if status.Convert(err).Code() != codes.FailedPrecondition {
			return err
		}
	}
	return nil
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
		lastMap:  make(map[int32]time.Time),
	}
	s.getter = &prodGetter{dial: s.FDialServer, log: s.Log}
	return s
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	pb.RegisterRecordMatcherServiceServer(server, s)
	rcpb.RegisterClientUpdateServiceServer(server, s)
}

// ReportHealth alerts if we're not healthy
func (s *Server) ReportHealth() bool {
	return true
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{}
}

func main() {
	var quiet = flag.Bool("quiet", false, "Show all output")
	flag.Parse()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	server := Init()
	server.PrepServer("recordmatcher")
	server.Register = server

	err := server.RegisterServerV2(false)
	if err != nil {
		return
	}

	server.Serve()
}
