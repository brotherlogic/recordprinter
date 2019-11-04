package main

import (
	"fmt"
	"testing"

	"github.com/brotherlogic/keystore/client"
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
	multiple    bool
	count       int
	flip        bool
}

func (t *testBridge) resolve(ctx context.Context, move *pbrm.RecordMove) ([]string, error) {
	if t.failResolve {
		return []string{}, fmt.Errorf("Built to fail")
	}
	return []string{"hello", "there"}, nil
}

func (t *testBridge) getRecord(ctx context.Context, id int32) (*pbrc.Record, error) {
	t.count--
	if t.count == 0 {
		return nil, fmt.Errorf("Built to fail")
	}
	if !t.flip {
		return &pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN}}, nil
	}
	return &pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL}}, nil
}

func (t *testBridge) getMoves(ctx context.Context) ([]*pbrm.RecordMove, error) {
	if t.failMove {
		return nil, fmt.Errorf("Built to fail")
	}
	if t.poorRecord {
		return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234), BeforeContext: &pbrm.Context{Location: "Before", BeforeInstance: 1}, AfterContext: &pbrm.Context{BeforeInstance: 1, AfterInstance: 1}}}, nil
	}

	if t.poorContext {
		return []*pbrm.RecordMove{&pbrm.RecordMove{InstanceId: int32(1234), Record: &pbrc.Record{Release: &pbgd.Release{InstanceId: 1234}}, BeforeContext: &pbrm.Context{Location: "Before"}, AfterContext: &pbrm.Context{Location: "After"}}}, nil
	}

	if t.multiple {
		return []*pbrm.RecordMove{
			&pbrm.RecordMove{
				InstanceId: int32(1234),
				Record: &pbrc.Record{
					Release: &pbgd.Release{
						InstanceId: 1234,
						Title:      "madeup",
					},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				},
				BeforeContext: &pbrm.Context{
					Location: "Before",
					Before: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "donkey",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
					After: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "donkey",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
				},
				AfterContext: &pbrm.Context{
					Before: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "magic",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
					After: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "magic",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
				},
			},
			&pbrm.RecordMove{
				InstanceId: int32(1234),
				Record: &pbrc.Record{
					Release: &pbgd.Release{
						InstanceId: 1234,
						Title:      "madeup",
					},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				},
				BeforeContext: &pbrm.Context{
					Location: "Before",
					Before: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "donkey",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
					After: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "donkey",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
				},
				AfterContext: &pbrm.Context{
					Before: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "magic",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
					After: &pbrc.Record{
						Release: &pbgd.Release{
							Title: "magic",
						},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
					},
				},
			},
		}, nil
	}

	if t.flip {
		return []*pbrm.RecordMove{
			&pbrm.RecordMove{
				InstanceId: int32(1234),
				Record: &pbrc.Record{
					Release:  &pbgd.Release{InstanceId: 1234, Title: "madeup"},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL},
				},
				BeforeContext: &pbrm.Context{
					Location: "Before",
					Before: &pbrc.Record{
						Release:  &pbgd.Release{Title: "donkey"},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL},
					},
					After: &pbrc.Record{Release: &pbgd.Release{Title: "donkey"},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL},
					},
				},
				AfterContext: &pbrm.Context{
					Before: &pbrc.Record{
						Release:  &pbgd.Release{Title: "magic"},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL},
					},
					After: &pbrc.Record{Release: &pbgd.Release{Title: "magic"},
						Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_LISTED_TO_SELL},
					}}}}, nil
	}

	return []*pbrm.RecordMove{
		&pbrm.RecordMove{
			InstanceId: int32(1234),
			Record: &pbrc.Record{
				Release:  &pbgd.Release{InstanceId: 1234, Title: "madeup"},
				Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
			},
			BeforeContext: &pbrm.Context{
				Location: "Before",
				Before: &pbrc.Record{
					Release:  &pbgd.Release{Title: "donkey"},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				},
				After: &pbrc.Record{Release: &pbgd.Release{Title: "donkey"},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				},
			},
			AfterContext: &pbrm.Context{
				Before: &pbrc.Record{
					Release:  &pbgd.Release{Title: "magic"},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				},
				After: &pbrc.Record{Release: &pbgd.Release{Title: "magic"},
					Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_FRESHMAN},
				}}}}, nil
}
func (t *testBridge) clearMove(ctx context.Context, move *pbrm.RecordMove) error {
	if t.failClear {
		return fmt.Errorf("Built to fail")
	}
	return nil
}

func (t *testBridge) print(ctx context.Context, lines []string, move *pbrm.RecordMove, makeMove bool) error {
	if t.failPrint {
		return fmt.Errorf("Built to fail")
	}
	return nil
}

func InitTestServer() *Server {
	s := Init()
	s.SkipLog = true
	s.GoServer.KSclient = *keystoreclient.GetTestClient(".test")
	s.bridge = &testBridge{}
	return s
}

func TestMove(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{}
	s.moveLoop(context.Background())
}
func TestMoveFlip(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{flip: true}
	s.moveLoop(context.Background())
}

func TestMoveFail1(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{count: 1}
	s.moveLoop(context.Background())
}
func TestMoveFail1Other(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{flip: true, count: 1}
	s.moveLoop(context.Background())
}
func TestMoveFail2(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{count: 2}
	s.moveLoop(context.Background())
}
func TestMoveFail2Other(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{flip: true, count: 2}
	s.moveLoop(context.Background())
}

func TestMoveFail3(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{count: 3}
	s.moveLoop(context.Background())
}
func TestMoveFail3Other(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{flip: true, count: 3}
	s.moveLoop(context.Background())
}

func TestMultiMove(t *testing.T) {
	s := InitTestServer()
	s.bridge = &testBridge{multiple: true}
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

func TestLocationMove(t *testing.T) {
	s := InitTestServer()
	s.move(context.Background(), &pbrm.RecordMove{BeforeContext: &pbrm.Context{Location: "same"}, AfterContext: &pbrm.Context{Location: "same"}})

}
