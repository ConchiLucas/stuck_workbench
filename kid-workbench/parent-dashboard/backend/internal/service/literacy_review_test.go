package service

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

const reviewSnapshot = `{"schemaVersion":2,"subjectCode":"literacy","questionType":"glyph_sense","interaction":"choice","stem":{"text":"山","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"glyph","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},"options":[{"id":"a","text":"山","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"b","text":"水","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"c","text":"火","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},{"id":"d","text":"木","image":{"revisionId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","kind":"sense","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}],"answerOptionId":"a"}`

func TestLiteracyReviewFrozenShuffle(t *testing.T) {
	for _, selected := range []string{"a", "b", ""} {
		r := buildLiteracyReview(99, reviewSnapshot, "1,0,3,2", selected)
		require.Empty(t, r.UnavailableReason)
		require.Equal(t, "b", r.Question.Options[0].ID)
		require.Equal(t, selected, r.SelectedOptionID)
		require.Equal(t, "a", r.AnswerOptionID)
		require.Equal(t, "/api/literacy/material-revisions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/media/sense", r.Question.Options[1].Image)
	}
}
func TestLiteracyReviewUnavailable(t *testing.T) {
	for _, x := range []struct {
		version              int64
		raw, order, selected string
	}{{0, reviewSnapshot, "", ""}, {99, "{}", "", ""}, {99, "bad", "", ""}, {99, reviewSnapshot, "0,0,2,3", ""}, {99, reviewSnapshot, "0,x,1,2,3", ""}, {99, reviewSnapshot, "0,1,2,3", "missing"}} {
		r := buildLiteracyReview(x.version, x.raw, x.order, x.selected)
		require.NotEmpty(t, r.UnavailableReason)
		require.Nil(t, r.Question)
	}
}

func TestLiteracyReviewRejectsMissingFrozenMedia(t *testing.T) {
	for _, raw := range []string{
		strings.Replace(reviewSnapshot, `"kind":"sense"`, `"kind":"unknown"`, 1),
		strings.Replace(reviewSnapshot, strings.Repeat("b", 64), "", 1),
	} {
		r := buildLiteracyReview(99, raw, "", "")
		require.NotEmpty(t, r.UnavailableReason)
		require.Nil(t, r.Question)
	}
}

func TestLiteracyReviewRequiresGlyphAndFourSenseImages(t *testing.T) {
	for _, kind := range []string{"missing_stem", "wrong_stem", "missing_option", "wrong_option"} {
		t.Run(kind, func(t *testing.T) {
			var s map[string]any
			require.NoError(t, json.Unmarshal([]byte(reviewSnapshot), &s))
			stem := s["stem"].(map[string]any)
			option := s["options"].([]any)[2].(map[string]any)
			switch kind {
			case "missing_stem":
				delete(stem, "image")
			case "wrong_stem":
				stem["image"].(map[string]any)["kind"] = "sense"
			case "missing_option":
				delete(option, "image")
			case "wrong_option":
				option["image"].(map[string]any)["kind"] = "glyph"
			}
			raw, e := json.Marshal(s)
			require.NoError(t, e)
			r := buildLiteracyReview(99, string(raw), "", "")
			require.NotEmpty(t, r.UnavailableReason)
			require.Nil(t, r.Question)
		})
	}
}
