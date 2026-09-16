package mathtask

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/conchi/study-learning/mathcontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/stretchr/testify/require"
)

func fixture() mathcontent.MathDetail {
	return mathcontent.MathDetail{ID: "addition-equation", GroupID: "addition", Title: "选答案", ModuleTitle: "加法", LearningGoal: "算加法", Rules: []string{"计算"}, Revision: 1, Example: mathcontent.MathExample{Kind: "choice", Prompt: "1 + 2 = ?", Options: []string{"1", "2", "3", "4"}, Answer: "3"}}
}
func service(t *testing.T, catalog *mathcontent.Catalog) *Service {
	t.Helper()
	return serviceWithMedia(t, catalog, nil)
}
func serviceWithMedia(t *testing.T, catalog *mathcontent.Catalog, audio map[string][]byte) *Service {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		if r.URL.Path == "/api/v1/math/details/published" {
			_ = json.NewEncoder(w).Encode(catalog)
			return
		}
		if data, ok := audio[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write(data)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	gdb, err := db.OpenSQLite(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	require.NoError(t, Migrate(gdb))
	return New(gdb, srv.URL)
}
func TestGeneratesMathematicallyCorrectFrozenQuestions(t *testing.T) {
	catalog := mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{fixture()}}
	svc := service(t, &catalog)
	task, err := svc.Create(context.Background(), CreateInput{Title: "加法", DetailIDs: []string{"addition-equation"}, RangeMax: 5, Count: 30})
	require.NoError(t, err)
	require.Len(t, task.Items, 30)
	for _, item := range task.Items {
		e := item.Detail.Example
		require.Len(t, e.Counts, 2)
		require.LessOrEqual(t, e.Counts[0]+e.Counts[1], 5)
		require.Equal(t, strconv.Itoa(e.Counts[0]+e.Counts[1]), e.Answer)
		seen := map[string]bool{}
		for _, v := range e.Options {
			require.False(t, seen[v])
			seen[v] = true
		}
		require.True(t, seen[e.Answer])
		require.Equal(t, 1, item.SourceRevision)
		require.Len(t, e.OptionIDs, len(e.Options))
		require.Contains(t, e.OptionIDs, e.AnswerOptionID)
		require.NotContains(t, e.Prompt, "= ?")
	}
	listed, err := svc.List()
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Empty(t, listed[0].Items)
	require.Equal(t, []string{"选答案"}, listed[0].Titles)
	catalog.Items[0].Revision = 2
	catalog.Items[0].Title = "已更改素材"
	saved, err := svc.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, task.Items, saved.Items)
	published, err := svc.Publish(task.ID)
	require.NoError(t, err)
	require.Equal(t, "published", published.Status)
	require.NotNil(t, published.PublishedAt)
	require.False(t, svc.db.Migrator().HasTable("attempts"))
	require.False(t, svc.db.Migrator().HasTable("mastery_skills"))
}
func TestRejectsMissingMaterialAndInvalidRequest(t *testing.T) {
	catalog := mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{}}
	svc := service(t, &catalog)
	_, err := svc.Create(context.Background(), CreateInput{DetailIDs: []string{"missing"}, RangeMax: 5, Count: 1})
	require.Error(t, err)
	catalog.Items = []mathcontent.MathDetail{fixture()}
	for _, in := range []CreateInput{{DetailIDs: []string{"addition-equation"}, RangeMax: 3, Count: 1}, {DetailIDs: []string{"addition-equation"}, RangeMax: 5, Count: 0}, {RangeMax: 5, Count: 1}, {DetailIDs: []string{"missing"}, RangeMax: 5, Count: 1}} {
		_, err = svc.Create(context.Background(), in)
		require.Error(t, err)
	}
	tasks, err := svc.List()
	require.NoError(t, err)
	require.Empty(t, tasks)
}
func TestPublishValidatesFrozenSnapshot(t *testing.T) {
	catalog := mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{fixture()}}
	svc := service(t, &catalog)
	task, err := svc.Create(context.Background(), CreateInput{DetailIDs: []string{"addition-equation"}, RangeMax: 10, Count: 2})
	require.NoError(t, err)
	task.Items[0].Detail.Example.Answer = "999"
	raw, err := json.Marshal(task.Items)
	require.NoError(t, err)
	require.NoError(t, svc.db.Model(&taskRow{}).Where("id = ?", task.ID).Update("items_json", string(raw)).Error)
	_, err = svc.Publish(task.ID)
	require.Error(t, err)
	got, err := svc.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, "draft", got.Status)
}
func TestArithmeticVariantsAndShapes(t *testing.T) {
	for _, group := range []string{"addition", "subtraction"} {
		for _, kind := range []string{"choice", "missing", "objects", "judgement"} {
			for _, multiple := range []bool{false, true} {
				d := fixture()
				d.GroupID = group
				d.Example.Kind = kind
				if kind == "objects" {
					d.Example.Operation = "add"
					d.Example.Counts = []int{1, 2}
					d.Example.Object = "star"
					if group == "subtraction" {
						d.Example.Operation = "sub"
						d.Example.Counts = []int{4, 1}
					}
				}
				if kind == "judgement" {
					d.Example.Statements = []string{"1 + 2 = 3"}
					d.Example.Options = []string{"对", "错"}
					d.Example.Answer = "对"
					if multiple {
						d.GroupID = "subtraction"
						d.Example.Statements = []string{"4 − 1 = 3", "3 − 1 = 3"}
						d.Example.Options = append([]string{}, d.Example.Statements...)
						d.Example.Answer = d.Example.Statements[1]
					}
				}
				for i := 0; i < 30; i++ {
					got, err := generate(d, 5, i)
					require.NoError(t, err)
					require.NoError(t, mathcontent.Validate(got))
					a, b := got.Example.Counts[0], got.Example.Counts[1]
					require.GreaterOrEqual(t, a, 0)
					require.GreaterOrEqual(t, b, 0)
					if group == "subtraction" || multiple && kind == "judgement" {
						require.GreaterOrEqual(t, a, b)
					}
				}
			}
		}
	}
	d := fixture()
	d.ID = "shape-name"
	d.GroupID = "shape"
	d.Example = mathcontent.MathExample{Kind: "shape-name", Prompt: "△", ShapeKeys: []string{"triangle"}, Options: []string{"圆形", "三角形", "正方形"}, Answer: "三角形"}
	got, err := generate(d, 5, 1)
	require.NoError(t, err)
	require.Equal(t, d.Example.Answer, got.Example.Answer)
	require.Equal(t, d.Example.ShapeKeys, got.Example.ShapeKeys)
}

