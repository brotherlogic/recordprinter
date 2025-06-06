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

	pqc "github.com/brotherlogic/printqueue/client"

	pbgd "github.com/brotherlogic/godiscogs/proto"
	pbg "github.com/brotherlogic/goserver/proto"
	pqcpb "github.com/brotherlogic/printqueue/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pbrm "github.com/brotherlogic/recordmover/proto"
	pb "github.com/brotherlogic/recordprinter/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
)

const (
	// KEY - where the wants are stored
	KEY = "/github.com/brotherlogic/recordprinter/config"
)

// Bridge link to other services
type Bridge interface {
	getMoves(ctx context.Context, id int32) ([]*pbrm.RecordMove, error)
	clearMove(ctx context.Context, move *pbrm.RecordMove) error
	print(ctx context.Context, lines []string, move *pbrm.RecordMove, makeMove bool) error
	resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error)
	getRecord(ctx context.Context, id int32) (*pbrc.Record, error)
}

type prodBridge struct {
	dial       func(ctx context.Context, server string) (*grpc.ClientConn, error)
	raiseIssue func(name string, body string, labels ...string)
	pqc        *pqc.PrintQueueClient
}

func (p *prodBridge) getRecord(ctx context.Context, id int32) (*pbrc.Record, error) {
	if id < -1 {
		return &pbrc.Record{Release: &pbgd.Release{Title: "END OF SECTION"}}, nil
	}
	conn, err := p.dial(ctx, "recordcollection")
	if err != nil {
		return nil, fmt.Errorf("unable to dial rc: %w", err)
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	rel, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
	if err != nil {
		return nil, fmt.Errorf("unable to get record: %w", err)
	}
	return rel.GetRecord(), err
}

func (p *prodBridge) getFolder(ctx context.Context, folderID int32) (string, error) {
	conn, err := p.dial(ctx, "recordsorganiser")
	if err != nil {
		return "", fmt.Errorf("unable to dial ro: %w", err)
	}
	defer conn.Close()

	client := pbro.NewOrganiserServiceClient(conn)
	r, err := client.GetQuota(ctx, &pbro.QuotaRequest{FolderId: folderID})
	if err != nil {
		return "", fmt.Errorf("unable to get quote: %w", err)
	}

	return r.LocationName, nil
}

func (p *prodBridge) getReleaseString(ctx context.Context, instanceID int32) (string, error) {
	rel, err := p.getRecord(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("unable to get record: %w", err)
	}
	return rel.GetRelease().Title + " [" + strconv.Itoa(int(instanceID)) + "]", nil
}

func (p *prodBridge) getLocation(ctx context.Context, rec *pbrc.Record, folder string) ([]string, error) {
	conn, err := p.dial(ctx, "recordsorganiser")
	if err != nil {
		return []string{}, fmt.Errorf("unable to dial ro: %w", err)
	}
	defer conn.Close()

	client := pbro.NewOrganiserServiceClient(conn)
	location, err := client.Locate(ctx, &pbro.LocateRequest{InstanceId: rec.GetRelease().InstanceId})
	str := []string{}
	if err != nil || location.GetFoundLocation().Name != folder {
		return []string{}, fmt.Errorf("Unable to locate instance (%v) because %v and %v given %v", rec.GetRelease().InstanceId, err, location.GetFoundLocation().Name, folder)
	}
	for i, r := range location.GetFoundLocation().GetReleasesLocation() {
		if r.GetInstanceId() == rec.GetRelease().InstanceId {
			str = append(str, fmt.Sprintf("  Slot %v\n", r.GetSlot()))
			if i > 0 {
				rString, err := p.getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i-1].InstanceId)
				if err != nil {
					return []string{}, fmt.Errorf("unable to resolve release string: %w", err)
				}
				str = append(str, fmt.Sprintf("  %v. %v\n", i-1, rString))
			}
			rString, err := p.getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i].InstanceId)
			if err != nil {
				return []string{}, fmt.Errorf("unable to resolve release string: %w", err)
			}
			str = append(str, fmt.Sprintf("  %v. %v\n", i, rString))
			if i < len(location.GetFoundLocation().GetReleasesLocation())-1 {
				rString, err := p.getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i+1].InstanceId)
				if err != nil {
					return []string{}, fmt.Errorf("unable to resolve release string: %w", err)
				}
				str = append(str, fmt.Sprintf("  %v. %v\n", i+1, rString))
			}
		}
	}
	return str, nil
}

