// +build js,wasm

package rpg2d

import "syscall/js"

func (s WorldState) JSValue() js.Value {
	v := js.Global().Get("Object").New()
	v.Set("Time", int64(s.Time))
	v.Set("Bounds", s.Bounds.JSValue())
	v.Set("EntitiesRemoved", s.EntitiesRemoved.JSValue())
	v.Set("EntitiesNew", s.EntitiesNew.JSValue())
	v.Set("EntitiesChanged", s.EntitiesChanged.JSValue())
	v.Set("EntitiesUnchanged", s.EntitiesUnchanged.JSValue())
	v.Set("TerrainMap", s.TerrainMap.JSValue())
	return v
}

func (s WorldStateDiff) JSValue() js.Value {
	v := js.Global().Get("Object").New()
	v.Set("Time", int64(s.Time))
	v.Set("Bounds", s.Bounds.JSValue())

	v.Set("Entities", s.Entities.JSValue())
	v.Set("Removed", s.Removed.JSValue())

	if s.TerrainMapSlices == nil || len(s.TerrainMapSlices.Slices) <= 0 {
		v.Set("TerrainMapSlices", js.Null())
	} else {
		a := js.Global().Get("Array").New(len(s.TerrainMapSlices.Slices))
		for i, slice := range s.TerrainMapSlices.Slices {
			a.SetIndex(i, slice)
		}
		vv := js.Global().Get("Object").New()
		vv.Set("Bounds", s.TerrainMapSlices.Bounds)
		vv.Set("Slices", a)
		v.Set("TerrainMapSlices", vv)
	}

	return v
}