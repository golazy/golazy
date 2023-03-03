package component

type ImportMap map[string]string

// Merge merges the given ImportMap into the current one.
func (m ImportMap) Merge(other ImportMap) {
	for k, v := range other {
		m[k] = v
	}
}

// MergeCopy merges the given ImportMap into a new one and returns it.
func (m ImportMap) MergeCopy(other ImportMap) ImportMap {
	n := make(ImportMap)
	n.Merge(m)
	n.Merge(other)
	return n
}
