package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbg "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
	pbp "github.com/brotherlogic/printer/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
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
	getMoves(ctx context.Context) ([]*pbrm.RecordMove, error)
	clearMove(ctx context.Context, move *pbrm.RecordMove) error
	print(ctx context.Context, lines []string, move *pbrm.RecordMove) error
	resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error)
	getRecord(ctx context.Context, id int32) (*pbrc.Record, error)
}

type prodBridge struct {
	dial       func(server string) (*grpc.ClientConn, error)
	raiseIssue func(ctx context.Context, name string, body string, super bool)
}

func (p *prodBridge) getRecord(ctx context.Context, id int32) (*pbrc.Record, error) {
	conn, err := p.dial("recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	rel, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
	if err != nil {
		return nil, err
	}
	return rel.GetRecord(), err
}

func (p *prodBridge) getFolder(ctx context.Context, folderID int32) (string, error) {
	conn, err := p.dial("recordsorganiser")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := pbro.NewOrganiserServiceClient(conn)
	r, err := client.GetQuota(ctx, &pbro.QuotaRequest{FolderId: folderID})
	if err != nil {
		return "", err
	}

	return r.LocationName, nil
}

func (p *prodBridge) getReleaseString(ctx context.Context, instanceID int32) (string, error) {
	rel, err := p.getRecord(ctx, instanceID)
	if err != nil {
		return "", err
	}
	return rel.GetRelease().Title + " [" + strconv.Itoa(int(instanceID)) + "]", nil
}

func (p *prodBridge) getLocation(ctx context.Context, rec *pbrc.Record, folder string) ([]string, error) {
	conn, err := p.dial("recordsorganiser")
	if err != nil {
		return []string{}, err
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
					return []string{}, err
				}
				str = append(str, fmt.Sprintf("  %v. %v\n", i-1, rString))
			}
			rString, err := p.getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i].InstanceId)
			if err != nil {
				return []string{}, err
			}
			str = append(str, fmt.Sprintf("  %v. %v\n", i, rString))
			if i < len(location.GetFoundLocation().GetReleasesLocation())-1 {
				rString, err := p.getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i+1].InstanceId)
				if err != nil {
					return []string{}, err
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
		return []string{}, err
	}

	f2, err := p.getFolder(ctx, move.ToFolder)
	if err != nil {
		return []string{}, err
	}

	loc, err := p.getLocation(ctx, move.Record, f2)
	if err != nil {
		return []string{}, err
	}

	strret := []string{fmt.Sprintf("%v: %v -> %v\n", move.Record.GetRelease().Title, f1, f2)}
	for _, v := range loc {
		strret = append(strret, v)
	}
	return strret, nil
}

func (p *prodBridge) getMoves(ctx context.Context) ([]*pbrm.RecordMove, error) {
	conn, err := p.dial("recordmover")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pbrm.NewMoveServiceClient(conn)
	resp, err := client.ListMoves(ctx, &pbrm.ListRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Moves, err
}

func (p *prodBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	conn, err := p.dial("recordmover")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pbrm.NewMoveServiceClient(conn)
	_, err = client.ClearMove(ctx, &pbrm.ClearRequest{InstanceId: move.InstanceId})
	return err
}

func (p *prodBridge) print(ctx context.Context, lines []string, move *pbrm.RecordMove) error {

	superstring := fmt.Sprintf("From %v\n\n", move)
	for _, line := range lines {
		superstring += line + "\n"
	}

	p.raiseIssue(ctx, "Would print", superstring, false)
	return fmt.Errorf("Failing")

	conn, err := p.dial("printer")
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pbp.NewPrintServiceClient(conn)
	_, err = client.Print(ctx, &pbp.PrintRequest{Lines: lines, Origin: "recordprinter"})
	return err
}

//Server main server type
type Server struct {
	*goserver.GoServer
	config    *pb.Config
	bridge    Bridge
	count     int64
	lastCount time.Time
	lastIssue string
	currMove  int32
}

// Init builds the server
func Init() *Server {
	s := &Server{
		&goserver.GoServer{},
		&pb.Config{},
		&prodBridge{},
		0,
		time.Unix(0, 0),
		"",
		0,
	}
	s.bridge = &prodBridge{s.DialMaster, s.RaiseIssue}
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

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

func (s *Server) load(ctx context.Context) error {
	config := &pb.Config{}
	data, _, err := s.KSclient.Read(ctx, KEY, config)

	if err != nil {
		return err
	}

	config = data.(*pb.Config)
	return nil
}

func (s *Server) save(ctx context.Context) {
	s.KSclient.Save(ctx, KEY, s.config)
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	if master {
		return s.load(ctx)
	}

	return nil
}

func (s *Server) checkTime(ctx context.Context) error {
	if time.Now().Sub(time.Unix(s.config.LastPrint, 0)) > time.Hour*24 {
		s.RaiseIssue(ctx, "No Prints", fmt.Sprintf("No prints since %v", time.Unix(s.config.LastPrint, 0)), false)
	}
	return nil
}

// GetState gets the state of the server
func (s *Server) GetState() []*pbg.State {
	return []*pbg.State{
		&pbg.State{Key: "last_time", TimeValue: s.config.LastPrint},
		&pbg.State{Key: "curr_count", Value: s.count},
		&pbg.State{Key: "last_count", Text: fmt.Sprintf("%v", s.lastCount)},
		&pbg.State{Key: "error", Text: s.lastIssue},
		&pbg.State{Key: "curr_move", Value: int64(s.currMove)},
	}
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
	server.PrepServer()
	server.Register = server
	server.RPCTracing = true

	err := server.RegisterServer("recordprinter", false)
	if err != nil {
		log.Fatalf("Registration Error: %v", err)
	}

	if *init {
		ctx, cancel := utils.BuildContext("recordprinter", "recordprinter")
		defer cancel()

		err := server.KSclient.Save(ctx, KEY, &pb.Config{LastPrint: time.Now().Unix()})
		fmt.Printf("Initialised: %v\n", err)
		return
	}

	server.RegisterRepeatingTask(server.moveLoop, "move_loop", time.Minute*30)
	server.RegisterRepeatingTask(server.checkTime, "check_time", time.Hour)

	fmt.Printf("%v", server.Serve())
}
