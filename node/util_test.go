package node

import (
	"reflect"
	"testing"
)

func TestDiffSliceStr(t *testing.T) {
	type sliceString []string
	slices := [][][]string{
		{
			{"a", "b", "c"},
			{"a", "b"},
		},
		{
			{"a", "b", "c"},
			{"a", "b", "c"},
		},
		{
			{"b", "c"},
			{"a", "b", "c"},
		},
	}
	diffs := [][]string{
		{"c"},
		{},
		{},
	}

	for i, ss := range slices {
		want, got := diffs[i], diffSliceStr(ss[0], ss[1])

		// when we get an empty slice, we are actually getting an uninitialized one
		// for reflect.DeepEqual an initialized and an uninitialized slices are not
		// equal so we must initialize got so it reflect does not bark
		if len(got) == 0 {
			got = make([]string, 0)
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("diffSliceStr(%v, %v): want %v, got %v", ss[0], ss[1], want, got)
		}
	}
}
