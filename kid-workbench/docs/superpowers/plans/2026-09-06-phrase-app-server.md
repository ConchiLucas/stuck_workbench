# Phrase App and Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an iPad-first 英语短句孩子端 that starts a real `phrase` study plan from four situation-based question types, writes mastery through `shared-go`, and appears on the parent dashboard.

**Architecture:** New `phrase-app` talks only to `phrase-server` at `/api/v1`. `phrase-server` reads `subjects.code = phrase` questions from shared PostgreSQL, snapshots them into `study_plans` / `plan_items`, and answers through the existing learning transaction. No MinIO: speech is snapshot text plus browser `speechSynthesis`. Do not extend `english-app`.

**Tech Stack:** Go 1.26, Gin, GORM, SQLite tests, React 19, Vite 8, React Router 7, TanStack Query 5, Zustand, Vitest, Docker, Nginx

**Locked product defaults**

- Ports: API `19161`, app `19162`.
- Home is a four-card gallery. Clicking a card `POST`s a type-filtered plan and goes to `/practice/:planId`.
- Types: `listen_zh` 听一听, `listen_en` 选句子, `scene` 什么时候说, `reply` 问与答.
- `reply` only exists for four adjacency pairs; a reply plan may have fewer than 8 items.
- No map page, no content-admin phrase page, no local demo bank, no keyboard typing.

---

## File map

| Path | Responsibility |
| --- | --- |
| `parent-dashboard/backend/internal/seed/catalog_phrase.go` | Scene + reply payload on existing 32 KPs |
| `parent-dashboard/backend/internal/quiz/phrase.go` | Generate four question codes |
| `parent-dashboard/backend/internal/quiz/quiz_test.go` | Generator coverage |
| `phrase-server/` | Independent Go API, `subject_code=phrase`, Postgres only |
| `phrase-app/` | Independent React PWA |
| `docker-compose.yml`, `README.md` | Wire services |

Copy patterns from `english-server` (plan/practice/http envelope) and `pinyin-app` (gallery + `PracticeStage`). Change every `english` string to `phrase`. Do not copy English word types or Pinyin four-line grids.

---

### Task 1: Enrich phrase catalog with scene and reply pairs

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/catalog_phrase.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/quiz_test.go` (payload in existing test)
- Test: `kid-workbench/parent-dashboard/backend/internal/seed/catalog_test.go` (still expects 32 phrase KPs)

- [ ] **Step 1: Write a failing catalog payload test**

Add to `catalog_test.go` next to the existing phrase count assertion, or add `catalog_phrase_test.go`:

```go
package seed

import (
	"encoding/json"
	"testing"
)

