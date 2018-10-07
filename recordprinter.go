package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/goserver/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbgd "github.com/brotherlogic/godiscogs"
	pbg "github.com/brotherlogic/goserver/proto"
	pbp "github.com/brotherlogic/printer/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbrm "github.com/brotherlogic/recordmover/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
)

// Bridge link to other services
type Bridge interface {
	getMoves(ctx context.Context) ([]*pbrm.RecordMove, error)
	clearMove(ctx context.Context, move *pbrm.RecordMove) error
	print(ctx context.Context, lines []string) error
	resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error)
}

type prodBridge struct{}

func getRecord(ctx context.Context, instanceID int32) *pbrc.Record {
	host, port, err := utils.Resolve("recordcollection")
	if err != nil {
		log.Fatalf("Unable to reach recordcollection: %v", err)
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	r, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbgd.Release{InstanceId: instanceID}}})
	if err != nil {
		log.Fatalf("Unable to get records: %v", err)
	}

	if len(r.GetRecords()) == 0 {
		log.Fatalf("Unable to get record: %v", instanceID)
	}
	return r.GetRecords()[0]
}

func getFolder(ctx context.Context, folderID int32) (string, error) {
	host, port, err := utils.Resolve("recordsorganiser")
	if err != nil {
		return "", err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return "", err
	}

	client := pbro.NewOrganiserServiceClient(conn)
	r, err := client.GetQuota(ctx, &pbro.QuotaRequest{FolderId: folderID})
	if err != nil {
		return "", err
	}

	return r.LocationName, nil
}

func getLocation(ctx context.Context, rec *pbrc.Record) ([]string, error) {
	host, port, err := utils.Resolve("recordsorganiser")
	if err != nil {
		return []string{}, err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return []string{}, err
	}

	client := pbro.NewOrganiserServiceClient(conn)
	location, err := client.Locate(ctx, &pbro.LocateRequest{InstanceId: rec.GetRelease().InstanceId})
	str := []string{}
	if err != nil {
		str = append(str, fmt.Sprintf("Unable to locate instance (%v) because %v\n", rec.GetRelease().InstanceId, err))
	} else {
		for i, r := range location.GetFoundLocation().GetReleasesLocation() {
			if r.GetInstanceId() == rec.GetRelease().InstanceId {
				str = append(str, fmt.Sprintf("  Slot %v\n", r.GetSlot()))
				if i > 0 {
					str = append(str, fmt.Sprintf("  %v. %v\n", i-1, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i-1].InstanceId)))
				}
				str = append(str, fmt.Sprintf("  %v. %v\n", i, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i].InstanceId)))
				if i < len(location.GetFoundLocation().GetReleasesLocation())-1 {
					str = append(str, fmt.Sprintf("  %v. %v\n", i+1, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i+1].InstanceId)))
				}
			}
		}
	}
	return str, nil
}

func getReleaseString(instanceID int32) string {
	host, port, err := utils.Resolve("recordcollection")
	if err != nil {
		log.Fatalf("Unable to reach collection: %v", err)
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	rel, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Force: true, Filter: &pbrc.Record{Release: &pbgd.Release{InstanceId: instanceID}}})
	if err != nil {
		log.Fatalf("unable to get record: %v", err)
	}
	return rel.GetRecords()[0].GetRelease().Title + " [" + strconv.Itoa(int(instanceID)) + "]"
}

func (p *prodBridge) resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error) {
	r := getRecord(ctx, move.InstanceId)
	f1, err := getFolder(ctx, move.FromFolder)
	if err != nil {
		return []string{}, err
	}

	f2, err := getFolder(ctx, move.ToFolder)
	if err != nil {
		return []string{}, err
	}

	loc, err := getLocation(ctx, r)
	if err != nil {
		return []string{}, err
	}

	strret := []string{fmt.Sprintf("%v: %v -> %v\n", r.GetRelease().Title, f1, f2)}
	for _, v := range loc {
		strret = append(strret, v)
	}
	return strret, nil
}

func (p *prodBridge) getMoves(ctx context.Context) ([]*pbrm.RecordMove, error) {
	host, port, err := utils.Resolve("recordmover")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	client := pbrm.NewMoveServiceClient(conn)
	resp, err := client.ListMoves(ctx, &pbrm.ListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Moves, err
}

func (p *prodBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	host, port, err := utils.Resolve("recordmover")
	if err != nil {
		return err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return err
	}

	client := pbrm.NewMoveServiceClient(conn)
	_, err = client.ClearMove(ctx, &pbrm.ClearRequest{InstanceId: move.InstanceId})
	return err
}

func (p *prodBridge) print(ctx context.Context, lines []string) error {
	host, port, err := utils.Resolve("printer")
	if err != nil {
		return err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return err
	}

	client := pbp.NewPrintServiceClient(conn)
	_, err = client.Print(ctx, &pbp.PrintRequest{Lines: lines})
	return err
}

//Server main server type
type Server struct {
	*goserver.GoServer
	bridge    Bridge
	count     int64
	lastCount time.Time
}

// Init builds the server
func Init() *Server {
	s := &Server{
		&goserver.GoServer{},
		&prodBridge{},
		0,
		time.Unix(0, 0),
	}
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

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{
		&pbg.State{Key: "count", Value: s.count},
		&pbg.State{Key: "last_count", Text: fmt.Sprintf("%v", s.lastCount)},
	}
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
	server.PrepServer()
	server.Register = server
	server.RegisterServer("recordprinter", false)

	server.RegisterRepeatingTask(server.moveLoop, "move_loop", time.Hour)

	server.Log("Starting!")
	fmt.Printf("%v", server.Serve())
}
