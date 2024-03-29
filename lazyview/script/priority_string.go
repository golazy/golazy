// Code generated by "stringer -type=Priority"; DO NOT EDIT.

package script

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Auto-0]
	_ = x[High-1]
	_ = x[Low-2]
}

const _Priority_name = "AutoHighLow"

var _Priority_index = [...]uint8{0, 4, 8, 11}

func (i Priority) String() string {
	if i < 0 || i >= Priority(len(_Priority_index)-1) {
		return "Priority(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Priority_name[_Priority_index[i]:_Priority_index[i+1]]
}
