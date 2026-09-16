package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/study-learning/poemcontent"
	"github.com/conchi/study-task-admin/internal/db"
	"github.com/conchi/study-task-admin/internal/poemtask"
	"github.com/stretchr/testify/require"
)

func TestPoemMissingService(t *testing.T) {
	r := NewRouter(Deps{})
	for _, path := range []string{"/api/v1/poem/question-tasks", "/api/v1/poem/question-tasks/1", "/api/v1/poem/task-media/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.wav"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 503, w.Code, path)
	}
}

func TestPoemHTTPGenerationAndMedia(t *testing.T) {
	g, err := db.OpenSQLite("file:poem-http-generation?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, g.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, code TEXT);
		CREATE TABLE modules(id INTEGER PRIMARY KEY AUTOINCREMENT, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER);
		CREATE TABLE knowledge_points(id INTEGER PRIMARY KEY AUTOINCREMENT, module_id INTEGER, code TEXT, title TEXT, payload TEXT, difficulty INTEGER, order_no INTEGER);
		CREATE TABLE questions(id INTEGER PRIMARY KEY AUTOINCREMENT, kp_id INTEGER, code TEXT, type TEXT, stem TEXT, options TEXT, answer TEXT, visual TEXT, speech TEXT, difficulty INTEGER);
		CREATE UNIQUE INDEX uq_questions_kp_code ON questions(kp_id, code);
		INSERT INTO subjects VALUES (1,'poem');
		INSERT INTO modules(id, subject_id, code, name, order_no) VALUES (1,1,'poem50','必背古诗',1);
		INSERT INTO knowledge_points(id, module_id, code, title, payload, difficulty, order_no) VALUES
		 (1,1,'pm001','静夜思','{"kind":"poem","author":"李白","dynasty":"唐","lines":["床前明月光","疑是地上霜","举头望明月","低头思故乡"]}',1,1),
		 (2,1,'pm002','春晓','{"kind":"poem","author":"孟浩然","dynasty":"唐","lines":["春眠不觉晓","处处闻啼鸟","夜来风雨声","花落知多少"]}',1,2),
		 (3,1,'pm003','咏鹅','{"kind":"poem","author":"骆宾王","dynasty":"唐","lines":["鹅鹅鹅","曲项向天歌","白毛浮绿水","红掌拨清波"]}',1,3),
		 (4,1,'pm004','悯农','{"kind":"poem","author":"李绅","dynasty":"唐","lines":["锄禾日当午","汗滴禾下土","谁知盘中餐","粒粒皆辛苦"]}',1,4),
		 (5,1,'pm005','登鹳雀楼','{"kind":"poem","author":"王之涣","dynasty":"唐","lines":["白日依山尽","黄河入海流","欲穷千里目","更上一层楼"]}',1,5);
	`).Error)
	require.NoError(t, poemcontent.EnsureMaterials(g))
	wav := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(poemcontent.SpeechWAV(r.URL.Path))
	}))
	t.Cleanup(wav.Close)
	require.NoError(t, poemtask.Migrate(g))
	r := NewRouter(Deps{Poem: poemtask.New(g, wav.URL)})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := request("POST", "/api/v1/poem/question-tasks", `{"title":"古诗题包","types":["title","fill"],"count":2}`)
	require.Equal(t, 200, w.Code, w.Body.String())
	var saved poemtask.Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	require.Len(t, saved.Items, 2)
	reload := request("GET", fmt.Sprintf("/api/v1/poem/question-tasks/%d", saved.ID), "")
	require.Equal(t, 200, reload.Code)
}
