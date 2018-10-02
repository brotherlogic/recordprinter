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
	pbrm "github.com/brotherlogic/recordmover/proto"
)

// Bridge link to other services
type Bridge interface {
	getMoves(ctx context.Context) ([]*pbrm.RecordMove, error)
	clearMove(ctx context.Context, move *pbrm.RecordMove) error
	print(ctx context.Context, text string) error
}

type prodBridge struct{}

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
		return nil, err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	client := pbrm.NewMoveServiceClient(conn)
	_, err = client.ClearMove(ctx, &pbrm.ClearRequest{InstanceId: move.InstanceId})
	return err
}

func (p *prodBridge) print(ctx context.Context, text string) error {
	host, port, err := utils.Resolve("printer")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	client := pbp.NewPrintServiceClient(conn)
	_, err = client.Print(ctx, &pbp.PrintRequest{Text: text})
	return err
}

//Server main server type
type Server struct {
	*goserver.GoServer
	bridge Bridge
	count  int64
}

// Init builds the server
func Init() *Server {
	s := &Server{
		&goserver.GoServer{},
		&prodBridge{},
		0,
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
