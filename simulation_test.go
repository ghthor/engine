package engine

import (
	"encoding/json"
	"fmt"
	"github.com/ghthor/gospec/src/gospec"
	. "github.com/ghthor/gospec/src/gospec"
	"strconv"
)

type noopConn int

func (c noopConn) SendJson(msg string, obj interface{}) error {
	c++
	return nil
}

var conn noopConn

type spyConn struct {
	packets chan string
}

func (c spyConn) SendJson(msg string, obj interface{}) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	c.packets <- msg + ":" + string(jsonBytes)
	return nil
}

func DescribeSimulation(c gospec.Context) {
	sim := newSimulation(40)

	c.Specify("starting and stopping", func() {

		sim.Start()
		c.Expect(sim.IsRunning(), IsTrue)

		sim.Stop()
		c.Expect(sim.IsRunning(), IsFalse)
	})

	c.Specify("clock ticks during each step", func() {
		c.Assume(sim.clock, Equals, Clock(0))

		sim.step()

		c.Expect(sim.clock, Equals, Clock(1))
	})

	c.Specify("Adding and removing players", func() {
		c.Assume(sim.nextEntityId, Equals, EntityId(0))

		// Need a Client endpoint
		var conn noopConn

		pd := PlayerDef{
			Name:          "thundercleese",
			Facing:        North,
			Coord:         WorldCoord{0, 0},
			MovementSpeed: 40,
			Conn:          conn,
		}

		c.Specify("adding", func() {
			player := sim.addPlayer(pd)

			c.Expect(player.Id(), Equals, EntityId(0))
			c.Expect(len(sim.state.entities), Equals, 1)
			c.Expect(len(sim.state.movableEntities), Equals, 1)
			c.Expect(len(sim.clients), Equals, 1)
		})

		c.Specify("adding while the simulation is running", func() {
			sim.Start()

			player := sim.AddPlayer(pd)

			sim.Stop()

			c.Expect(player.Id(), Equals, EntityId(0))
			c.Expect(len(sim.state.entities), Equals, 1)
			c.Expect(len(sim.state.movableEntities), Equals, 1)
			c.Expect(len(sim.clients), Equals, 1)
		})

		c.Specify("removing", func() {
			player := sim.addPlayer(pd)

			c.Assume(player.Id(), Equals, EntityId(0))
			c.Assume(len(sim.state.entities), Equals, 1)
			c.Assume(len(sim.state.movableEntities), Equals, 1)
			c.Assume(len(sim.clients), Equals, 1)

			sim.removePlayer(player)

			c.Expect(len(sim.state.entities), Equals, 0)
			c.Expect(len(sim.state.movableEntities), Equals, 0)
			c.Expect(len(sim.clients), Equals, 0)
		})

		c.Specify("removing while the simulation is running", func() {
			player := sim.addPlayer(pd)

			c.Assume(player.Id(), Equals, EntityId(0))
			c.Assume(len(sim.state.entities), Equals, 1)
			c.Assume(len(sim.state.movableEntities), Equals, 1)
			c.Assume(len(sim.clients), Equals, 1)

			sim.Start()

			sim.RemovePlayer(player)

			sim.Stop()

			c.Expect(len(sim.state.entities), Equals, 0)
			c.Expect(len(sim.state.movableEntities), Equals, 0)
			c.Expect(len(sim.clients), Equals, 0)
		})

		c.Specify("player removes self while simulation is running", func() {
			player := sim.addPlayer(pd)

			c.Assume(player.Id(), Equals, EntityId(0))
			c.Assume(len(sim.state.entities), Equals, 1)
			c.Assume(len(sim.state.movableEntities), Equals, 1)
			c.Assume(len(sim.clients), Equals, 1)

			sim.Start()

			player.Disconnect()

			sim.Stop()

			c.Expect(len(sim.state.entities), Equals, 0)
			c.Expect(len(sim.state.movableEntities), Equals, 0)
			c.Expect(len(sim.clients), Equals, 0)
		})

		// TODO drain the newPlayer/dcedPlayer channels after the loop has broken
		c.Specify("when the simulation is stopping shouldn't block", nil)
	})

	c.Specify("simulation loop runs at the intended fps", nil)
}

type MockEntity int

func (e MockEntity) Id() EntityId { return EntityId(e) }
func (e MockEntity) Json() interface{} {
	return struct {
		Id   EntityId `json:"id"`
		Name string   `json:"name"`
	}{
		e.Id(),
		e.String(),
	}
}
func (e MockEntity) String() string {
	return fmt.Sprintf("MockEntity%v", e.Id())
}

