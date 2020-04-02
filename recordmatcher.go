package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbg "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordmatcher/proto"
)

const (
	// KEY used to save scores
	KEY = "github.com/brotherlogic/recordmatcher/config"
)

//Server main server type
type Server struct {
	*goserver.GoServer
	getter getter
	config *pb.Config
	count  int
}

type prodGetter struct {
	dial func(server string) (*grpc.ClientConn, error)
}

func (p prodGetter) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.GetRecordRequest{InstanceId: instanceID}
	resp, err := client.GetRecord(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetRecord(), nil
}

func (p prodGetter) getRecordsSince(ctx context.Context, t int64) ([]int32, error) {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_UpdateTime{t}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) getRecordsWithMaster(ctx context.Context, m int32) ([]int32, error) {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_MasterId{m}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) getRecordsWithID(ctx context.Context, i int32) ([]int32, error) {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	req := &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_ReleaseId{i}}
	resp, err := client.QueryRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetInstanceIds(), nil
}

func (p prodGetter) update(ctx context.Context, r *pbrc.Record) error {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	_, err = client.UpdateRecord(ctx, &pbrc.UpdateRecordRequest{Requestor: "recordmatcher", Update: r})
	if err != nil {
		return err
	}
	return nil
}

// Init builds the server
func Init() *Server {
	s := &Server{
		GoServer: &goserver.GoServer{},
		config:   &pb.Config{},
	}
	s.getter = &prodGetter{s.DialMaster}
	s.GoServer.KSclient = *keystoreclient.GetClient(s.DialMaster)
	return s
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	//Pass
}

// ReportHealth alerts if we're not healthy
func (s *Server) ReportHealth() bool {
	return true
}

func (s *Server) saveConfig(ctx context.Context) {
	s.KSclient.Save(ctx, KEY, s.config)
}

func (s *Server) readConfig(ctx context.Context) error {
	config := &pb.Config{}
	data, _, err := s.KSclient.Read(ctx, KEY, config)

	if err != nil {
		return err
	}

	s.config = data.(*pb.Config)
	return nil
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.saveConfig(ctx)
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	if master {
		err := s.readConfig(ctx)
		return err
	}

	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{
		&pbg.State{Key: "processed_records", Value: int64(len(s.config.ProcessedRecords))},
		&pbg.State{Key: "found_records", Value: int64(s.count)},
		&pbg.State{Key: "last_run", TimeValue: s.config.LastRun},
	}
}

func main() {
	var quiet = flag.Bool("quiet", false, "Show all output")
	var init = flag.Bool("init", false, "Init the system")
	flag.Parse()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	server := Init()
	server.PrepServer()
	server.Register = server

	server.RegisterServerV2("recordmatcher", false, false)

	if *init {
		ctx, cancel := utils.BuildContext("recordmatcher", "recordmatcher")
		defer cancel()
		server.config.TotalProcessed++
		server.saveConfig(ctx)
		return
	}

	server.RegisterRepeatingTask(server.processRecords, "process_records", time.Second*5)
	server.Serve()
}
