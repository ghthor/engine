// generated by stringer -type=Quad -output=quadrant_string.go; DO NOT EDIT

package coord

import "fmt"

const _Quad_name = "NWNESESW"

var _Quad_index = [...]uint8{0, 2, 4, 6, 8}

func (i Quad) String() string {
	if i < 0 || i+1 >= Quad(len(_Quad_index)) {
		return fmt.Sprintf("Quad(%d)", i)
	}
	return _Quad_name[_Quad_index[i]:_Quad_index[i+1]]
}
