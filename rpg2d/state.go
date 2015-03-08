package rpg2d

import (
	"encoding/json"

	"github.com/ghthor/engine/rpg2d/coord"
	"github.com/ghthor/engine/rpg2d/entity"
	"github.com/ghthor/engine/sim/stime"
)

// Used to calculate diff's
type TerrainMapState struct {
	TerrainMap
}

func (m TerrainMapState) MarshalJSON() ([]byte, error) {
	return json.Marshal(TerrainMapStateSlice{
		Bounds:  m.TerrainMap.Bounds,
		Terrain: m.TerrainMap.String(),
	})
}

type TerrainMapStateSlice struct {
	Bounds  coord.Bounds `json:"bounds"`
	Terrain string       `json:"terrain"`
}

type TerrainMapStateDiff struct {
	Bounds  coord.Bounds        `json:"bounds"`
	Changes []TerrainTypeChange `json:"changes"`
}

func (m *TerrainMapState) IsEmpty() bool {
	if m == nil {
		return true
	}
	return m.TerrainMap.TerrainTypes == nil
}

func (m *TerrainMapState) Diff(other *TerrainMapState) []*TerrainMapState {
	if m.IsEmpty() || !m.Bounds.Overlaps(other.Bounds) {
		return []*TerrainMapState{other}
	}

	mBounds, oBounds := m.TerrainMap.Bounds, other.TerrainMap.Bounds
	rects := mBounds.DiffFrom(oBounds)

	// mBounds == oBounds
	if len(rects) == 0 {
		// TODO Still need to calc changes to map types in cells
		return nil
	}

	slices := make([]*TerrainMapState, 0, len(rects))
	for _, r := range rects {
		slices = append(slices, &TerrainMapState{other.Slice(r)})
	}

	return slices
}

func (m *TerrainMapState) Clone() (*TerrainMapState, error) {
	if m == nil {
		return m, nil
	}

	tm, err := m.TerrainMap.Clone()
	if err != nil {
		return nil, err
	}

	return &TerrainMapState{TerrainMap: tm}, nil
}

type WorldState struct {
	Time   stime.Time   `json:"time"`
	Bounds coord.Bounds `json:"bounds"`

	Entities []entity.State `json:"entities"`

	TerrainMap *TerrainMapState `json:"terrainMap,omitempty"`
}

type WorldStateDiff struct {
	Time   stime.Time   `json:"time"`
	Bounds coord.Bounds `json:"bounds"`

	Entities []entity.State `json:"entities"`
	Removed  []entity.State `json:"removed"`

	TerrainMapSlices []*TerrainMapState `json:"terrainMapSlices,omitempty"`
}

func (s WorldState) Clone() WorldState {
	terrainMap, err := s.TerrainMap.Clone()
	if err != nil {
		panic("error cloning terrain map: " + err.Error())
	}
	clone := WorldState{
		Time:       s.Time,
		Entities:   make([]entity.State, len(s.Entities)),
		TerrainMap: terrainMap,
	}
	copy(clone.Entities, s.Entities)
	return clone
}

// Returns a world state that only contains
// entities and terrain within bounds.
// Does NOT change world state type.
func (s WorldState) Cull(bounds coord.Bounds) (culled WorldState) {
	culled.Time = s.Time

	// Cull Entities
	for _, e := range s.Entities {
		if bounds.Overlaps(e.Bounds()) {
			culled.Entities = append(culled.Entities, e)
		}
	}

	// Cull Terrain
	// TODO Maybe remove the ability to have an empty TerrainMap
	// Requires updating some tests to have a terrain map that don't have one
	if !s.TerrainMap.IsEmpty() {
		culled.TerrainMap = &TerrainMapState{TerrainMap: s.TerrainMap.Slice(bounds)}
	}
	return
}

// Returns a world state that only contains
// entities and terrain that is different such
// that state + diff == other. Diff is therefor
// the changes necessary to get from state to other.
func (state WorldState) Diff(other WorldState) (diff WorldStateDiff) {
	diff.Time = other.Time

	if len(state.Entities) == 0 && len(other.Entities) > 0 {
		diff.Entities = other.Entities
	} else {
		// Find the entities that have changed from the old state to the new one
	nextEntity:
		for _, entity := range other.Entities {
			for _, old := range state.Entities {
				if entity.Id() == old.Id() {
					if old.IsDifferentFrom(entity) {
						diff.Entities = append(diff.Entities, entity)
					}
					continue nextEntity
				}
			}
			// This is a new Entity
			diff.Entities = append(diff.Entities, entity)
		}

		// Check if all the entities in old state exist in the new state
	entityStillExists:
		for _, old := range state.Entities {
			for _, entity := range other.Entities {
				if old.Id() == entity.Id() {
					continue entityStillExists
				}
			}
			diff.Removed = append(diff.Removed, old)
		}
	}

	// Diff the TerrainMap
	diff.TerrainMapSlices = state.TerrainMap.Diff(other.TerrainMap)
	return
}
