// Package nodes provides data structures to represents Html ElementNodes,
// TextNodes and Attributes.
package nodes

import (
	"sort"

	"golang.org/x/exp/constraints"
)

func Each[T any, J any](items []T, fn func(T) J) []J {

	data := make([]J, len(items))
	for i, item := range items {
		data[i] = fn(item)
	}

	return data
}

func sortSlice[T constraints.Ordered](s []T) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}

func EachMapKey[T constraints.Ordered, J any, K any](items map[T]J, fn func(T) K) []K {
	// Get the list of keys
	i := 0
	keys := make([]T, len(items))
	for k := range items {
		keys[i] = k
		i++
	}
	// Sort it
	sortSlice(keys)

	// Gen the output
	data := make([]K, len(items))
	for i, k := range keys {
		data[i] = fn(k)
	}

	return data

}
func EachMap[T constraints.Ordered, J any, K any](items map[T]J, fn func(T, J) K) []K {

	// Get the list of keys
	i := 0
	keys := make([]T, len(items))
	for k := range items {
		keys[i] = k
		i++
	}
	// Sort it
	sortSlice(keys)

	// Gen the output
	data := make([]K, len(items))
	for i, k := range keys {
		item := items[k]
		data[i] = fn(k, item)
	}

	return data
}
