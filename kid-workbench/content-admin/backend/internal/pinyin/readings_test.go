package pinyin_test

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/pinyin"
)

func TestEveryReadingHasFiveSimpleExampleWords(t *testing.T) {
	require.Len(t, pinyin.Readings, 45)
	for letter, reading := range pinyin.Readings {
		require.Len(t, reading.Words, 5, letter)
		seen := map[string]bool{}
		for _, word := range reading.Words {
			require.Equal(t, 1, utf8.RuneCountInString(word), "%s example %q should be one character", letter, word)
			require.False(t, seen[word], "%s repeats %q", letter, word)
			seen[word] = true
		}
		require.Equal(t, reading.Words[0], reading.Word())
	}
	require.Equal(t, []string{"包", "八", "不", "白", "北"}, pinyin.Readings["b"].Words)
	require.Equal(t, "灯", pinyin.Readings["eng"].Word())
}
