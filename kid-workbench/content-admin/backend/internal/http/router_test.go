package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/catalog"
	"github.com/conchi/study-content-admin/internal/chengyu"
	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/db"
	httpapi "github.com/conchi/study-content-admin/internal/http"
	"github.com/conchi/study-content-admin/internal/logic"
	contentmath "github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-content-admin/internal/phrase"
	"github.com/conchi/study-content-admin/internal/pinyin"
	"github.com/conchi/study-content-admin/internal/poem"
	"github.com/conchi/study-content-admin/internal/science"
)

func TestPoemSyncAndListRoutes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
CREATE TABLE questions (id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INTEGER);
CREATE UNIQUE INDEX uq_questions_kp_code ON questions(kp_id, code);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'poem', '古诗', '', 6);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'poem50', '必背古诗', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no)
VALUES (1, 1, 'pm001', '静夜思', '{"kind":"poem","author":"李白","line1":"床前明月光","line2":"疑是地上霜","lines":["床前明月光","疑是地上霜"]}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	r := httpapi.NewRouter(httpapi.Deps{Poem: poem.NewService(gdb)})

	status, body := scienceRequest(t, r, http.MethodPost, "/api/v1/poem/sync", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), body["total"])

	status, body = scienceRequest(t, r, http.MethodGet, "/api/v1/poem/items?view=groups", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), body["total"])
	groups := body["groups"].([]any)
	require.Len(t, groups, 1)
	first := groups[0].(map[string]any)
	require.Equal(t, "必背古诗", first["moduleName"])
	item := first["items"].([]any)[0].(map[string]any)
	require.Equal(t, "静夜思", item["title"])
	require.Equal(t, "李白", item["author"])
}

func TestLogicListRoutes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'logic', '逻辑', '', 7);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'pattern', '找规律', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no)
VALUES (1, 1, 'pt001', '红蓝交替', '{"kind":"pattern","seq":["🔴","🔵"],"a":"🔴","wrong":["🟢"],"prompt":"下一个是哪个？"}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	r := httpapi.NewRouter(httpapi.Deps{Logic: logic.NewService(gdb)})

	status, body := scienceRequest(t, r, http.MethodGet, "/api/v1/logic/items?view=groups", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), body["total"])
	groups := body["groups"].([]any)
	require.Len(t, groups, 1)
	first := groups[0].(map[string]any)
	require.Equal(t, "找规律", first["moduleName"])
	item := first["items"].([]any)[0].(map[string]any)
	require.Equal(t, "红蓝交替", item["title"])
	require.Equal(t, "🔴", item["answer"])
}

func TestChengyuListRoutes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'chengyu', '成语', '', 8);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'daily', '日常成语', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no)
VALUES (1, 1, 'cy001', '一心一意', '{"kind":"chengyu","pinyin":"yì xīn yì yì","meaning":"集中精神，做事专心","example":"做作业要一心一意。","wrong":["三心二意"]}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	r := httpapi.NewRouter(httpapi.Deps{Chengyu: chengyu.NewService(gdb)})

	status, body := scienceRequest(t, r, http.MethodGet, "/api/v1/chengyu/items?view=groups", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), body["total"])
	groups := body["groups"].([]any)
	require.Len(t, groups, 1)
	first := groups[0].(map[string]any)
	require.Equal(t, "日常成语", first["moduleName"])
	item := first["items"].([]any)[0].(map[string]any)
	require.Equal(t, "一心一意", item["title"])
	require.Equal(t, "yì xīn yì yì", item["pinyin"])
	require.Equal(t, "集中精神，做事专心", item["meaning"])
}