func DescribeWorldState(c gospec.Context) {
	c.Specify("processes movement requests and generates PathActions", func() {
		playerA := &Player{
			Name:     "thundercleese",
			entityId: 0,
			mi:       newMotionInfo(WorldCoord{0, 0}, North, 35),
			conn:     conn,
		}
		playerB := &Player{
			Name:     "zorak",
			entityId: 1,
			mi:       newMotionInfo(WorldCoord{1, 0}, South, 40),
			conn:     conn,
		}

		playerA.mux()
		playerB.mux()

		worldState := newWorldState(Clock(0))
		worldState.AddMovableEntity(playerA)
		worldState.AddMovableEntity(playerB)

		c.Assume(len(worldState.entities), Equals, 2)
		c.Assume(len(worldState.movableEntities), Equals, 2)

		c.Specify("consume moveRequest's and produce PathActions", func() {
			playerA.SubmitInput("move=0", "north")
			playerB.SubmitInput("move=0", "south")

			worldState.stepTo(WorldTime(1))

			c.Expect(playerA.mi.moveRequest, IsNil)
			c.Expect(playerA.mi.facing, Equals, North)
			c.Expect(len(playerA.mi.pathActions), Equals, 1)

			pathActionA := playerA.mi.pathActions[0]
			c.Expect(pathActionA.Orig, Equals, WorldCoord{0, 0})
			c.Expect(pathActionA.Dest, Equals, WorldCoord{0, 0}.Neighbor(North))
			c.Expect(pathActionA.duration, Equals, int64(35))

			c.Expect(playerB.mi.moveRequest, IsNil)
			c.Expect(playerB.mi.facing, Equals, South)
			c.Expect(len(playerB.mi.pathActions), Equals, 1)

			pathActionB := playerB.mi.pathActions[0]
			c.Expect(pathActionB.Orig, Equals, WorldCoord{1, 0})
			c.Expect(pathActionB.Dest, Equals, WorldCoord{1, 0}.Neighbor(South))
			c.Expect(pathActionB.duration, Equals, int64(40))

			c.Specify("and the pathActions are removed when they have completed", func() {
				aEnd := worldState.time + WorldTime(playerA.mi.speed)
				bEnd := worldState.time + WorldTime(playerB.mi.speed)

				for i := worldState.time + 1; i < aEnd; i++ {
					worldState.stepTo(i)
					c.Expect(len(playerA.mi.pathActions), Equals, 1)
					c.Expect(len(playerB.mi.pathActions), Equals, 1)
				}
				worldState.step()

				c.Expect(len(playerA.mi.pathActions), Equals, 0)
				c.Expect(len(playerB.mi.pathActions), Equals, 1)

				for i := worldState.time + 1; i < bEnd; i++ {
					worldState.stepTo(i)
					c.Expect(len(playerA.mi.pathActions), Equals, 0)
					c.Expect(len(playerB.mi.pathActions), Equals, 1)
				}
				worldState.step()

				c.Expect(len(playerA.mi.pathActions), Equals, 0)
				c.Expect(len(playerB.mi.pathActions), Equals, 0)
			})
		})

		c.Specify("consume moveRequest's and changes entities facing", func() {
			step := func() {
				worldState.step()
				json := worldState.Json()
				playerA.SendWorldState(json)
				playerB.SendWorldState(json)
			}

			playerA.SubmitInput("move=0", "west")
			playerB.SubmitInput("move=0", "west")

			step()

			c.Expect(playerA.mi.moveRequest, IsNil)
			c.Expect(playerA.mi.facing, Equals, West)
			c.Expect(len(playerA.mi.pathActions), Equals, 0)

			c.Expect(playerB.mi.moveRequest, IsNil)
			c.Expect(playerB.mi.facing, Equals, West)
			c.Expect(len(playerB.mi.pathActions), Equals, 0)

			playerA.SubmitInput("move=0", "north")
			playerB.SubmitInput("move=0", "south")

			step()

			c.Expect(playerA.mi.moveRequest, IsNil)
			c.Expect(playerA.mi.facing, Equals, North)
			c.Expect(len(playerA.mi.pathActions), Equals, 0)

			c.Expect(playerB.mi.moveRequest, IsNil)
			c.Expect(playerB.mi.facing, Equals, South)
			c.Expect(len(playerB.mi.pathActions), Equals, 0)
		})
	})

	c.Specify("generates json compatitable state object", func() {
		worldState := newWorldState(Clock(0))

		entity := MockEntity(0)
		worldState.entities[entity.Id()] = entity

		jsonState := worldState.Json()

		c.Assume(jsonState.Time, Equals, WorldTime(0))
		c.Assume(len(jsonState.Entities), Equals, 1)

		jsonBytes, err := json.Marshal(jsonState)
		c.Expect(err, IsNil)
		c.Expect(string(jsonBytes), Equals, `{"time":0,"entities":[{"id":0,"name":"MockEntity0"}]}`)
	})
}