func TestAudioProxyOnlyReadsImmutableFiles(t *testing.T) {
	name := strings.Repeat("a", 64) + ".mp3"
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/api/v1/math/detail-audio/"+name, r.URL.Path)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("audio bytes"))
	}))
	defer srv.Close()
	svc := New(nil, srv.URL)
	data, err := svc.Audio(context.Background(), name)
	require.NoError(t, err)
	require.Equal(t, []byte("audio bytes"), data)
	_, err = svc.Audio(context.Background(), "../../secret")
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
func TestNumericVariantsNeverReuseSourceAudio(t *testing.T) {
	d := fixture()
	d.Example.AudioURL = "/api/v1/math/detail-audio/old.mp3"
	got, err := generate(d, 5, 0)
	require.NoError(t, err)
	require.Empty(t, got.Example.AudioURL)
	require.NotEmpty(t, d.Example.AudioURL)
}
func TestAudioShapeWithoutAudioFailsCreate(t *testing.T) {
	d := fixture()
	d.ID = "shape-find"
	d.GroupID = "shape"
	d.Example = mathcontent.MathExample{Kind: "audio-shape", Prompt: "听一听", Options: []string{"○", "△"}, Answer: "○", ShapeKeys: []string{"circle", "triangle"}}
	svc := service(t, &mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{d}})
	_, err := svc.Create(context.Background(), CreateInput{DetailIDs: []string{"shape-find"}, RangeMax: 5, Count: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "音频")
	tasks, err := svc.List()
	require.NoError(t, err)
	require.Empty(t, tasks)
}
func TestFreezesListenShapeAudioBytes(t *testing.T) {
	name := strings.Repeat("a", 64) + ".mp3"
	source := "/api/v1/math/detail-audio/" + name
	d := fixture()
	d.ID = "shape-find"
	d.GroupID = "shape"
	d.Example = mathcontent.MathExample{Kind: "audio-shape", Prompt: "听一听", Options: []string{"○", "△"}, Answer: "○", ShapeKeys: []string{"circle", "triangle"}, AudioURL: source}
	svc := serviceWithMedia(t, &mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{d}}, map[string][]byte{source: []byte("ID3 frozen math")})
	task, err := svc.Create(context.Background(), CreateInput{DetailIDs: []string{"shape-find"}, RangeMax: 5, Count: 1})
	require.NoError(t, err)
	require.Contains(t, task.Items[0].Detail.Example.AudioURL, "/api/v1/math/task-media/")
	require.NotEqual(t, source, task.Items[0].Detail.Example.AudioURL)
	hash := task.Items[0].MediaSHA256[source]
	require.Len(t, hash, 64)
	data, err := svc.Media(hash)
	require.NoError(t, err)
	require.Equal(t, []byte("ID3 frozen math"), data)
}

func TestFreezesObjectImageBytes(t *testing.T) {
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	source := "/api/v1/math/detail-image/" + strings.Repeat("c", 64) + ".png"
	d := fixture()
	d.ID = "addition-story"
	d.Title = "看数量图选答案"
	d.Example = mathcontent.MathExample{
		Kind: "objects", Prompt: "一共有几颗星星？", Options: []string{"4", "5", "6", "7"}, Answer: "5",
		Operation: "add", Counts: []int{3, 2}, Object: "star", ObjectImageURL: source,
	}
	svc := serviceWithMedia(t, &mathcontent.Catalog{SchemaVersion: 1, Items: []mathcontent.MathDetail{d}}, map[string][]byte{source: png})
	task, err := svc.Create(context.Background(), CreateInput{DetailIDs: []string{"addition-story"}, RangeMax: 5, Count: 1})
	require.NoError(t, err)
	require.Contains(t, task.Items[0].Detail.Example.ObjectImageURL, "/api/v1/math/task-media/")
	require.True(t, strings.HasSuffix(task.Items[0].Detail.Example.ObjectImageURL, ".png"))
	require.NotEqual(t, source, task.Items[0].Detail.Example.ObjectImageURL)
	hash := task.Items[0].MediaSHA256[source]
	require.Len(t, hash, 64)
	data, err := svc.Media(hash)
	require.NoError(t, err)
	require.Equal(t, png, data)
}