func TestPhraseCatalogHasSceneAndReplyPairs(t *testing.T) {
	var scenes, replies int
	for _, mod := range phraseModules() {
		for _, kp := range mod.Kps {
			var p struct {
				Kind    string `json:"kind"`
				Zh      string `json:"zh"`
				Scene   string `json:"scene"`
				ReplyTo string `json:"replyTo"`
			}
			if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil {
				t.Fatal(err)
			}
			if p.Kind != "phrase" || p.Zh == "" || p.Scene == "" {
				t.Fatalf("%s payload incomplete: %+v", kp.Code, p)
			}
			scenes++
			if p.ReplyTo != "" {
				replies++
			}
		}
	}
	if scenes != 32 {
		t.Fatalf("scenes = %d", scenes)
	}
	if replies != 4 {
		t.Fatalf("reply pairs = %d", replies)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/seed -run TestPhraseCatalogHasSceneAndReplyPairs`

Expected: FAIL because `scene` / `replyTo` are missing.

- [ ] **Step 3: Extend the seed struct and payload**

```go
type phraseItem struct {
	En      string
	Zh      string
	Wrong   []string
	Scene   string
	ReplyTo string
	Diff    int
}

func ph(en, zh string, wrong []string, scene, replyTo string) phraseItem {
	return phraseItem{En: en, Zh: zh, Wrong: wrong, Scene: scene, ReplyTo: replyTo, Diff: 1}
}
```

Greet:

```go
{"greet", "问候", "ph", []phraseItem{
	ph("Good morning.", "早上好。", []string{"下午好。", "晚上好。", "晚安。"}, "早上见到老师", ""),
	ph("Good afternoon.", "下午好。", []string{"早上好。", "晚安。", "再见。"}, "下午见到同学", ""),
	ph("Good evening.", "晚上好。", []string{"早上好。", "下午好。", "晚安。"}, "晚上见到家人", ""),
	ph("Good night.", "晚安。", []string{"早上好。", "下午好。", "你好。"}, "睡觉前跟妈妈说", ""),
	ph("Hello!", "你好！", []string{"再见。", "谢谢。", "对不起。"}, "第一次见面打招呼", ""),
	ph("How are you?", "你好吗？", []string{"你叫什么名字？", "再见。", "早上好。"}, "想问问朋友好不好", ""),
	ph("I'm fine.", "我很好。", []string{"我饿了。", "我累了。", "我不舒服。"}, "别人问你好不好", "How are you?"),
	ph("Nice to meet you.", "很高兴见到你。", []string{"再见。", "明天见。", "谢谢你。"}, "刚认识新朋友", "Hello!"),
}},
```

Class: every item gets a classroom scene, `ReplyTo` empty.

```go
ph("Sit down, please.", "请坐下。", []string{"请站起来。", "请举手。", "请安静。"}, "老师让大家坐下", ""),
ph("Stand up, please.", "请站起来。", []string{"请坐下。", "请打开书。", "请合上书。"}, "老师让大家站起来", ""),
ph("Listen to me.", "听我说。", []string{"看着我。", "跟我读。", "请安静。"}, "老师要开始讲课", ""),
ph("Look at me.", "看着我。", []string{"听我说。", "举手。", "坐下。"}, "老师要你看着她", ""),
ph("Open your book.", "打开书。", []string{"合上书本。", "站起来。", "坐下。"}, "开始读书了", ""),
ph("Close your book.", "合上书本。", []string{"打开书。", "举手。", "站起来。"}, "书读完了", ""),
ph("Raise your hand.", "举手。", []string{"坐下。", "站起来。", "安静。"}, "你想发言", ""),
ph("Let's begin.", "我们开始吧。", []string{"再见。", "休息吧。", "请坐下。"}, "课要开始了", ""),
```

Daily:

```go
ph("Thank you.", "谢谢你。", []string{"不客气。", "对不起。", "再见。"}, "别人帮了你", ""),
ph("You're welcome.", "不客气。", []string{"谢谢你。", "对不起。", "再见。"}, "别人跟你说谢谢", "Thank you."),
ph("Excuse me.", "对不起/打扰一下。", []string{"谢谢你。", "再见。", "你好。"}, "想借过或打扰别人", ""),
ph("I'm sorry.", "我很抱歉。", []string{"谢谢你。", "不客气。", "没关系。"}, "不小心撞到别人", ""),
ph("May I come in?", "我可以进来吗？", []string{"请坐下。", "请出去。", "请安静。"}, "想进教室", ""),
ph("What's your name?", "你叫什么名字？", []string{"你好吗？", "再见。", "早上好。"}, "想认识新朋友", ""),
ph("My name is Lily.", "我的名字是莉莉。", []string{"你叫什么名字？", "再见。", "早上好。"}, "别人问你叫什么", "What's your name?"),
ph("See you tomorrow.", "明天见。", []string{"晚安。", "再见。", "早上好。"}, "放学要回家了", ""),
```

Feel: all scenes, no `ReplyTo`.

```go
ph("I like it.", "我喜欢。", []string{"我不喜欢。", "我饿了。", "我累了。"}, "看到喜欢的东西", ""),
ph("I don't like it.", "我不喜欢。", []string{"我喜欢。", "我很好。", "我很高兴。"}, "不想吃某种食物", ""),
ph("I'm hungry.", "我饿了。", []string{"我渴了。", "我累了。", "我很好。"}, "肚子咕咕叫", ""),
ph("I'm thirsty.", "我渴了。", []string{"我饿了。", "我累了。", "我很好。"}, "口很干", ""),
ph("Let's play.", "我们一起玩吧。", []string{"安静。", "坐下。", "再见。"}, "想约小朋友玩", ""),
ph("Come here.", "过来。", []string{"等等我。", "再见。", "坐下。"}, "叫小朋友过来", ""),
ph("Wait for me.", "等等我。", []string{"过来。", "再见。", "开始吧。"}, "你跑得比较慢", ""),
ph("Be quiet.", "安静。", []string{"大声点。", "一起玩。", "站起来。"}, "教室太吵了", ""),
```

Write payload:

```go
Payload: mustPayload(map[string]any{
	"kind": "phrase", "zh": it.Zh, "wrong": it.Wrong,
	"scene": it.Scene, "replyTo": it.ReplyTo,
}),
```

- [ ] **Step 4: Re-run catalog tests**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/seed -count=1`

Expected: PASS, phrase KP count still 32.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/internal/seed
git commit -m "$(cat <<'EOF'
feat: add scene and reply metadata to phrase catalog

EOF
)"
```

---

### Task 2: Generate scene and reply questions

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/phrase.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/quiz_test.go`

- [ ] **Step 1: Extend the existing phrase test and add two new ones**

Keep `TestPhraseListenZhAndEn` passing with the new optional JSON fields. Add:

```go
func TestPhraseSceneUsesSituationNotTheEnglishTitle(t *testing.T) {
	kp := Kp{
		ID: 921, Title: "Good morning.", SubjectCode: "phrase", ModuleCode: "greet",
		Siblings: []string{"Good morning.", "Good afternoon.", "Hello!", "Good night."},
		Payload:  `{"kind":"phrase","zh":"早上好。","wrong":["下午好。","晚上好。","晚安。"],"scene":"早上见到老师"}`,
	}
	specs := Generate(kp)
	scene, ok := find(specs, "scene")
	if !ok {
		t.Fatal("缺少 scene")
	}
	if scene.Stem != "这种时候该说哪一句？" {
		t.Errorf("stem = %q", scene.Stem)
	}
	if scene.Visual.Text != "早上见到老师" {
		t.Errorf("visual = %q", scene.Visual.Text)
	}
	if scene.Options[scene.AnswerIndex].Label != "Good morning." {
		t.Errorf("answer = %q", scene.Options[scene.AnswerIndex].Label)
	}
	if scene.Speech.Text != "Good morning." || scene.Speech.Lang != LangEN {
		t.Errorf("speech = %+v", scene.Speech)
	}
}

func TestPhraseReplyOnlyWhenReplyToPresent(t *testing.T) {
	prompt := Kp{
		ID: 922, Title: "How are you?", SubjectCode: "phrase",
		Siblings: []string{"How are you?", "I'm fine.", "Hello!", "Thank you."},
		Payload:  `{"kind":"phrase","zh":"你好吗？","wrong":["再见。"],"scene":"想问问朋友好不好"}`,
	}
	if _, ok := find(Generate(prompt), "reply"); ok {
		t.Fatal("问句知识点不应生成 reply")
	}
	answer := Kp{
		ID: 923, Title: "I'm fine.", SubjectCode: "phrase",
		Siblings: []string{"How are you?", "I'm fine.", "Hello!", "Thank you."},
		Payload:  `{"kind":"phrase","zh":"我很好。","wrong":["我饿了。"],"scene":"别人问你好不好","replyTo":"How are you?"}`,
	}
	reply, ok := find(Generate(answer), "reply")
	if !ok {
		t.Fatal("缺少 reply")
	}
	if reply.Visual.Text != "How are you?" {
		t.Errorf("prompt visual = %q", reply.Visual.Text)
	}
	if reply.Options[reply.AnswerIndex].Label != "I'm fine." {
		t.Errorf("reply = %q", reply.Options[reply.AnswerIndex].Label)
	}
}
```

- [ ] **Step 2: Run and confirm failure**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz -run 'TestPhrase' -count=1`

Expected: FAIL, `scene` / `reply` missing.

- [ ] **Step 3: Implement generator**

Replace `phrase.go` with:

```go
package quiz

import "encoding/json"

type phrasePayload struct {
	Kind    string   `json:"kind"`
	Zh      string   `json:"zh"`
	Wrong   []string `json:"wrong"`
	Scene   string   `json:"scene"`
	ReplyTo string   `json:"replyTo"`
}

func phraseSpecs(kp Kp) []Spec {
	var p phrasePayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.Kind != "phrase" || p.Zh == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}
	pool := make([]string, 0, len(kp.Siblings))
	for _, s := range kp.Siblings {
		if s != kp.Title {
			pool = append(pool, s)
		}
	}
	if len(pool) < optionCount-1 {
		return nil
	}

	specs := make([]Spec, 0, 4)
	{
		opts, idx := labelOptions(p.Zh, [][]string{p.Wrong}, rngFor(kp.ID, 1))
		specs = append(specs, Spec{
			Code: "listen_zh", Stem: "听一听，选出中文意思",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	{
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 2))
		specs = append(specs, Spec{
			Code: "listen_en", Stem: "听一听，点出这句英语",
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	if p.Scene != "" {
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 3))
		specs = append(specs, Spec{
			Code: "scene", Stem: "这种时候该说哪一句？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "scene", Text: p.Scene},
			Speech: Speech{Text: kp.Title, Lang: LangEN},
		})
	}
	if p.ReplyTo != "" {
		opts, idx := labelOptions(kp.Title, [][]string{pool}, rngFor(kp.ID, 4))
		specs = append(specs, Spec{
			Code: "reply", Stem: "对方说了这句话，你怎么答？",
			Options: opts, AnswerIndex: idx,
			Visual: Visual{Kind: "prompt", Text: p.ReplyTo},
			Speech: Speech{Text: p.ReplyTo, Lang: LangEN},
		})
	}
	return specs
}
```

- [ ] **Step 4: Run quiz tests**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/internal/quiz
git commit -m "$(cat <<'EOF'
feat: generate scene and reply phrase questions

EOF
)"
```

---

### Task 3: Scaffold phrase-server health endpoints

**Files:**
- Create: `kid-workbench/phrase-server/go.mod`
- Create: `kid-workbench/phrase-server/internal/config/config.go`
- Create: `kid-workbench/phrase-server/internal/db/db.go`
- Create: `kid-workbench/phrase-server/internal/http/response.go`
- Create: `kid-workbench/phrase-server/internal/http/router.go`
- Create: `kid-workbench/phrase-server/internal/http/router_test.go`
- Create: `kid-workbench/phrase-server/cmd/server/main.go`

- [ ] **Step 1: Write health tests**

```go
package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/conchi/phrase-server/internal/http"
)

func TestHealthz(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
}

func TestReadyzWithoutChecker(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
```

- [ ] **Step 2: Run and confirm failure**

Run: `cd kid-workbench/phrase-server && go test ./internal/http -count=1`

Expected: FAIL, module/package missing.

- [ ] **Step 3: Implement scaffold**

`go.mod`:

```
module github.com/conchi/phrase-server

go 1.26.1

require (
	github.com/conchi/study-learning v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.12.0
	github.com/glebarez/sqlite v1.11.0
	github.com/stretchr/testify v1.12.1
	gorm.io/driver/postgres v1.6.2
	gorm.io/gorm v1.31.2
)

replace github.com/conchi/study-learning => ../shared-go
```

Copy `english-server/internal/db/db.go` and `internal/http/response.go`, changing the import path to `github.com/conchi/phrase-server`. Config default addr `:19161`, no MinIO fields.

Router:

```go
func NewRouter(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Readiness == nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "服务依赖尚未就绪")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := deps.Readiness.Check(ctx); err != nil {
			writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "服务依赖尚未就绪")
			return
		}
		writeData(c, stdhttp.StatusOK, gin.H{"status": "ready"})
	})
	return router
}
```

`main.go` opens Postgres and serves with `Readiness: db.Checker{DB: database}` only.

Then `go mod tidy`.

- [ ] **Step 4: Run tests**

Run: `cd kid-workbench/phrase-server && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/phrase-server
git commit -m "$(cat <<'EOF'
feat: scaffold phrase-server health endpoints

EOF
)"
```

---

### Task 4: Create type-filtered phrase plans

**Files:**
- Create: `kid-workbench/phrase-server/internal/plan/model.go`
- Create: `kid-workbench/phrase-server/internal/plan/service.go`
- Create: `kid-workbench/phrase-server/internal/plan/service_test.go`

- [ ] **Step 1: Write the plan test**

Use SQLite in-memory. Schema can omit `english_assets`. Insert one `phrase` module with four questions (`listen_zh`, `listen_en`, `scene`, `reply`) and one `pinyin` question that must never appear.

```go
func TestCreatePlanFiltersByPhraseQuestionCode(t *testing.T) {
	db := openDB(t)
	s := plan.NewService(db)
	created, err := s.Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "scene", Count: 8})
	require.NoError(t, err)
	require.Equal(t, "phrase", created.Plan.SubjectCode)
	require.Len(t, created.Items, 1)
	require.Equal(t, "scene", created.Items[0].Question.Code)
	require.Equal(t, "早上见到老师", created.Items[0].Scene)
	require.NoError(t, db.Table("questions").Where("id=?", created.Items[0].Question.ID).Update("stem", "changed").Error)
	again, err := s.Get(context.Background(), 1, created.Plan.ID)
	require.NoError(t, err)
	require.Equal(t, created.Items[0].Question.Stem, again.Items[0].Question.Stem)
}

func TestCreateRejectsUnknownType(t *testing.T) {
	_, err := plan.NewService(openDB(t)).Create(context.Background(), 1, plan.CreateInput{Mode: "type", QuestionCode: "listen"})
	require.ErrorIs(t, err, plan.ErrInvalidType)
}
```

Seed SQL must include `plan_items` snapshot columns used by `english-server` tests.

- [ ] **Step 2: Run and confirm failure**

Run: `cd kid-workbench/phrase-server && go test ./internal/plan -count=1`

Expected: FAIL, package missing.

- [ ] **Step 3: Implement plan service**

`CreateInput`:

```go
type CreateInput struct {
	Mode         string `json:"mode"`
	QuestionCode string `json:"questionCode"`
	Count        int    `json:"count"`
}

var phraseCodes = []string{"listen_zh", "listen_en", "scene", "reply"}
```

Default `mode` to `today` (all four codes). `mode=type` requires `QuestionCode` in `phraseCodes`. Query:

```sql
JOIN knowledge_points kp ... JOIN modules m ... JOIN subjects s
WHERE s.code = 'phrase' AND q.code IN ?
```

Do not join MinIO/asset tables. Speech is valid if `speech.text` is non-empty. Snapshot stem/options/answer/visual/speech onto `plan_items`. `Item` DTO:

```go
type Item struct {
	ID, Seq, KpID, Tries int
	Phrase, MeaningZh, Scene, ReplyTo, Bucket, Status, Picks, OptionOrder string
	Question Question
}
```

Parse `kp.payload` for `zh`, `scene`, `replyTo`. Shuffle option order like `english-server`. `TargetCount` is `len(candidates)`, which may be < requested count.

- [ ] **Step 4: Run plan tests**

Run: `cd kid-workbench/phrase-server && go test ./internal/plan -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/phrase-server/internal/plan
git commit -m "$(cat <<'EOF'
feat: create snapshot phrase plans by question type

EOF
)"
```

---

### Task 5: Answer and finish through shared-go

**Files:**
- Create: `kid-workbench/phrase-server/internal/practice/service.go`
- Create: `kid-workbench/phrase-server/internal/practice/service_test.go`

- [ ] **Step 1: Copy the english practice test, scoped to phrase**

Assert: wrong first try can retry; second correct completes the item; same `clientId` is idempotent; `Finish` needs all items done; `subject_code` stays `phrase`.

- [ ] **Step 2: Run and confirm failure**

Run: `cd kid-workbench/phrase-server && go test ./internal/practice -count=1`

Expected: FAIL.

- [ ] **Step 3: Implement practice**

Copy `english-server/internal/practice/service.go` and replace:

- import path `phrase-server`
- every `subject_code = 'english'` → `'phrase'`
- `FinishResult.WeakWords` → `WeakPhrases` with `{kpId, phrase, meaningZh}`
- parse payload `zh` instead of `meaningZh`

Keep two-try rule, `learning.ApplyOne`, flower ledger sum.

- [ ] **Step 4: Run practice tests**

Run: `cd kid-workbench/phrase-server && go test ./internal/practice -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/phrase-server/internal/practice
git commit -m "$(cat <<'EOF'
feat: record phrase answers with shared learning kernel

EOF
)"
```

---

### Task 6: Expose phrase HTTP API

**Files:**
- Modify: `kid-workbench/phrase-server/internal/http/router.go`
- Create: `kid-workbench/phrase-server/internal/http/handler_plan.go`
- Create: `kid-workbench/phrase-server/internal/http/handler_answer.go`
- Create: `kid-workbench/phrase-server/internal/http/handler_plan_test.go`
- Modify: `kid-workbench/phrase-server/cmd/server/main.go`

- [ ] **Step 1: Write a router integration test**

SQLite-backed real `plan.Service` + `practice.Service`. `POST /api/v1/children/1/phrase/plans` with `{"mode":"type","questionCode":"listen_zh","count":2}` returns 201, `data.plan.subjectCode=phrase`, items have no answer index. Then answer the pending item.

- [ ] **Step 2: Run and confirm failure**

Run: `cd kid-workbench/phrase-server && go test ./internal/http -count=1`

Expected: FAIL, 404 on `/api/v1/children/1/phrase/plans`.

- [ ] **Step 3: Wire routes**

```go
plans := router.Group("/api/v1/children/:childId/phrase/plans")
plans.POST("", createPlan(deps.Plans))
plans.GET("/:planId", getPlan(deps.Plans))
plans.POST("/:planId/start", startPlan(deps.Plans))
plans.POST("/:planId/items/:itemId/answer", answerPlanItem(deps.Practice))
plans.POST("/:planId/finish", finishPlan(deps.Practice))
```

Copy english handlers; error messages say「短句题单」not 英语. `main.go` constructs plan + practice services.

- [ ] **Step 4: Run all server tests**

Run: `cd kid-workbench/phrase-server && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/phrase-server
git commit -m "$(cat <<'EOF'
feat: expose phrase plan and answer HTTP API

EOF
)"
```

---

### Task 7: Scaffold phrase-app

**Files:**
- Create: `kid-workbench/phrase-app/package.json`
- Create: `kid-workbench/phrase-app/vite.config.ts`
- Create: `kid-workbench/phrase-app/tsconfig.json`, `tsconfig.app.json`, `tsconfig.node.json`
- Create: `kid-workbench/phrase-app/index.html`
- Create: `kid-workbench/phrase-app/src/main.tsx`
- Create: `kid-workbench/phrase-app/src/App.tsx`
- Create: `kid-workbench/phrase-app/src/App.test.tsx`
- Create: `kid-workbench/phrase-app/src/test/setup.ts`
- Create: `kid-workbench/phrase-app/src/styles/tokens.css`
- Create: `kid-workbench/phrase-app/src/styles/app.css`

- [ ] **Step 1: Copy pinyin-app package.json / tsconfig / vitest setup**

Change name to `phrase-app`. Vite port `19162`, proxy `/api` → `http://localhost:19161`. Title: `短句小舞台`. Copy `pinyin-app/src/styles/tokens.css` as the token baseline.

- [ ] **Step 2: Write a failing home test**

```tsx
it('renders four phrase type cards', () => {
  render(<MemoryRouter><AppRoutes /></MemoryRouter>)
  expect(screen.getByRole('button', { name: '听一听' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '选句子' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '什么时候说' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '问与答' })).toBeInTheDocument()
})
```

Do not use Pinyin titles or English-app titles (`听音选词`, `看图选词`).

- [ ] **Step 3: Run and confirm failure**

Run: `cd kid-workbench/phrase-app && npm install && npm test -- --run src/App.test.tsx`

Expected: FAIL.

- [ ] **Step 4: Minimal routes**

```tsx
<Route element={<Shell />}>
  <Route index element={<HomePage />} />
  <Route path="practice/:planId" element={<PracticePage />} />
  <Route path="practice/:planId/result" element={<ResultPage />} />
</Route>
```

Home can render four disabled-looking buttons first; Practice/Result can be placeholders.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/phrase-app
git commit -m "$(cat <<'EOF'
feat: scaffold phrase kid app shell

EOF
)"
```

---

### Task 8: Typed client and pending answer IDs

**Files:**
- Create: `kid-workbench/phrase-app/src/api/client.ts`
- Create: `kid-workbench/phrase-app/src/api/types.ts`
- Create: `kid-workbench/phrase-app/src/api/phrase.ts`
- Create: `kid-workbench/phrase-app/src/api/client.test.ts`
- Create: `kid-workbench/phrase-app/src/store/childStore.ts`
- Create: `kid-workbench/phrase-app/src/store/pendingAnswerStore.ts`
- Create: `kid-workbench/phrase-app/src/store/pendingAnswerStore.test.ts`

- [ ] **Step 1: Tests**

Copy pinyin `client.ts` envelope parsing tests. Assert default child `1`, and one UUID per `planId:itemId:try`.

Types:

```ts
export type PhraseCode = 'listen_zh' | 'listen_en' | 'scene' | 'reply'
export type QuestionOption = { label: string }
export interface PlanQuestion {
  id: number
  code: PhraseCode
  stem: string
  options: QuestionOption[]
  visual: { kind?: string; text?: string }
  speech: { text?: string; lang?: string }
}
```

No answer index on GET plan.

```ts
createPlan: (childId, body: { mode: 'type'; questionCode: PhraseCode; count?: number }) =>
  api.post(`/children/${childId}/phrase/plans`, body)
```

- [ ] **Step 2: Implement and run**

Run: `cd kid-workbench/phrase-app && npm test -- --run src/api src/store`

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add kid-workbench/phrase-app/src/api kid-workbench/phrase-app/src/store
git commit -m "$(cat <<'EOF'
feat: add phrase API client and answer idempotency keys

EOF
)"
```

---

### Task 9: Home gallery starts a real type plan

**Files:**
- Create: `kid-workbench/phrase-app/src/pages/HomePage.tsx`
- Modify: `kid-workbench/phrase-app/src/App.test.tsx`
- Modify: `kid-workbench/phrase-app/src/styles/app.css`

- [ ] **Step 1: Expand the home test**

Stub `fetch`. Clicking「什么时候说」POSTs `{"mode":"type","questionCode":"scene"}` to `/api/v1/children/1/phrase/plans` and navigates to `/practice/9`.

```tsx
expect(await screen.findByLabelText('当前题目')).toBeInTheDocument()
```

- [ ] **Step 2: Implement HomePage**

Four cards, CSS-only (no PNG pipeline):

```ts
const types = [
  { code: 'listen_zh', title: '听一听', sample: 'Good morning.' },
  { code: 'listen_en', title: '选句子', sample: 'Sit down, please.' },
  { code: 'scene', title: '什么时候说', sample: '早上见到老师' },
  { code: 'reply', title: '问与答', sample: 'How are you?' },
] as const
```

On click: `createPlan` then `navigate(/practice/${plan.id})`. Show a retry card on error. Gallery is immersive like pinyin (no topbar).

- [ ] **Step 3: Run tests and commit**

Run: `cd kid-workbench/phrase-app && npm test -- --run src/App.test.tsx`

```bash
git add kid-workbench/phrase-app
git commit -m "$(cat <<'EOF'
feat: start real phrase plans from type gallery

EOF
)"
```

---

### Task 10: Practice and result for four phrase interactions

**Files:**
- Create: `kid-workbench/phrase-app/src/pages/PracticePage.tsx`
- Create: `kid-workbench/phrase-app/src/pages/PracticePage.test.tsx`
- Create: `kid-workbench/phrase-app/src/pages/ResultPage.tsx`
- Create: `kid-workbench/phrase-app/src/components/PracticeStage.tsx`
- Create: `kid-workbench/phrase-app/src/components/ListenButton.tsx`

- [ ] **Step 1: Write practice tests with a stubbed plan**

One pending `scene` item: prompt visual「早上见到老师」, four English options. Clicking an option POSTs answer with `clientId`. After the item is no longer pending, finish is called and result heading「答题结果」shows.

Add a `reply` case: visual shows `How are you?`, stem「对方说了这句话，你怎么答？」.

Add a `listen_zh` case: play button uses `speechSynthesis` with `en-US`, options are Chinese.

- [ ] **Step 2: Implement rendering rules**

| code | Prompt area | Options |
| --- | --- | --- |
| `listen_zh` | Play English audio | Chinese labels |
| `listen_en` | Play English audio | English sentences |
| `scene` | Big Chinese situation, optional play of the answer phrase after pick | English sentences |
| `reply` | Big English prompt + play the prompt | English replies |

Copy `pinyin-app` `PracticeStage` (exit, progress, no prev/next for real plans). Current item = first `pending`. Answer uses pending store. On last item done, `POST finish` then `/practice/:id/result`.

Result list: phrase text + 答对/答错. Link back home.

Do not render four-line grids, number equations, or plant labels.

- [ ] **Step 3: Run frontend tests**

Run: `cd kid-workbench/phrase-app && npm test -- --run && npm run build`

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add kid-workbench/phrase-app
git commit -m "$(cat <<'EOF'
feat: practice and result real phrase plans

EOF
)"
```

---

### Task 11: Docker, compose, and README

**Files:**
- Create: `kid-workbench/phrase-server/Dockerfile`
- Create: `kid-workbench/phrase-app/Dockerfile`
- Create: `kid-workbench/phrase-app/nginx.conf`
- Create: `kid-workbench/phrase-app/README.md`
- Create: `kid-workbench/phrase-server/README.md`
- Modify: `kid-workbench/docker-compose.yml`
- Modify: `kid-workbench/README.md`

- [ ] **Step 1: Dockerfiles**

Server Dockerfile like `english-server/Dockerfile`, `APP_ADDR=:19161`, healthcheck `19161`. App Dockerfile like `pinyin-app`. Nginx proxies `/api/` to `http://phrase-server:19161`.

