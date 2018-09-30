package main

import (
	"fmt"
	"testing"

	"golang.org/x/net/context"

	pbrm "github.com/brotherlogic/recordmover/proto"
)

type testBridge struct {
	failMove  bool
	failPrint bool
	failClear bool
}

func (t *testBridge) getMoves(ctx context.Context) ([]*pbrm.RecordMove, error) {
	if t.failMove {
		return nil, fmt.Errorf("Built to fail")
	}
	return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234)}}, nil
}
func (t *testBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	if t.failClear {
		return fmt.Errorf("Built to fail")
	}
	return nil
}

func (t *testBridge) print(ctx context.Context, text string) error {
	if t.failPrint {
		return fmt.Errorf("Built to fail")
	}
	return nil
}

func InitTestServer() *Server {
	s := Init()
	s.SkipLog = true
	return s
}

func TestMove(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{}
	s.moveLoop(context.Background())
}

func TestMoveFail(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{failMove: true}
	s.moveLoop(context.Background())
}

func TestPrintFail(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{failPrint: true}
	s.moveLoop(context.Background())

}

func TestClearFail(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{failClear: true}
	s.moveLoop(context.Background())
}
