package engine

import (
	"../server/protocol"
	"strconv"
	"strings"
)

type (
	EntityId int64

	entity interface {
		Id() EntityId
		Coord() WorldCoord
		AABB() AABB
		Json() interface{}
	}

	moveRequest struct {
		t WorldTime
		Direction
	}

	motionInfo struct {
		coord  WorldCoord
		facing Direction
		speed  uint

		moveRequest *moveRequest

		// fifo
		pathActions []*PathAction

		lastMoveAction MoveAction
		UndoLastApply  func()
	}

	movableEntity interface {
		entity
		motionInfo() *motionInfo
	}

	collidableEntity interface {
		entity
		collides(collidableEntity) bool
		collideWith(collidableEntity, WorldTime)
	}
)

func newMotionInfo(coord WorldCoord, facing Direction, speed uint) *motionInfo {
	return &motionInfo{
		coord,
		facing,
		speed,
		nil,
		make([]*PathAction, 0, 2),
		nil,
		nil,
	}
}

func (mi motionInfo) isMoving() bool {
	return len(mi.pathActions) != 0
}

func (mi motionInfo) AABB() (aabb AABB) {
	if mi.isMoving() {
		pa := mi.pathActions[0]
		switch pa.Direction() {
		case West:
			fallthrough
		case North:
			aabb.TopL = pa.Dest
			aabb.BotR = pa.Orig
		case East:
			fallthrough
		case South:
			aabb.TopL = pa.Orig
			aabb.BotR = pa.Dest
		}
	} else {
		aabb = AABB{mi.coord, mi.coord}
	}
	return
}

func (mi *motionInfo) Apply(moveAction MoveAction) {
	switch action := moveAction.(type) {
	case TurnAction:
		mi.UndoLastApply = nil
		mi.facing = action.to
		mi.lastMoveAction = action

	case *PathAction:
		prevFacing := mi.facing
		prevMoveRequest := mi.moveRequest
		mi.UndoLastApply = func() {
			mi.UndoLastApply = nil
			mi.facing = prevFacing
			mi.pathActions = mi.pathActions[:0]
			mi.moveRequest = prevMoveRequest
		}

		mi.facing = action.Direction()
		mi.pathActions = append(mi.pathActions, action)

	default:
		panic("unknown MoveRequest type")
	}

	mi.moveRequest = nil
}

func collide(ce1, ce2 collidableEntity, t WorldTime) {
	ce1.collideWith(ce2, t)
	ce2.collideWith(ce1, t)
}

type InputCmd struct {
	timeIssued WorldTime
	cmd        string
	params     string
}

func newMoveRequest(input InputCmd) *moveRequest {
	switch input.params {
	case "north":
		return &moveRequest{
			t:         input.timeIssued,
			Direction: North,
		}
	case "east":
		return &moveRequest{
			t:         input.timeIssued,
			Direction: East,
		}
	case "south":
		return &moveRequest{
			t:         input.timeIssued,
			Direction: South,
		}
	case "west":
		return &moveRequest{
			t:         input.timeIssued,
			Direction: West,
		}

	}
	panic("never reached")
}

// Externaly Accessible Actions
type PlayerEntity interface {
	entity
	SubmitInput(cmd, params string) error
	Disconnect()
}

// This object is used to create a player
// All the Fields must be provided
type PlayerDef struct {

	// The Players Name
	Name string

	// Where the Player is locationed
	Coord WorldCoord
	// Which Direction the Player is Facing
	Facing Direction
	// Movement speed in Frames per PathAction
	MovementSpeed uint

	// A Connection to send WorldState too
	Conn protocol.JsonOutputConn

	// This is Used internally by the simulation to return the new
	// Player object after is has been created
	newPlayer chan *Player
}

type Player struct {
	// The Players Name
	Name     string
	entityId EntityId
	mi       *motionInfo

	// A handle to the simulation this player is in
	sim Simulation

	// A Connection to send WorldState too
	conn protocol.JsonOutputConn

	// Communication channels used inside the muxer
	collectInput    chan InputCmd
	serveMotionInfo chan *motionInfo
	routeWorldState chan WorldStateJson
	killMux         chan bool
}

type PlayerJson struct {
	Id          EntityId         `json:"id"`
	Name        string           `json:"name"`
	Facing      string           `json:"facing"`
	PathActions []PathActionJson `json:"pathActions"`
	Coord       WorldCoord       `json:"coord"`
}

func (p *Player) Id() EntityId {
	return p.entityId
}

func (p *Player) Coord() WorldCoord {
	return p.mi.coord
}

func (p *Player) AABB() (aabb AABB) {
	return p.mi.AABB()
}

func (p *Player) Json() interface{} {
	ps := PlayerJson{
		Id:     p.entityId,
		Name:   p.Name,
		Facing: p.mi.facing.String(),
		Coord:  p.mi.coord,
	}

	if len(p.mi.pathActions) > 0 {
		ps.PathActions = make([]PathActionJson, len(p.mi.pathActions))
		for i, pa := range p.mi.pathActions {
			ps.PathActions[i] = pa.Json()
		}
	}
	return ps
}

func (p *Player) mux() {
	p.collectInput = make(chan InputCmd)
	p.serveMotionInfo = make(chan *motionInfo)
	p.routeWorldState = make(chan WorldStateJson)
	p.killMux = make(chan bool)

	go func() {
		for {
			// Prioritize stopping the mux loop
			select {
			case <-p.killMux:
				return
			default:
			}

			select {
			case input := <-p.collectInput:
				switch input.cmd {
				case "move":
					p.mi.moveRequest = newMoveRequest(input)
				case "moveCancel":
					if p.mi.moveRequest != nil {
						if p.mi.moveRequest.Direction.String() == input.params {
							p.mi.moveRequest = nil
						}
					}
				default:
					panic("Unknown InputCmd: " + input.cmd)
				}

			// The simulation has requested access to the motionInfo
			// This 'locks' the motionInfo until the server publishs a WorldState
			case p.serveMotionInfo <- p.mi:
			lockedMotionInfo:
				for {
					select {
					//case input := <-p.collectInput:
					// Buffer all input to be processed after WorldState is published
					case p.serveMotionInfo <- p.mi:
					case worldState := <-p.routeWorldState:
						// Take the worldState and cut out anything the client shouldn't know
						// Package up this localized WorldState and send it over the wire
						p.conn.SendJson("update", worldState)
						break lockedMotionInfo
					case <-p.killMux:
						return
					}
				}
			case <-p.killMux:
				return
			}
		}
	}()
}

func (p *Player) stopMux() {
	p.killMux <- true
}

// External interface of the muxer presented to the simulation
func (p *Player) motionInfo() *motionInfo             { return <-p.serveMotionInfo }
func (p *Player) SendWorldState(state WorldStateJson) { p.routeWorldState <- state }

// External interface of the muxer presented to the Node
func (p *Player) SubmitInput(cmd, params string) error {
	parts := strings.Split(cmd, "=")

	timeIssued, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	p.collectInput <- InputCmd{
		timeIssued: WorldTime(timeIssued),
		cmd:        parts[0],
		params:     params,
	}
	return nil
}

func (p *Player) Disconnect() {
	p.sim.RemovePlayer(p)
}