func (p *prodBridge) resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error) {
	f1, err := p.getFolder(ctx, move.FromFolder)
	if err != nil {
		return []string{}, fmt.Errorf("unable to get folder: %w", err)
	}

	f2, err := p.getFolder(ctx, move.ToFolder)
	if err != nil {
		return []string{}, fmt.Errorf("unable to get folder: %w", err)
	}

	loc, err := p.getLocation(ctx, move.Record, f2)
	if err != nil {
		return []string{}, fmt.Errorf("unable to get location: %w", err)
	}

	strret := []string{fmt.Sprintf("%v: %v -> %v\n", move.Record.GetRelease().Title, f1, f2)}
	for _, v := range loc {
		strret = append(strret, v)
	}
	return strret, nil
}

func (p *prodBridge) getMoves(ctx context.Context, id int32) ([]*pbrm.RecordMove, error) {
	conn, err := p.dial(ctx, "recordmover")
	if err != nil {
		return nil, fmt.Errorf("unable to dial mover: %w", err)
	}
	defer conn.Close()

	client := pbrm.NewMoveServiceClient(conn)
	resp, err := client.ListMoves(ctx, &pbrm.ListRequest{InstanceId: id})
	if err != nil {
		return nil, fmt.Errorf("unable to list moves: %w", err)
	}
	return resp.Moves, err
}

func (p *prodBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	conn, err := p.dial(ctx, "recordmover")
	if err != nil {
		return fmt.Errorf("Unable to dial mover %w", err)
	}
	defer conn.Close()

	client := pbrm.NewMoveServiceClient(conn)
	_, err = client.ClearMove(ctx, &pbrm.ClearRequest{InstanceId: move.InstanceId})
	return err
}

func (p *prodBridge) print(ctx context.Context, lines []string, move *pbrm.RecordMove, makeMove bool) error {

	superstring := fmt.Sprintf("From %v\n\n", move)
	for _, line := range lines {
		superstring += line + "\n"
	}

	if !makeMove {
		p.raiseIssue("Would print", superstring)
		return fmt.Errorf("Failing")
	}

	_, err := p.pqc.Print(ctx, &pqcpb.PrintRequest{Lines: lines, Origin: "recordprinter", Destination: pqcpb.Destination_DESTINATION_RECEIPT})
	if err != nil {
		return fmt.Errorf("unable to print: %w", err)
	}
	return err
}

// Server main server type
type Server struct {
	*goserver.GoServer
	bridge    Bridge
	count     int64
	lastCount time.Time
	lastIssue string
	currMove  int32
	pqc       *pqc.PrintQueueClient
}

// Init builds the server
func Init() *Server {
	ctx, cancel := utils.ManualContext("rpinit", time.Minute)
	defer cancel()

	cl, err := pqc.NewPrintQueueClient(ctx)
	if err != nil {
		panic(err)
	}

	s := &Server{
		&goserver.GoServer{},
		&prodBridge{},
		0,
		time.Unix(0, 0),
		"",
		0,
		cl,
	}
	s.bridge = &prodBridge{s.FDialServer, s.RaiseIssue, s.pqc}
	return s
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
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
	var init = flag.Bool("init", false, "Show all output")
	flag.Parse()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	server := Init()
	server.PrepServer("recordprinter")
	server.Register = server
	server.RPCTracing = true

	err := server.RegisterServerV2(false)
	if err != nil {
		return
	}

	if *init {
		ctx, cancel := utils.BuildContext("recordprinter", "recordprinter")
		defer cancel()

		err := server.KSclient.Save(ctx, KEY, &pb.Config{LastPrint: time.Now().Unix()})
		fmt.Printf("Initialised: %v\n", err)
		return
	}

	server.Serve()
}
