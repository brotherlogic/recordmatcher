package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/brotherlogic/goserver/utils"

	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordmatcher/proto"
)

func main() {
	ctx, cancel := utils.BuildContext("recordmatcher-cli", "recordmatcher")
	defer cancel()

	conn, err := utils.LFDialServer(ctx, "recordmatcher")
	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewRecordMatcherServiceClient(conn)

	switch os.Args[1] {
	case "match":
		addFlags := flag.NewFlagSet("AddRecords", flag.ExitOnError)
		var id = addFlags.Int("id", -1, "Id of the record to add")

		if err := addFlags.Parse(os.Args[2:]); err == nil {
			_, err := client.Match(ctx, &pb.MatchRequest{Force: true, InstanceId: int32(*id)})
			if err != nil {
				log.Fatalf("Error on Add Record: %v", err)
			}
			fmt.Printf("Match processed\n")
		}

	case "fullping":
		conn2, err2 := utils.LFDialServer(ctx, "recordcollection")
		if err2 != nil {
			log.Fatalf("Can't dial RC: %v", err2)
		}
		rcclient := rcpb.NewRecordCollectionServiceClient(conn2)
		ids, err := rcclient.QueryRecords(ctx, &rcpb.QueryRecordsRequest{Query: &rcpb.QueryRecordsRequest_FolderId{242017}})
		if err != nil {
			log.Fatalf("Err: %v", err)
		}

		sclient := rcpb.NewClientUpdateServiceClient(conn)

		for i, id := range ids.GetInstanceIds() {
			_, err = sclient.ClientUpdate(ctx, &rcpb.ClientUpdateRequest{InstanceId: int32(id)})
			if err != nil {
				log.Fatalf("Error on GET: %v", err)
			}
		}
	}
}
