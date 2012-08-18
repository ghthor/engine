package engine

import (
	"../server/protocol"
	"fmt"
)

type (
	EntityId int64

	entity interface {
		Id() EntityId
	}

	moveRequest struct {
		t WorldTime
		Direction
	}

	motionInfo struct {
		coord  WorldCoord
		facing Direction

		moveRequest *moveRequest

		// fifo
		pathActions []*PathAction
	}

	movableEntity interface {
		entity
		motionInfo() *motionInfo
	}
)

func newMotionInfo(c WorldCoord, f Direction) *motionInfo {
	return &motionInfo{
		c,
		f,
		nil,
		make([]*PathAction, 0, 2),
	}
}

func (mi motionInfo) isMoving() bool {
	return len(mi.pathActions) == 0
}

type UserInput struct {
	Type    string
	Payload interface{}
}

type PlayerDef struct {
	Name   string
	Facing Direction
	Coord  WorldCoord
	Conn   protocol.Conn

	// This is Used internally by the simulation to return the new
	// Player object after is has been created
	newPlayer chan *Player
}

type Player struct {
	entityId EntityId
	Name     string
	mi       *motionInfo

	// Communication channels used inside the muxer
	collectInput    chan UserInput
	serveMotionInfo chan *motionInfo
	routeWorldState chan *WorldState

	// A handle to the simulation this player is in
	sim Simulation

	conn protocol.Conn
}

func (p *Player) Id() EntityId {
	return p.entityId
}

func (p *Player) mux() {
	p.collectInput = make(chan UserInput)
	p.serveMotionInfo = make(chan *motionInfo)
	p.routeWorldState = make(chan *WorldState)

	go func() {
		for {
			select {
			case input := <-p.collectInput:
				// Process Input
				// Create ActionReq's, like a MoveRequest
				fmt.Println(input)

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
						p.conn.SendMessage("worldState", worldState.String())
						break lockedMotionInfo
					}
				}
			}
		}
	}()
}

// External interface of the muxer presented to the simulation
func (p *Player) motionInfo() *motionInfo          { return <-p.serveMotionInfo }
func (p *Player) SendWorldState(state *WorldState) { p.routeWorldState <- state }

// External interface of the muxer presented to the Node
func (p *Player) SubmitInput(ui UserInput) { p.collectInput <- ui }