Compose after `english-app`:

```yaml
  phrase-server:
    build:
      context: .
      dockerfile: phrase-server/Dockerfile
    environment:
      <<: *db_env
      APP_ADDR: :19161
    ports:
      - "19161:19161"
    depends_on:
      seed:
        condition: service_completed_successfully
    restart: unless-stopped

  phrase-app:
    build:
      context: .
      dockerfile: phrase-app/Dockerfile
    ports:
      - "19162:80"
    depends_on:
      phrase-server:
        condition: service_healthy
    restart: unless-stopped
```

No MinIO env. README table:

| 英语短句 API | http://localhost:19161 |
| 英语短句孩子端 | http://localhost:19162 |

Local run notes: `go run ./cmd/server` and `npm run dev`. Seed must be re-run after catalog/quiz changes so `scene`/`reply` rows exist.

- [ ] **Step 2: Verify**

Run:

```bash
cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed -count=1
cd kid-workbench/phrase-server && go test ./... -count=1
cd kid-workbench/phrase-app && npm test -- --run && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add kid-workbench/phrase-server/Dockerfile kid-workbench/phrase-app/Dockerfile kid-workbench/phrase-app/nginx.conf kid-workbench/phrase-app/README.md kid-workbench/phrase-server/README.md kid-workbench/docker-compose.yml kid-workbench/README.md
git commit -m "$(cat <<'EOF'
feat: wire phrase app and server into workbench compose

EOF
)"
```

---

## Manual check after implementation

1. `make seed` so new question codes exist.
2. Open http://localhost:19162.
3. Tap each of the four cards; each should start a plan and speak English (browser TTS).
4. Finish a set; parent dashboard subject 英语短句 should show updated mastery.

If `reply` has only four questions, a 4-item plan is correct. Adjust copy or pair count later from live effect, not in this plan.