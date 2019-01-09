package main

import (
	"fmt"
	"testing"

	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbrm "github.com/brotherlogic/recordmover/proto"
)

type testBridge struct {
	failMove    bool
	failPrint   bool
	failClear   bool
	failResolve bool
	poorRecord  bool
	poorContext bool
}

func (t *testBridge) resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error) {
	if t.failResolve {
		return []string{}, fmt.Errorf("Built to fail")
	}
	return []string{"hello", "there"}, nil
}

func (t *testBridge) getMoves(ctx context.Context) ([]*pbrm.RecordMove, error) {
	if t.failMove {
		return nil, fmt.Errorf("Built to fail")
	}
	if t.poorRecord {
		return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234), BeforeContext: &pbrm.Context{Location: "Before", Before: &pbrc.Record{Release: &pbgd.Release{Title: "donkey"}}}, AfterContext: &pbrm.Context{Before: &pbrc.Record{Release: &pbgd.Release{Title: "magic"}}, After: &pbrc.Record{Release: &pbgd.Release{Title: "magic"}}}}}, nil
	}

	if t.poorContext {
		return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234), Record: &pbrc.Record{Release: &pbgd.Release{InstanceId: 1234}}, BeforeContext: &pbrm.Context{Location: "Before"}, AfterContext: &pbrm.Context{Location: "After"}}}, nil
	}

	return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234), Record: &pbrc.Record{Release: &pbgd.Release{InstanceId: 1234, Title: "madeup"}}, BeforeContext: &pbrm.Context{Location: "Before", Before: &pbrc.Record{Release: &pbgd.Release{Title: "donkey"}}}, AfterContext: &pbrm.Context{Before: &pbrc.Record{Release: &pbgd.Release{Title: "magic"}}, After: &pbrc.Record{Release: &pbgd.Release{Title: "magic"}}}}}, nil
}
func (t *testBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	if t.failClear {
		return fmt.Errorf("Built to fail")
	}
	return nil
}

func (t *testBridge) print(ctx context.Context, lines []string) error {
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

func TestMovePoor(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{poorRecord: true}
	s.moveLoop(context.Background())
}

func TestMovePoorContext(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{poorContext: true}
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

func TestResolveFail(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{failResolve: true}
	s.moveLoop(context.Background())

}

func TestClearFail(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{failClear: true}
	s.moveLoop(context.Background())
}