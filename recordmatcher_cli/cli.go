package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/brotherlogic/goserver/utils"

	pb "github.com/brotherlogic/recordmatcher/proto"

	//Needed to pull in gzip encoding init
	_ "google.golang.org/grpc/encoding/gzip"
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
			_, err := client.Match(ctx, &pb.MatchRequest{InstanceId: int32(*id)})
			if err != nil {
				log.Fatalf("Error on Add Record: %v", err)
			}
			fmt.Printf("Match processed\n")
		}
	}
}
