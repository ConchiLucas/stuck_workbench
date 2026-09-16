package literacy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"os"
	"testing"
)

func templateJSON(char string) []byte {
	b, _ := json.Marshal(map[string]any{"schemaVersion": 1, "character": char, "coordinateSystem": map[string]any{"width": 1024, "height": 1024, "yAxis": "up", "baseline": 900}, "strokes": []string{"M 100 500 L 900 500 Z"}, "medians": [][][]float64{{{100, 500}, {900, 500}}}, "source": map[string]string{"name": "test fixture", "url": "https://example.org/fixture"}, "license": "CC0-1.0"})
	return b
}
func TestWritingImportFreezeWithoutSenseAndPreserveRevision(t *testing.T) {
	ctx := context.Background()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, g.Create(&literacy.Asset{KpID: 1, CharText: "的", SpeechAudioURL: "speech"}).Error)
	store := &memStore{objects: map[string][]byte{}}
	store.PutBytes(ctx, store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	svc := literacy.NewService(g, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(ctx, "")
	require.NoError(t, err)
	require.False(t, list.Items[0].Capabilities["write_char"].Ready)
	_, err = svc.ImportWritingTemplate(ctx, 1, templateJSON("山"))
	require.ErrorIs(t, err, literacy.ErrInvalidWritingTemplate)
	imported, err := svc.ImportWritingTemplate(ctx, 1, templateJSON("的"))
	require.NoError(t, err)
	require.NotEmpty(t, imported.Version)
	list, err = svc.GenerationMaterials(ctx, "")
	require.NoError(t, err)
	require.True(t, list.Items[0].Capabilities["write_char"].Ready)
	require.False(t, list.Items[0].Capabilities["glyph_sense"].Ready)
	req := []literacy.FreezeItem{{KpID: 1, SourceRevision: list.Items[0].SourceRevision, QuestionTypes: []string{"write_char"}}}
	frozen, err := svc.FreezeMaterials(ctx, req)
	require.NoError(t, err)
	require.Nil(t, frozen.Items[0].Sense)
	require.NotNil(t, frozen.Items[0].WritingTemplate)
	old, ct, err := svc.RevisionMedia(ctx, frozen.Items[0].RevisionID, "writing_template")
	require.NoError(t, err)
	require.Equal(t, "application/json", ct)
	_, err = svc.ImportWritingTemplate(ctx, 1, []byte(`{}`))
	require.Error(t, err)
	again, err := svc.FreezeMaterials(ctx, req)
	require.NoError(t, err)
	require.Equal(t, frozen, again)
	var changed map[string]any
	require.NoError(t, json.Unmarshal(templateJSON("的"), &changed))
	changed["license"] = "CC0-1.0 updated attribution"
	b, _ := json.Marshal(changed)
	_, err = svc.ImportWritingTemplate(ctx, 1, b)
	require.NoError(t, err)
	_, err = svc.FreezeMaterials(ctx, req)
	require.ErrorIs(t, err, literacy.ErrSourceChanged)
	got, _, err := svc.RevisionMedia(ctx, frozen.Items[0].RevisionID, "writing_template")
	require.NoError(t, err)
	require.Equal(t, old, got)
	require.NoError(t, g.Model(&literacy.Asset{}).Where("kp_id = 1").Update("speech_audio_url", "").Error)
	list, err = svc.GenerationMaterials(ctx, "")
	require.NoError(t, err)
	require.False(t, list.Items[0].Capabilities["write_char"].Ready)
}

func TestWritingRealOfflineFixturesAndCorruptionReadiness(t *testing.T) {
	ctx := context.Background()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	svc := literacy.NewService(g, nil, nil, nil, nil, nil, nil)
	for i, char := range []string{"山", "水", "一", "的", "二", "三", "十"} {
		id := int64(i + 1)
		require.NoError(t, g.Create(&literacy.Asset{KpID: id, CharText: char, SpeechAudioURL: "speech"}).Error)
		b, err := os.ReadFile("testdata/writing-templates/" + char + ".json")
		require.NoError(t, err)
		detail, err := svc.ImportWritingTemplate(ctx, id, b)
		require.NoError(t, err)
		require.NotEmpty(t, detail.Version)
	}
	require.NoError(t, g.Table("literacy_writing_templates").Where("kp_id = 1").Update("content", "{}").Error)
	list, err := svc.GenerationMaterials(ctx, "")
	require.NoError(t, err)
	require.False(t, list.Items[0].Capabilities["write_char"].Ready)
}
func TestWritingRejectsMalformedPathAndPreservesCurrent(t *testing.T) {
	ctx := context.Background()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, g.Create(&literacy.Asset{KpID: 1, CharText: "一"}).Error)
	svc := literacy.NewService(g, nil, nil, nil, nil, nil, nil)
	for _, path := range []string{"M 100 500 L Z", "M 100 500 L 99999999 0 Z", "M 100 500 Q 0 Z"} {
		var input map[string]any
		require.NoError(t, json.Unmarshal(templateJSON("一"), &input))
		input["strokes"] = []string{path}
		b, _ := json.Marshal(input)
		_, err := svc.ImportWritingTemplate(ctx, 1, b)
		require.ErrorIs(t, err, literacy.ErrInvalidWritingTemplate, path)
	}
}
func TestExplicitChoiceRequiresGlyphButLegacyStillFreezes(t *testing.T) {
	ctx := context.Background()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	require.NoError(t, g.Create(&literacy.Asset{KpID: 1, CharText: "一", SpeechAudioURL: "speech", SenseImageURL: "sense"}).Error)
	store := &memStore{objects: map[string][]byte{}}
	store.PutBytes(ctx, store.SpeechKey(1), fixtureMP3(t), "audio/mpeg")
	var img bytes.Buffer
	require.NoError(t, png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	store.PutPNG(ctx, store.SenseKey(1), img.Bytes())
	svc := literacy.NewService(g, store, nil, nil, nil, nil, nil)
	list, err := svc.GenerationMaterials(ctx, "")
	require.NoError(t, err)
	item := literacy.FreezeItem{KpID: 1, SourceRevision: list.Items[0].SourceRevision}
	_, err = svc.FreezeMaterials(ctx, []literacy.FreezeItem{item})
	require.NoError(t, err)
	item.QuestionTypes = []string{"glyph_sense"}
	_, err = svc.FreezeMaterials(ctx, []literacy.FreezeItem{item})
	require.ErrorIs(t, err, literacy.ErrMaterialNotReady)
}
func TestIdenticalTemplateCanBelongToTwoKnowledgePoints(t *testing.T) {
	ctx := context.Background()
	g, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(g))
	svc := literacy.NewService(g, nil, nil, nil, nil, nil, nil)
	for _, id := range []int64{1, 2} {
		require.NoError(t, g.Create(&literacy.Asset{KpID: id, CharText: "一"}).Error)
		_, err := svc.ImportWritingTemplate(ctx, id, templateJSON("一"))
		require.NoError(t, err)
		_, err = svc.WritingTemplate(ctx, id, "")
		require.NoError(t, err)
	}
}
