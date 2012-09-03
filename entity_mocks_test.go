package engine

import (
	"fmt"
	"github.com/ghthor/gospec/src/gospec"
	. "github.com/ghthor/gospec/src/gospec"
)

type (
	MockEntity struct {
		id    EntityId
		coord WorldCoord
	}

	MockMobileEntity struct {
		id EntityId
		mi *motionInfo
	}

	MockCollision struct {
		time WorldTime
		A, B collidableEntity
	}

	MockCollidableEntity struct {
		id         EntityId
		coord      WorldCoord
		collisions []MockCollision
	}

	MockAliveEntity struct {
		id         EntityId
		mi         *motionInfo
		collisions []MockCollision
	}
)

func (e MockEntity) Id() EntityId      { return e.id }
func (e MockEntity) Coord() WorldCoord { return e.coord }
func (e MockEntity) AABB() AABB        { return AABB{e.coord, e.coord} }
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

func (e *MockMobileEntity) Id() EntityId      { return e.id }
func (e *MockMobileEntity) Coord() WorldCoord { return e.mi.coord }
func (e *MockMobileEntity) AABB() AABB        { return e.mi.AABB() }
func (e *MockMobileEntity) Json() interface{} {
	return struct {
		Id   EntityId `json:"id"`
		Name string   `json:"name"`
	}{
		e.Id(),
		e.String(),
	}
}

func (e *MockMobileEntity) motionInfo() *motionInfo { return e.mi }

func (e *MockMobileEntity) String() string {
	return fmt.Sprintf("MockMobileEntity%v", e.Id())
}

func (e *MockCollidableEntity) Id() EntityId      { return e.id }
func (e *MockCollidableEntity) Coord() WorldCoord { return e.coord }
func (e *MockCollidableEntity) AABB() AABB        { return AABB{e.coord, e.coord} }
func (e *MockCollidableEntity) Json() interface{} {
	return struct {
		Id   EntityId `json:"id"`
		Name string   `json:"name"`
	}{
		e.Id(),
		e.String(),
	}
}

func (e *MockCollidableEntity) collides(other collidableEntity) bool { return true }
func (e *MockCollidableEntity) collideWith(other collidableEntity, t WorldTime) {
	e.collisions = append(e.collisions, MockCollision{t, e, other})
}

func (e *MockCollidableEntity) String() string {
	return fmt.Sprintf("MockCollidableEntity%v", e.Id())
}

func (e *MockAliveEntity) Id() EntityId      { return e.id }
func (e *MockAliveEntity) Coord() WorldCoord { return e.mi.coord }
func (e *MockAliveEntity) AABB() AABB        { return e.mi.AABB() }
func (e *MockAliveEntity) Json() interface{} {
	return struct {
		Id   EntityId `json:"id"`
		Name string   `json:"name"`
	}{
		e.Id(),
		e.String(),
	}
}

func (e *MockAliveEntity) motionInfo() *motionInfo { return e.mi }

func (e *MockAliveEntity) collides(other collidableEntity) bool { return true }
func (e *MockAliveEntity) collideWith(other collidableEntity, t WorldTime) {
	e.collisions = append(e.collisions, MockCollision{t, e, other})
}

func (e *MockAliveEntity) String() string {
	return fmt.Sprintf("MockAliveEntity%v", e.Id())
}

func CollidedWith(a, b interface{}) (collided bool, pos, neg gospec.Message, err error) {
	var collisionsA, collisionsB []MockCollision

	switch e := a.(type) {
	case *MockCollidableEntity:
		collisionsA = e.collisions
	case *MockAliveEntity:
		collisionsA = e.collisions
	}

	switch e := b.(type) {
	case *MockCollidableEntity:
		collisionsB = e.collisions
	case *MockAliveEntity:
		collisionsB = e.collisions
	}

	var time WorldTime
outer:
	for _, collision1 := range collisionsA {
		for _, collision2 := range collisionsB {
			if collision1.time == collision2.time {
				if collision1.A == collision2.B && collision1.B == collision2.A {
					time = collision1.time
					collided = true
					break outer
				}
			}
		}
	}

	pos = gospec.Messagef(a, "had a collision with %v @%d", b, time)
	neg = gospec.Messagef(a, "did not collide with %v", b)
	return
}

func DescribeMockEntities(c gospec.Context) {
	c.Specify("mock entity", func() {
		e := entity(&MockEntity{})

		c.Specify("is an entity", func() {
			_, isAnEntity := e.(entity)
			c.Expect(isAnEntity, IsTrue)
		})

		c.Specify("is not a movable entity", func() {
			_, isAMovableEntity := e.(movableEntity)
			c.Expect(isAMovableEntity, IsFalse)
		})

		c.Specify("is not a collidable entity", func() {
			_, isACollidableEntity := e.(collidableEntity)
			c.Expect(isACollidableEntity, IsFalse)
		})
	})

	c.Specify("mock movable entity", func() {
		e := entity(&MockMobileEntity{})

		c.Specify("is an entity", func() {
			_, isAnEntity := e.(entity)
			c.Expect(isAnEntity, IsTrue)
		})

		c.Specify("is a movable entity", func() {
			_, isAMovableEntity := e.(movableEntity)
			c.Expect(isAMovableEntity, IsTrue)
		})

		c.Specify("is not a collidable entity", func() {
			_, isACollidableEntity := e.(collidableEntity)
			c.Expect(isACollidableEntity, IsFalse)
		})
	})

	c.Specify("mock collidable entity", func() {
		e := entity(&MockCollidableEntity{})

		c.Specify("is an entity", func() {
			_, isAnEntity := e.(entity)
			c.Expect(isAnEntity, IsTrue)
		})

		c.Specify("is not a movable entity", func() {
			_, isAMovableEntity := e.(movableEntity)
			c.Expect(isAMovableEntity, IsFalse)
		})

		c.Specify("is a collidable entity", func() {
			_, isACollidableEntity := e.(collidableEntity)
			c.Expect(isACollidableEntity, IsTrue)
		})
	})

	c.Specify("mock alive entity", func() {
		e := entity(&MockAliveEntity{})

		c.Specify("is an entity", func() {
			_, isAnEntity := e.(entity)
			c.Expect(isAnEntity, IsTrue)
		})

		c.Specify("is a movable entity", func() {
			_, isAMovableEntity := e.(movableEntity)
			c.Expect(isAMovableEntity, IsTrue)
		})

		c.Specify("is a collidable entity", func() {
			_, isACollidableEntity := e.(collidableEntity)
			c.Expect(isACollidableEntity, IsTrue)
		})
	})

	c.Specify("CollidedWith matcher", func() {
		entities := [...]*MockCollidableEntity{
			&MockCollidableEntity{collisions: make([]MockCollision, 0, 2)},
			&MockCollidableEntity{collisions: make([]MockCollision, 0, 2)},
		}

		c.Expect(entities[0], Not(CollidedWith), entities[1])
		c.Expect(entities[1], Not(CollidedWith), entities[0])

		collide(entities[0], entities[1], 0)

		c.Expect(entities[0], CollidedWith, entities[1])
		c.Expect(entities[1], CollidedWith, entities[0])

		collide(entities[0], entities[1], 0)

		c.Expect(entities[0], CollidedWith, entities[1])
		c.Expect(entities[1], CollidedWith, entities[0])
	})
}