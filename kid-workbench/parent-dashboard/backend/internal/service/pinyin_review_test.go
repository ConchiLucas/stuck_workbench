package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const listenSnapshot = `{"instanceId":"listen-1","type":"listen","targetId":1,"kpId":100,"stem":"听一听，选出你听到的拼音","speechUrl":"/api/v1/pinyin/items/100/speech/solo.mp3","visual":{"kind":"sound"},"options":[{"id":"option-0","label":"b"},{"id":"option-1","label":"p"},{"id":"option-2","label":"m"},{"id":"option-3","label":"f"}]}`

const inwordSnapshot = `{"instanceId":"inword-1","type":"inword","stem":"听一听，这个字里藏着哪个拼音？","speechUrl":"/api/v1/pinyin/items/100/speech/word.mp3","visual":{"kind":"char","text":"播"},"options":[{"id":"option-0","label":"p"},{"id":"option-1","label":"b"},{"id":"option-2","label":"d"},{"id":"option-3","label":"t"}]}`

const shapeSnapshot = `{"instanceId":"shape-1","type":"shape","stem":"看一看，选出四线格里拼音的读音","visual":{"kind":"glyph","text":"ɑ","imageUrl":"/api/v1/pinyin/items/7/glyph.png"},"options":[{"id":"option-0","label":"o","speechUrl":"/api/v1/pinyin/items/8/speech/solo.mp3"},{"id":"option-1","label":"ɑ","speechUrl":"/api/v1/pinyin/items/7/speech/solo.mp3"},{"id":"option-2","label":"e","speechUrl":"/api/v1/pinyin/items/9/speech/solo.mp3"},{"id":"option-3","label":"i","speechUrl":"/api/v1/pinyin/items/10/speech/solo.mp3"}]}`

const blendSnapshot = `{"instanceId":"blend-1","type":"blend","stem":"把声母和韵母拼在一起","visual":{"kind":"blend","initial":"b","final":"ā","syllable":"bā"},"options":[{"id":"option-0","label":"pā","speechUrl":"http://minio.example/ba.mp3"},{"id":"option-1","label":"bā","speechUrl":"/api/v1/pinyin/items/40/speech/solo.mp3"},{"id":"option-2","label":"mā"},{"id":"option-3","label":"fā","speechUrl":"/api/v1/pinyin/items/41/speech/word-1.mp3"}]}`

func TestPinyinReviewListenRewritesLockedSpeech(t *testing.T) {
	r := buildPinyinReview(listenSnapshot, "option-1")
	require.Empty(t, r.UnavailableReason)
	require.Equal(t, "listen", r.Question.Type)
	require.Equal(t, "option-1", r.SelectedOptionID)
	require.Equal(t, "/api/pinyin/items/100/speech/solo.mp3", r.Question.SpeechURL)
	require.Equal(t, "p", r.Question.Options[1].Label)
}

func TestPinyinReviewInwordShapeBlend(t *testing.T) {
	inword := buildPinyinReview(inwordSnapshot, "option-0")
	require.Empty(t, inword.UnavailableReason)
	require.Equal(t, "播", inword.Question.Visual.Text)
	require.Equal(t, "/api/pinyin/items/100/speech/word.mp3", inword.Question.SpeechURL)

	shape := buildPinyinReview(shapeSnapshot, "")
	require.Empty(t, shape.UnavailableReason)
	require.Equal(t, "ɑ", shape.Question.Visual.Text)
	require.Equal(t, "/api/pinyin/items/7/glyph.png", shape.Question.Visual.ImageURL)
	require.Equal(t, "/api/pinyin/items/7/speech/solo.mp3", shape.Question.Options[1].SpeechURL)

	blend := buildPinyinReview(blendSnapshot, "option-1")
	require.Empty(t, blend.UnavailableReason)
	require.Equal(t, "b", blend.Question.Visual.Initial)
	require.Equal(t, "ā", blend.Question.Visual.Final)
	require.Empty(t, blend.Question.Options[0].SpeechURL)
	require.Equal(t, "/api/pinyin/items/40/speech/solo.mp3", blend.Question.Options[1].SpeechURL)
	require.Equal(t, "/api/pinyin/items/41/speech/word-1.mp3", blend.Question.Options[3].SpeechURL)
}

func TestPinyinReviewUnavailable(t *testing.T) {
	for _, raw := range []string{"", "{}", "bad", `{"type":"listen","options":[]}`, strings.Replace(listenSnapshot, `"type":"listen"`, `"type":"other"`, 1), strings.Replace(inwordSnapshot, `"text":"播"`, `"text":""`, 1), strings.Replace(listenSnapshot, `"id":"option-0"`, `"id":"option-1"`, 1)} {
		r := buildPinyinReview(raw, "")
		require.NotEmpty(t, r.UnavailableReason, raw)
		require.Nil(t, r.Question)
	}
	r := buildPinyinReview(listenSnapshot, "missing")
	require.NotEmpty(t, r.UnavailableReason)
	require.Nil(t, r.Question)
}

func TestRewritePinyinMediaRejectsArbitraryURLs(t *testing.T) {
	require.Empty(t, rewritePinyinMedia("http://content.test/api/v1/pinyin/items/1/speech/solo.mp3"))
	require.Empty(t, rewritePinyinMedia("//evil.test/api/v1/pinyin/items/1/speech/solo.mp3"))
	require.Empty(t, rewritePinyinMedia("/api/v1/pinyin/items/1/speech/solo.mp3?url=http://evil.test"))
	require.Empty(t, rewritePinyinMedia("/api/v1/pinyin/items/1/speech/word-9.mp3"))
	require.Equal(t, "/api/pinyin/items/12/speech/word-2.mp3", rewritePinyinMedia("/api/v1/pinyin/items/12/speech/word-2.mp3"))
}

func TestRewritePinyinSyllableMedia(t *testing.T) {
	require.Equal(t, "/api/pinyin/syllables/12/speech.mp3", rewritePinyinMedia("/api/v1/pinyin/syllables/12/speech.mp3"))
	for _, path := range []string{"/api/v1/pinyin/syllables/0/speech.mp3", "/api/v1/pinyin/syllables/12/other.mp3", "/api/v1/pinyin/syllables/12/speech.mp3?url=evil", "https://evil/api/v1/pinyin/syllables/12/speech.mp3"} {
		require.Empty(t, rewritePinyinMedia(path))
	}
}

func TestRewritePinyinSyllableVersion(t *testing.T) {
	require.Equal(t, "/api/pinyin/syllables/12/speech.mp3?v=0123456789abcdef", rewritePinyinMedia("/api/v1/pinyin/syllables/12/speech.mp3?v=0123456789abcdef"))
	for _, query := range []string{"v=bad", "v=0123456789abcdef&url=evil", "v=0123456789ABCDEF", "v=0123456789abcdef&v=0123456789abcdef"} {
		require.Empty(t, rewritePinyinMedia("/api/v1/pinyin/syllables/12/speech.mp3?"+query))
	}
}
