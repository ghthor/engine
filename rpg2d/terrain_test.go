package rpg2d

import (
	"github.com/ghthor/engine/rpg2d/coord"

	"github.com/ghthor/gospec"
	. "github.com/ghthor/gospec"
)

func DescribeTerrainMap(c gospec.Context) {
	C := func(x, y int) coord.Cell { return coord.Cell{x, y} }

	c.Specify("a terrain map", func() {
		terrainMap, err := NewTerrainMap(coord.Bounds{
			C(-2, 3),
			C(2, -2),
		}, "G")
		c.Expect(err, IsNil)
		c.Expect(len(terrainMap.TerrainTypes), Equals, 6)
		c.Expect(len(terrainMap.TerrainTypes[0]), Equals, 5)

		for _, row := range terrainMap.TerrainTypes {
			for x, _ := range row {
				c.Expect(row[x], Equals, TT_GRASS)
			}
		}
		// TODO I wonder if I can check the memory allocations during this spec
		// to see if my byte slice is large enough
		c.Specify("can be converted to a string", func() {
			c.Expect(terrainMap.String(), Equals, `
GGGGG
GGGGG
GGGGG
GGGGG
GGGGG
GGGGG
`)

			terrainMap.TerrainTypes[2][3] = TT_DIRT
			terrainMap.TerrainTypes[3][2] = TT_ROCK
			c.Expect(terrainMap.String(), Equals, `
GGGGG
GGGGG
GGGDG
GGRGG
GGGGG
GGGGG
`)
		})

		c.Specify("can be created with a string", func() {
			terrainMap, err = NewTerrainMap(coord.Bounds{C(0, 0), C(5, -6)}, `
RRRRRD
RRRRRD
RRRRRD
RRRRRD
RRRRRD
RRRRRD
DDDDDD
`)
			c.Expect(err, IsNil)

			c.Expect(terrainMap.String(), Equals, `
RRRRRD
RRRRRD
RRRRRD
RRRRRD
RRRRRD
RRRRRD
DDDDDD
`)
		})

		c.Specify("can be accessed", func() {
			terrainMap.TerrainTypes[0][0] = TT_ROCK

			c.Specify("directly", func() {
				c.Expect(terrainMap.TerrainTypes[0][0], Equals, TT_ROCK)
				c.Expect(terrainMap.TerrainTypes[1][1], Equals, TT_GRASS)
			})
			c.Specify("by cell", func() {
				c.Expect(terrainMap.Cell(C(-2, 3)), Equals, TT_ROCK)
				c.Expect(terrainMap.Cell(C(-1, 2)), Equals, TT_GRASS)
			})
		})

		c.Specify("can be sliced into a smaller rectangle", func() {
			terrainMap.TerrainTypes[1][2] = TT_DIRT
			terrainMap.TerrainTypes[4][3] = TT_ROCK

			slice := terrainMap.Slice(coord.Bounds{
				C(-1, 2),
				C(1, -1),
			})
			c.Expect(slice.String(), Equals, `
GDG
GGG
GGG
GGR
`)

			c.Specify("that can be sliced again", func() {
				slice = slice.Slice(coord.Bounds{
					C(-1, 2),
					C(0, 2),
				})
				c.Expect(slice.String(), Equals, "\nGD\n")
			})

			c.Specify("that shares memory with the original slice", func() {
				c.Assume(terrainMap.TerrainTypes[1][1], Equals, TT_GRASS)
				slice.TerrainTypes[0][0] = TT_ROCK
				c.Expect(terrainMap.TerrainTypes[1][1], Equals, TT_ROCK)
			})
		})

		c.Specify("can be sliced by an overlapping rectangle", func() {
			slice := terrainMap.Slice(coord.Bounds{
				C(-5, 2),
				C(2, -1),
			})
			c.Expect(slice.Bounds, Equals, coord.Bounds{
				C(-2, 2),
				C(2, -1),
			})
		})

		c.Specify("cannot be sliced by a non overlapping rectangle", func() {
			defer func() {
				e := recover()
				c.Expect(e, Equals, "invalid terrain map slicing operation: no overlap")
			}()

			terrainMap.Slice(coord.Bounds{
				C(-3000, -3000),
				C(-3000, -3001),
			})
		})
	})

	c.Specify("a terrain map state", func() {
		fullMap, err := NewTerrainMap(coord.Bounds{
			C(0, 0),
			C(3, -3),
		}, `
GRGG
DDDD
DRRR
DGGR
`)
		c.Assume(err, IsNil)

		c.Specify("can calculate there are no differences", func() {
			terrainState := fullMap.ToState()

			diff := terrainState.Diff(terrainState)
			c.Expect(diff.IsEmpty(), IsTrue)
		})

		c.Specify("will, when unintialized calculate a full diff with a non empty map", func() {
			old := &TerrainMapState{}
			terrainState := fullMap.ToState()
			diff := old.Diff(terrainState)

			c.Expect(diff.IsEmpty(), IsFalse)
			c.Expect(diff.TerrainMap.Bounds, Equals, terrainState.TerrainMap.Bounds)
			c.Expect(len(diff.TerrainMap.TerrainTypes), Equals, len(terrainState.TerrainMap.TerrainTypes))
		})

		c.Specify("can calculate row or col differences with another TerrainMap", func() {
			terrainMap := fullMap.Slice(coord.Bounds{
				C(1, -1),
				C(2, -2),
			})
			oldTerrain := terrainMap.ToState()

			c.Specify("if the width is the same and the left and right edges are the same", func() {
				c.Specify("and it overlaps the top", func() {
					terrainMap = fullMap.Slice(coord.Bounds{
						C(1, 0),
						C(2, -1),
					})
					c.Assume(terrainMap.Bounds.Overlaps(*oldTerrain.Bounds), IsTrue)

					newTerrain := terrainMap.ToState()
					diff := oldTerrain.Diff(newTerrain)

					c.Expect(*diff.Bounds, Equals, coord.Bounds{
						C(1, 0),
						C(2, 0),
					})
					c.Expect(diff.Terrain, Equals, "\nRG\n")
				})

				c.Specify("and it overlaps the bottom", func() {
					terrainMap = fullMap.Slice(coord.Bounds{
						C(1, -2),
						C(2, -3),
					})
					c.Assume(terrainMap.Bounds.Overlaps(*oldTerrain.Bounds), IsTrue)

					newTerrain := terrainMap.ToState()
					diff := oldTerrain.Diff(newTerrain)

					c.Expect(*diff.Bounds, Equals, coord.Bounds{
						C(1, -3),
						C(2, -3),
					})
					c.Expect(diff.Terrain, Equals, "\nGG\n")
				})
			})

			c.Specify("if the height is the same and the top and bottom edges are the same", func() {
				c.Specify("and it overlaps the left", func() {
					terrainMap = fullMap.Slice(coord.Bounds{
						C(0, -1),
						C(1, -2),
					})
					c.Assume(terrainMap.Bounds.Overlaps(*oldTerrain.Bounds), IsTrue)

					newTerrain := terrainMap.ToState()
					diff := oldTerrain.Diff(newTerrain)

					c.Expect(*diff.Bounds, Equals, coord.Bounds{
						C(0, -1),
						C(0, -2),
					})
					c.Expect(diff.Terrain, Equals, "\nD\nD\n")
				})

				c.Specify("and it overlaps the right", func() {
					terrainMap = fullMap.Slice(coord.Bounds{
						C(2, -1),
						C(3, -2),
					})
					c.Assume(terrainMap.Bounds.Overlaps(*oldTerrain.Bounds), IsTrue)

					newTerrain := terrainMap.ToState()
					diff := oldTerrain.Diff(newTerrain)

					c.Expect(*diff.Bounds, Equals, coord.Bounds{
						C(3, -1),
						C(3, -2),
					})
					c.Expect(diff.Terrain, Equals, "\nD\nR\n")
				})
			})
		})
	})
}
