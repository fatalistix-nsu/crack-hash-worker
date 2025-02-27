package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPermutationGenerator_Next(t *testing.T) {
	tests := []struct {
		alphabet string
		current  []uint64
		expected string
	}{
		{"ab", []uint64{0}, "b"},
		{"ab", []uint64{1}, "aa"},
		{"ab", []uint64{0, 1}, "ba"},
		{"ab", []uint64{1, 1}, "aaa"},
	}

	for _, test := range tests {
		g := PermutationGenerator{
			alphabet: []rune(test.alphabet),
			current:  test.current,
			id:       0,
			size:     2,
		}

		assert.True(t, g.HasNext())
		_ = g.Next()

		assert.True(t, g.HasNext())
		next := g.Next()

		assert.Equal(t, test.expected, next)
	}
}
