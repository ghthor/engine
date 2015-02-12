package entity

import "github.com/ghthor/engine/rpg2d/coord"

// A basic entity in the world.
type Entity interface {
	// Unique ID
	Id() int64

	// Location in the world
	Cell() coord.Cell

	// Returns a bounding object incorporating
	// the entities potential interaction with
	// the other entities in the world.
	Bounds() coord.Bounds

	// Returns a state value that represents
	// the entity in its current state.
	ToState() State
}

// Used by the world state to calculate
// differences between world states.
// An object that implements this interface
// should also be friendly to the Json
// marshaller and expect to be sent to the
// client over the wire.
type State interface {
	// Unique ID
	Id() int64

	// Bounds of the entity
	Bounds() coord.Bounds

	// Compare to another entity
	IsDifferentFrom(State) bool
}
