package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/math-server/internal/catalog"
)

func TestArithmeticStage(t *testing.T) {
	tests := []struct {
		module string
		a, b   int
		want   string
	}{
		{"add10", 2, 3, "within5"}, {"add10", 4, 6, "within10"},
		{"add10", 8, 7, "within20"}, {"sub10", 5, 2, "within5"},
		{"sub10", 10, 4, "within10"}, {"sub10", 20, 8, "within20"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, catalog.StageFor(tc.module, tc.a, tc.b))
	}
	require.Equal(t, "basic-shapes", catalog.StageFor("shape", 0, 0))
}

func TestStageRejectsOutOfRangeArithmetic(t *testing.T) {
	require.Empty(t, catalog.StageFor("add10", 20, 1))
	require.Empty(t, catalog.StageFor("sub10", 21, 1))
	require.Empty(t, catalog.StageFor("unknown", 1, 1))
}
