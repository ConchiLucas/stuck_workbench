package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGardenHasSevenPavilionsInPathOrder(t *testing.T) {
	rows := Pavilions()
	require.Equal(t, []string{"moon", "dawn", "goose", "farm", "tower", "snow", "scroll"}, pavilionCodes(rows))
	require.Equal(t, "望月", rows[0].KidTitle)
	require.Equal(t, "闻鸟", rows[1].KidTitle)
	require.Equal(t, "诵句", rows[2].KidTitle)
	require.Equal(t, "补字", rows[3].KidTitle)
	require.Equal(t, "对句", rows[4].KidTitle)
	require.Equal(t, "寻影", rows[5].KidTitle)
	require.Equal(t, "读画", rows[6].KidTitle)
}

func TestEachPavilionHasObjectiveClues(t *testing.T) {
	for _, pavilion := range Pavilions() {
		clues := Clues(pavilion.Code)
		require.GreaterOrEqual(t, len(clues), 3, pavilion.Code)
		require.LessOrEqual(t, len(clues), 4, pavilion.Code)
		require.Equal(t, len(clues), pavilion.ClueCount)
		require.NotEqual(t, "title", pavilion.Kind)
		require.NotEqual(t, "nextline", pavilion.Kind)
		require.Equal(t, pavilion.Kind, clues[0].Kind)
		for _, clue := range clues {
			require.NotEmpty(t, clue.Kind)
			require.NotEmpty(t, clue.Prompt)
			require.NotEmpty(t, clue.AnswerID)
			require.Len(t, clue.Options, 4, clue.ID)
			require.NotContains(t, clue.Prompt, "Pack your bag")
			if clue.Kind == "recite" {
				require.Equal(t, len(clue.Sequence), len(clue.Options), clue.ID)
				continue
			}
			matches := 0
			for _, option := range clue.Options {
				if option.ID == clue.AnswerID {
					matches++
				}
			}
			require.Equal(t, 1, matches, clue.ID)
		}
	}
}

func TestMoonClueAsksForMoonlightNotAWordCard(t *testing.T) {
	clues := Clues("moon")
	require.Equal(t, "床前明月光", clues[0].Line)
	require.Equal(t, "moon", clues[0].AnswerID)
	require.Equal(t, "choice", clues[0].Kind)
	require.Equal(t, "title", clues[2].Kind)
	require.Equal(t, "jingyesi", clues[2].AnswerID)
	require.Equal(t, "author", clues[3].Kind)
	require.Equal(t, "libai", clues[3].AnswerID)
}

func TestGooseClueIsRecitingYongEInOrder(t *testing.T) {
	clues := Clues("goose")
	require.Equal(t, "recite", clues[0].Kind)
	require.Equal(t, []string{"鹅鹅鹅", "曲项向天歌", "白毛浮绿水", "红掌拨清波"}, clues[0].Sequence)
}

func pavilionCodes(rows []Pavilion) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Code
	}
	return out
}