func DescribePlayer(c gospec.Context) {
	conn := spyConn{make(chan string)}

	player := &Player{
		Name:     "thundercleese",
		entityId: 0,
		mi:       newMotionInfo(WorldCoord{0, 0}, North, 40),
		conn:     conn,
	}

	player.mux()
	defer player.stopMux()

	c.Specify("motionInfo becomes locked when accessed by the simulation until the worldstate is published", func() {
		_ = player.motionInfo()

		locked := make(chan bool)

		go func() {
			select {
			case player.collectInput <- InputCmd{0, "move", "north"}:
				panic("MotionInfo not locked")
			case <-conn.packets:
				locked <- true
			}
		}()

		player.SendWorldState(newWorldState(Clock(0)).Json())
		c.Expect(<-locked, IsTrue)

		c.Specify("and is unlocked afterwards", func() {
			select {
			case player.collectInput <- InputCmd{0, "move", "north"}:
			default:
				panic("MotionInfo not unlocked")
			}
		})
	})

	c.Specify("a request to move is generated when the user inputs a move cmd", func() {
		player.SubmitInput("move=0", "north")

		moveRequest := player.motionInfo().moveRequest

		c.Expect(moveRequest, Not(IsNil))
	})

	c.Specify("a requst to move is canceled by a moveCancel cmd", func() {
		player.SubmitInput("move=0", "north")
		player.SubmitInput("moveCancel=0", "north")

		c.Expect(player.motionInfo().moveRequest, IsNil)
	})

	c.Specify("a moveCancel cmd is dropped if it doesn't cancel the current move request", func() {
		player.SubmitInput("move=0", "north")
		player.SubmitInput("moveCancel=0", "south")

		c.Expect(player.motionInfo().moveRequest, Not(IsNil))
	})

	c.Specify("generates json compatitable state object", func() {
		jsonBytes, err := json.Marshal(player.Json())
		c.Expect(err, IsNil)
		c.Expect(string(jsonBytes), Equals, `{"id":0,"name":"thundercleese","facing":"north","pathActions":null,"coord":{"x":0,"y":0}}`)

		player.mi.pathActions = append(player.mi.pathActions, &PathAction{
			NewTimeSpan(0, 10),
			WorldCoord{0, 0},
			WorldCoord{0, 1},
		})

		jsonBytes, err = json.Marshal(player.Json())
		c.Expect(err, IsNil)
		c.Expect(string(jsonBytes), Equals, `{"id":0,"name":"thundercleese","facing":"north","pathActions":[{"start":0,"end":10,"orig":{"x":0,"y":0},"dest":{"x":0,"y":1}}],"coord":{"x":0,"y":0}}`)
	})
}

func DescribeInputCommands(c gospec.Context) {
	c.Specify("creating movement requests from InputCmds", func() {
		c.Specify("north", func() {
			moveRequest := newMoveRequest(InputCmd{
				0,
				"move",
				"north",
			})

			c.Expect(moveRequest.t, Equals, WorldTime(0))
			c.Expect(moveRequest.Direction, Equals, North)
		})

		c.Specify("east", func() {
			moveRequest := newMoveRequest(InputCmd{
				0,
				"move",
				"east",
			})

			c.Expect(moveRequest.t, Equals, WorldTime(0))
			c.Expect(moveRequest.Direction, Equals, East)
		})

		c.Specify("south", func() {
			moveRequest := newMoveRequest(InputCmd{
				0,
				"move",
				"south",
			})

			c.Expect(moveRequest.t, Equals, WorldTime(0))
			c.Expect(moveRequest.Direction, Equals, South)
		})

		c.Specify("west", func() {
			moveRequest := newMoveRequest(InputCmd{
				0,
				"move",
				"west",
			})

			c.Expect(moveRequest.t, Equals, WorldTime(0))
			c.Expect(moveRequest.Direction, Equals, West)
		})
	})

	c.Specify("embedding worldtime in the cmd msg", func() {
		player := &Player{
			Name:         "thundercleese",
			entityId:     0,
			mi:           newMotionInfo(WorldCoord{0, 0}, North, 40),
			conn:         conn,
			collectInput: make(chan InputCmd, 1),
		}

		c.Specify("string splits on = and parses 64bit int", func() {
			player.SubmitInput("move=0", "north")
			input := <-player.collectInput

			c.Expect(input.timeIssued, Equals, WorldTime(0))

			player.SubmitInput("move=1824081", "north")
			input = <-player.collectInput

			c.Expect(input.timeIssued, Equals, WorldTime(1824081))

			player.SubmitInput("move=99", "north")
			input = <-player.collectInput

			c.Expect(input.timeIssued, Equals, WorldTime(99))
		})

		c.Specify("errors with invalid input and doesn't publish the command", func() {
			err := player.SubmitInput("move=a", "north")
			e := err.(*strconv.NumError)

			c.Expect(e, Not(IsNil))
			c.Expect(e.Err, Equals, strconv.ErrSyntax)

			err = player.SubmitInput("move=", "north")
			e = err.(*strconv.NumError)

			c.Expect(e, Not(IsNil))
			c.Expect(e.Err, Equals, strconv.ErrSyntax)
		})
	})
}
