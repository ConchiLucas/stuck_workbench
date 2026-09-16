package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChengyuReviewRewritesFrozenMedia(t *testing.T) {
	raw := `{"schema":1,"kind":"meaning","skillCode":"meaning","example":{"kind":"meaning","speechUrl":"/api/v1/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3","options":[{"id":"label:a","label":"a"},{"id":"label:b","label":"b"}],"answerId":"label:a"}}`
	r := buildChengyuReview(raw, "label:b")
	require.NotNil(t, r.Example)
	require.Equal(t, "/api/chengyu/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.mp3", r.Example.SpeechURL)
}

func TestBuildChengyuReviewStripsLiveSpeechURL(t *testing.T) {
	raw := `{"schema":1,"kind":"meaning","skillCode":"meaning","example":{"kind":"meaning","speechUrl":"/api/v1/chengyu/items/21/speech.mp3","options":[{"id":"label:a","label":"a"},{"id":"label:b","label":"b"}],"answerId":"label:a"}}`
	r := buildChengyuReview(raw, "label:a")
	require.Equal(t, "", r.Example.SpeechURL)
	require.Contains(t, r.UnavailableReason, "未冻结")
}

func TestBuildChengyuReviewIgnoresAccumulatedPicks(t *testing.T) {
	raw := `{"schema":1,"kind":"pick","skillCode":"pick","example":{"kind":"pick","prompt":"专心","options":[{"id":"1","label":"一心一意"},{"id":"2","label":"三心二意"}],"answerId":"1"}}`
	r := buildChengyuReview(raw, `[{"clientId":"a","optionIndex":1}]`)
	require.Equal(t, "", r.Selected)
}

func TestBuildChengyuReviewMissingSnapshot(t *testing.T) {
	r := buildChengyuReview("", "")
	require.Contains(t, r.UnavailableReason, "未保存")
}

func TestBuildChengyuReviewRejectsCodeTypeOnlySnapshot(t *testing.T) {
	r := buildChengyuReview(`{"code":"meaning","type":"choice"}`, "2")
	require.Contains(t, r.UnavailableReason, "无法还原")
}
