package rpg2d

import (
	"github.com/ghthor/filu/rpg2d/entity"
	"github.com/ghthor/filu/rpg2d/quad"
	"github.com/ghthor/filu/sim/stime"
)

type World struct {
	time     stime.Time
	quadTree quad.Quad
	terrain  TerrainMap
}

func NewWorld(now stime.Time, quad quad.Quad, terrain TerrainMap) *World {
	return &World{
		time:     now,
		quadTree: quad,
		terrain:  terrain,
	}
}

type stepToFn func(quad.Quad, stime.Time) quad.Quad

func (w *World) stepTo(t stime.Time, stepTo stepToFn) {
	w.quadTree = stepTo(w.quadTree, t)

	w.time = t
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
		Time:     w.time,
		Bounds:   w.quadTree.Bounds(),
		Entities: make(entity.StateSlice, len(entities)),
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
