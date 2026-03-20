package kis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		size     int
		expected int
	}{
		{"빈 슬라이스", nil, 5, 0},
		{"size보다 작음", []string{"a", "b"}, 5, 1},
		{"정확히 나눠짐", []string{"a", "b", "c", "d"}, 2, 2},
		{"나머지 있음", []string{"a", "b", "c"}, 2, 2},
		{"size=1", []string{"a", "b", "c"}, 1, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk(tt.items, tt.size)
			assert.Len(t, result, tt.expected)
		})
	}
}