func TestPhraseListRoutes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT, name TEXT, icon TEXT, order_no INT);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INT, code TEXT, name TEXT, order_no INT);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INT, code TEXT, title TEXT, payload TEXT, difficulty INT, order_no INT);
INSERT INTO subjects(id, code, name, icon, order_no) VALUES (1, 'phrase', '英语短句', '', 9);
INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1, 1, 'greet', '问候', 1);
INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no)
VALUES (1, 1, 'ph001', 'Good morning.', '{"kind":"phrase","zh":"早上好。","wrong":["下午好。"],"scene":"早上见到老师","replyTo":""}', 1, 1);
`).Error)
	require.NoError(t, db.Migrate(gdb))
	r := httpapi.NewRouter(httpapi.Deps{Phrase: phrase.NewService(gdb)})

	status, body := scienceRequest(t, r, http.MethodGet, "/api/v1/phrase/items?view=groups", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, float64(1), body["total"])
	groups := body["groups"].([]any)
	require.Len(t, groups, 1)
	first := groups[0].(map[string]any)
	require.Equal(t, "问候", first["moduleName"])
	item := first["items"].([]any)[0].(map[string]any)
	require.Equal(t, "Good morning.", item["title"])
	require.Equal(t, "早上好。", item["zh"])
	require.Equal(t, "早上见到老师", item["scene"])
}

func TestHealthz(t *testing.T) {
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(configclient.New(""))})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGeneratePinyinQuizRoute(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	assets := []pinyin.Asset{
		{KpID: 1, Letter: "b", ModuleCode: "initials", SoloText: "波", WordText: "爸", GlyphImageURL: "/b.png"},
		{KpID: 2, Letter: "p", ModuleCode: "initials", SoloText: "坡", WordText: "怕", GlyphImageURL: "/p.png"},
		{KpID: 3, Letter: "m", ModuleCode: "initials", SoloText: "摸", WordText: "妈", GlyphImageURL: "/m.png"},
		{KpID: 4, Letter: "f", ModuleCode: "initials", SoloText: "佛", WordText: "飞", GlyphImageURL: "/f.png"},
	}
	require.NoError(t, gdb.Create(&assets).Error)
	r := httpapi.NewRouter(httpapi.Deps{Pinyin: pinyin.NewService(gdb, nil, nil, nil, nil)})

	status, body := scienceRequest(t, r, http.MethodPost, "/api/v1/pinyin/quiz/generate", map[string]any{
		"type": "listen", "excludeTargetIds": []int64{1, 2, 3},
	})
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "listen", body["type"])
	require.Equal(t, float64(4), body["targetId"])
	require.Len(t, body["options"], 4)

	status, body = scienceRequest(t, r, http.MethodPost, "/api/v1/pinyin/quiz/generate", map[string]any{"type": "unknown"})
	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "无效的拼音题型", body["error"])
}

func TestSciencePublicationWorkflow(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, gdb.Exec(`CREATE TABLE questions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kp_id INTEGER NOT NULL,
		code TEXT NOT NULL
	)`).Error)
	require.NoError(t, gdb.Create(&science.Asset{
		KpID: 7, Title: "冬眠", NeedsSenseImage: true,
		ReviewStatus: "draft", ContentVersion: 1,
	}).Error)
	r := httpapi.NewRouter(httpapi.Deps{Science: science.NewService(gdb, nil, nil, nil, nil, nil, nil)})

	status, body := scienceRequest(t, r, http.MethodPost, "/api/v1/science/items/7/publish", nil)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, "review_required", body["code"])

	status, body = scienceRequest(t, r, http.MethodPost, "/api/v1/science/items/7/review", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "reviewed", body["reviewStatus"])

	status, body = scienceRequest(t, r, http.MethodPost, "/api/v1/science/items/7/publish", nil)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, "no_published_question", body["code"])

	require.NoError(t, gdb.Exec(`INSERT INTO questions(kp_id, code) VALUES (7, 'recognize')`).Error)
	status, body = scienceRequest(t, r, http.MethodPost, "/api/v1/science/items/7/publish", nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "published", body["reviewStatus"])

	status, body = scienceRequest(t, r, http.MethodPatch, "/api/v1/science/items/7/content", map[string]any{
		"summary": "动物过冬", "explanation": "冬眠能节省能量", "funFact": "心跳会变慢",
	})
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "draft", body["reviewStatus"])
	require.Equal(t, float64(2), body["contentVersion"])
}

func scienceRequest(t *testing.T, handler http.Handler, method, path string, payload any) (int, map[string]any) {
	t.Helper()
	var raw []byte
	if payload != nil {
		var err error
		raw, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body
}

func TestCatalogUnavailable(t *testing.T) {
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(configclient.New(""))})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/runtime-config/catalog", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "无法加载共享配置中心", body["error"])
}

func TestCatalogOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/runtime/v1/configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"schemaVersion": "1",
				"generatedAt":   time.Now().UTC().Format(time.RFC3339),
				"ai": map[string]any{
					"activeProviderId": "p1",
					"providers": []map[string]any{{
						"id": "p1", "label": "Demo", "type": "openai-compatible",
						"baseUrl": "http://x", "apiKey": "k", "model": "m",
						"maxTokens": 100, "capabilities": []string{"TEXT_GENERATION"},
						"options": map[string]string{}, "enabled": true,
					}},
				},
				"databases":     []any{},
				"objectStorage": map[string]any{"configured": false},
				"localCli":      map[string]any{"activeConfigId": "", "configs": []any{}},
			})
		case "/api/admin/v1/configuration/image-models", "/api/admin/v1/configuration/video-models", "/api/admin/v1/configuration/voice-models":
			_ = json.NewEncoder(w).Encode(map[string]any{"activeProviderId": "", "providers": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := configclient.New(srv.URL)
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(client)})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/runtime-config/catalog", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body catalog.Catalog
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "p1", body.AI.Active)
	require.Len(t, body.AI.Providers, 1)
	require.True(t, body.AI.Providers[0].Active)
}

func TestMathQuestionSpeechRoutes(t *testing.T) {
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT NOT NULL);
CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER NOT NULL, code TEXT NOT NULL, order_no INTEGER NOT NULL DEFAULT 0);
CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER NOT NULL, order_no INTEGER NOT NULL DEFAULT 0);
CREATE TABLE questions (id INTEGER PRIMARY KEY, kp_id INTEGER NOT NULL, code TEXT NOT NULL, speech TEXT NOT NULL DEFAULT '{}', media_url TEXT NOT NULL DEFAULT '');
INSERT INTO subjects(id, code) VALUES (1, 'math');
INSERT INTO modules(id, subject_id, code) VALUES (1, 1, 'add10');
INSERT INTO knowledge_points(id, module_id) VALUES (10, 1);
INSERT INTO questions(id, kp_id, code, speech) VALUES (42, 10, 'calc', '{"text":"二加五等于几"}');
`).Error)
	r := httpapi.NewRouter(httpapi.Deps{Math: contentmath.NewService(gdb, nil, nil, nil, nil)})

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/v1/math/question-speech/status", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/math/question-speech/status?moduleCode=add10", http.StatusOK},
		{http.MethodPost, "/api/v1/math/question-speech/batch", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/math/questions/not-a-number/speech", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/math/questions/999/speech", http.StatusNotFound},
		{http.MethodPost, "/api/v1/math/questions/42/speech", http.StatusServiceUnavailable},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.want, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
	}
}

func TestQuestionTasksMovedOut(t *testing.T) {
	r := httpapi.NewRouter(httpapi.Deps{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/question-tasks", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "接口不存在", body["error"])
}
