package rpg2d

import (
	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/rpg2d/quad"
	"github.com/ghthor/engine/sim/stime"
)

type World struct {
	time     stime.Time
	quadTree quad.Quad
	terrain  TerrainMap
}

func newWorld(clock stime.Clock, bounds coord.Bounds) (*World, error) {
	quadTree, err := quad.New(bounds, 20, nil)
	if err != nil {
		return nil, err
	}

	terrain, err := NewTerrainMap(bounds, string(TT_GRASS))
	if err != nil {
		return nil, err
	}

	return &World{
		time:     clock.Now(),
		quadTree: quadTree,
		terrain:  terrain,
	}, nil
}

func (w *World) Insert(e entity.Entity) {
	w.quadTree = w.quadTree.Insert(e)
}

func (w *World) Remove(e entity.Entity) {
	w.quadTree = w.quadTree.Remove(e)
}

func (w World) ToState() WorldState {
	entities := w.quadTree.QueryBounds(w.quadTree.Bounds())
	s := WorldState{
		w.time,
		make([]entity.State, len(entities)),
		nil,
		nil,
	}

	i := 0
	for _, e := range entities {
		s.Entities[i] = e.ToState()
		i++
	}

	terrain := w.terrain.ToState()
	if !terrain.IsEmpty() {
		s.TerrainMap = terrain
	}
	return s
}