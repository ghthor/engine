package engine

import (
	"fmt"
	"sync"
	"time"
)

/*
The Client generates InputEvents and sends them to the server.  The Node processes these input events and packages them up for the simulation. When the simulation runs it gathers all the Clients InputEvent's and Currently executing Actions and steps simulates a step forward in time generating the next WorldState.  This WorldState is then published to all the clients via a channel
*/
type (
	WorldState struct {
		processingTime  time.Duration
		time            WorldTime
		entities        map[EntityId]entity
		movableEntities map[EntityId]movableEntity
	}

	Simulation interface {
		Start()
		IsRunning() bool
		Stop()

		AddPlayer(PlayerDef) *Player
		AddClient(stateConn)
	}
)

type (
	stateConn interface {
		SendWorldState(*WorldState)
	}

	simulation struct {
		clock        Clock
		nextEntityId EntityId
		state        *WorldState

		clients []stateConn

		// Comm channels for adding/removing Players
		newPlayer  chan PlayerDef
		dcedPlayer chan dcedPlayer

		fps int

		stop    chan bool
		running bool
		sync.Mutex
	}
)

func (ws *WorldState) String() string {
	return fmt.Sprintf(":%v\tEntities:%v", ws.time, len(ws.entities))
}

func (ws *WorldState) AddMovableEntity(e movableEntity) {
	ws.entities[e.Id()] = e
	ws.movableEntities[e.Id()] = e
}

func NewSimulation(fps int) Simulation {
	return newSimulation(fps)
}

func newWorldState(clock Clock) *WorldState {
	return &WorldState{
		time:            clock.Now(),
		entities:        make(map[EntityId]entity),
		movableEntities: make(map[EntityId]movableEntity),
	}
}

func newSimulation(fps int) *simulation {
	clk := Clock(0)

	s := &simulation{
		clock: clk,

		state: &WorldState{
			time:            clk.Now(),
			entities:        make(map[EntityId]entity, 10),
			movableEntities: make(map[EntityId]movableEntity, 10),
		},

		clients: make([]stateConn, 0, 10),

		newPlayer:  make(chan PlayerDef),
		dcedPlayer: make(chan dcedPlayer),

		fps: fps,

		stop: make(chan bool),
	}
	return s
}

func (s *simulation) Start() {
	s.startLoop()
}

func (s *simulation) startLoop() {

	ticker := time.NewTicker(time.Duration(1000/s.fps) * time.Millisecond)

	s.Lock()

	go func() {
		s.running = true
		s.Unlock()

	stepLoop:
		for {
			s.step()
			select {
			case <-s.stop:
				break stepLoop
			default:
			}

			select {
			case <-ticker.C:
			case playerDef := <-s.newPlayer:
				playerDef.newPlayer <- s.addPlayer(playerDef)
			case dced := <-s.dcedPlayer:
				dced.removed <- s.removePlayer(dced.player)
			case <-s.stop:
				break stepLoop

			}
		}

		s.Lock()
		s.running = false
		s.Unlock()
		s.stop <- true
	}()
}

func (s *simulation) step() {
	s.clock = s.clock.Tick()
	s.state = s.state.stepTo(s.clock.Now())
	for _, c := range s.clients {
		c.SendWorldState(s.state)
	}
}

func (s *simulation) Stop() {
	s.stop <- true
	<-s.stop
}

func (s *simulation) IsRunning() bool {
	s.Lock()
	defer s.Unlock()
	return s.running
}

func (s *simulation) AddPlayer(pd PlayerDef) *Player {
	pd.newPlayer = make(chan *Player)
	s.newPlayer <- pd
	return <-pd.newPlayer
}

func (s *simulation) addPlayer(pd PlayerDef) *Player {
	p := &Player{
		Name: pd.Name,

		entityId: s.nextEntityId,
		mi:       newMotionInfo(pd.Coord, pd.Facing, pd.MovementSpeed),
		sim:      s,
		conn:     pd.Conn,
	}
	s.nextEntityId++

	// Add the Player entity
	s.state.entities[p.entityId] = p
	s.state.movableEntities[p.entityId] = p

	// Add the Player as a client that recieves WorldState's
	s.clients = append(s.clients, p)

	// Start the muxer
	p.mux()
	return p
}

type dcedPlayer struct {
	player  *Player
	removed chan *Player
}

func (s *simulation) RemovePlayer(p *Player) {
	dced := dcedPlayer{
		p,
		make(chan *Player),
	}
	s.dcedPlayer <- dced
	<-dced.removed
	return
}

func (s *simulation) removePlayer(p *Player) *Player {
	// Stop the muxer
	p.stopMux()

	// Remove the Player entity
	delete(s.state.entities, p.Id())
	delete(s.state.movableEntities, p.Id())

	// Remove the client
	for i, c := range s.clients {
		if c == p {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			break
		}
	}
	return p
}

func (s *simulation) AddClient(c stateConn) {
}

func (s *WorldState) step() *WorldState {
	return s.stepTo(s.time + 1)
}

func (s *WorldState) stepTo(t WorldTime) *WorldState {

	// TODO this is going to cause awful allocations
	attemptingMove := make([]movableEntity, 0, 20)
	for _, entity := range s.movableEntities {
		mi := entity.motionInfo()

		// Removed finished pathActions
		for _, pa := range mi.pathActions {
			if pa.end <= t {
				mi.pathActions = mi.pathActions[:0]
				mi.coord = pa.Dest
			}
		}

		// Select entities with pending moveRequests
		if mi.moveRequest != nil && len(mi.pathActions) == 0 {
			attemptingMove = append(attemptingMove, entity)
		}
	}
	// TODO Filter out already moving

	// TODO Filter out Actions that are impossible - Walking into a wall

	// Filter Conflicting Actions - 2+ Characters moving in to the same Coord
	// TODO this is going to cause awful allocations
	newPaths := make(map[WorldCoord]movableEntity, len(attemptingMove))
	for _, entity := range attemptingMove {
		mi := entity.motionInfo()
		dest := mi.coord.Neighbor(mi.moveRequest.Direction)

		if newPaths[dest] == nil {
			newPaths[dest] = entity
		} else {
			// TODO Resolve Conflicting Actions based on speed
			newPaths[dest] = entity
		}
	}

	// Apply All remaining Pending Actions as Running Actions
	for dest, entity := range newPaths {
		mi := entity.motionInfo()
		pathAction := &PathAction{
			NewTimeSpan(t, t+WorldTime(mi.speed)),
			mi.coord,
			dest,
		}

		// Update MoveRequest
		mi.moveRequest = nil
		mi.facing = pathAction.Direction()
		mi.pathActions = append(mi.pathActions, pathAction)
	}
	// Write out new state and return
	s.time = t
	return s
}
