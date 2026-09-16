# 单词乐园题型练习闭环 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `english-app` play like the other kid subject galleries: five type cards, four questions each, instant right/wrong, and live word-bank quizzes for 听音选词 and 看图选词.

**Architecture:** Keep the existing gallery routes. Local demos remain the fallback. Add `english-server` `POST /api/v1/english/quiz/generate` modeled on pinyin runtime quizzes. `audio-choice` maps to `listen`, `image-text` maps to `look`. Official plan/practice APIs stay unchanged and are not used by the gallery.

**Tech Stack:** Go 1.26, Gin, GORM, SQLite tests, React 19, TypeScript, Vite, Vitest, Testing Library

**Spec:** `kid-workbench/docs/superpowers/specs/2026-09-06-english-type-practice-loop-design.md`

**Commit policy:** Skip `git commit` steps unless the user explicitly asks to commit.

---

## File map

| File | Responsibility |
| --- | --- |
| `kid-workbench/english-server/internal/quiz/service.go` | Build one `listen` or `look` question from English KPs + assets |
| `kid-workbench/english-server/internal/quiz/service_test.go` | SQLite-backed generator tests |
| `kid-workbench/english-server/internal/http/handler_quiz.go` | POST generate handler |
| `kid-workbench/english-server/internal/http/router.go` | Register `/api/v1/english/quiz/generate` |
| `kid-workbench/english-server/internal/http/router_test.go` | Route tests |
| `kid-workbench/english-server/cmd/server/main.go` | Wire `quiz.NewService` |
| `kid-workbench/english-app/src/api/client.ts` | JSON envelope + non-JSON error |
| `kid-workbench/english-app/src/api/types.ts` | Generated quiz DTOs |
| `kid-workbench/english-app/src/api/english.ts` | `generateQuiz` + set helper |
| `kid-workbench/english-app/src/store/liveQuizStore.ts` | Load 4 questions, fallback flag |
| `kid-workbench/english-app/src/QuestionTypePreview.tsx` | Instant feedback, one retry, auto-advance |
| `kid-workbench/english-app/src/App.tsx` | Use live quizzes for listen/look |
| `kid-workbench/english-app/src/App.test.tsx` | Gallery, feedback, fallback |

Do not rewrite `english-server` plan/practice packages in this plan.

---

### Task 1: Instant right/wrong on local choice demos

**Files:**
- Modify: `kid-workbench/english-app/src/QuestionTypePreview.tsx`
- Modify: `kid-workbench/english-app/src/App.test.tsx`
- Modify: `kid-workbench/english-app/src/styles.css` (only if existing `.kid-option.is-right` / `.is-wrong` / `.kid-feedback` are missing)

- [ ] **Step 1: Rewrite the choice-feedback assertions so they fail**

In `script detail is a kid practice screen, not a design brief`, replace the block that forbids status text:

```tsx
fireEvent.click(screen.getByRole('button', { name: '苹果' }))
expect(screen.getByRole('status')).toHaveTextContent('答对了')
expect(screen.getByRole('button', { name: '苹果' })).toHaveClass('is-right')
```

Add a new test after it:

```tsx
test('a wrong first tap can be retried once', () => {
  render(<MemoryRouter initialEntries={['/types/audio-choice']}><App /></MemoryRouter>)
  fireEvent.click(screen.getByRole('button', { name: '香蕉' }))
  expect(screen.getByRole('status')).toHaveTextContent('再试一次')
  fireEvent.click(screen.getByRole('button', { name: '苹果' }))
  expect(screen.getByRole('status')).toHaveTextContent('答对了')
})
```

Keep the existing “not a design brief” assertions that hide 效果预览 / 覆盖题型 / 提交答案.

- [ ] **Step 2: Run the failing tests**

Run: `cd kid-workbench/english-app && npm test -- src/App.test.tsx`

Expected: FAIL because clicking 苹果 still only adds `is-picked` and there is no `status` node.

- [ ] **Step 3: Add choice grading in `ChoicePlay`**

Keep `ListenButton`, `Face`, `Stage`, `TypePlay`, `TilePlay`. Change `ChoicePlay` so a tap grades against `answerId`. First wrong tap stays unlocked. Correct tap or second wrong tap locks the board, marks `is-right` / `is-wrong`, and calls `onChange(answerId)` only when finished so the result page still records a final pick.

```tsx
function ChoicePlay({
  prompt, speech, options, lead, answerId, value, onChange, nextTo,
}: {
  prompt?: string
  speech?: string
  options: Choice[]
  lead?: ReactNode
  answerId: string
  value?: string
  onChange: (value: string) => void
  nextTo?: string
}) {
  const navigate = useNavigate()
  const [picked, setPicked] = useState<string | undefined>(value)
  const [tries, setTries] = useState(0)
  const locked = Boolean(value) || tries >= 2 || picked === answerId
  const correct = picked === answerId
  useEffect(() => {
    if (!locked || !nextTo) return
    const timer = window.setTimeout(() => navigate(nextTo), 450)
    return () => window.clearTimeout(timer)
  }, [locked, nextTo, navigate])
  return (
    <Stage
      prompt={prompt}
      lead={lead}
      action={speech ? <ListenButton onClick={() => speak(speech)} /> : undefined}
      feedback={picked ? (correct ? '答对了' : tries >= 2 ? '看正确答案' : '再试一次') : undefined}
    >
      <div className="kid-options">
        {options.map((option) => {
          const isPicked = picked === option.id
          const showRight = locked && option.id === answerId
          const showWrong = locked && isPicked && option.id !== answerId
          return (
            <button
              key={option.id}
              className={`kid-option${isPicked ? ' is-picked' : ''}${showRight ? ' is-right' : ''}${showWrong ? ' is-wrong' : ''}${hideEnglishOnPicture(option.label, option.picture) ? ' is-pic' : ''}`}
              aria-label={option.label}
              aria-pressed={isPicked}
              disabled={locked}
              onClick={() => {
                if (locked) return
                setPicked(option.id)
                const nextTries = tries + 1
                setTries(nextTries)
                if (option.id === answerId || nextTries >= 2) onChange(option.id)
              }}
            >
              <Face label={option.label} picture={option.picture} state={showRight ? 'right' : showWrong ? 'wrong' : undefined} />
            </button>
          )
        })}
      </div>
    </Stage>
  )
}
```

Pass `answerId` and `nextTo` from `QuestionTypePreview` when `demo.mode === 'choice'`.

If `.kid-option.is-right` / `.is-wrong` and `.kid-feedback` do not exist, append:

```css
.kid-option.is-right { outline: 3px solid #2f9d5c; }
.kid-option.is-wrong { outline: 3px solid #d64545; }
.kid-feedback { min-height: 1.4em; font-size: 1.1rem; text-align: center; }
```

- [ ] **Step 4: Re-run tests**

Run: `cd kid-workbench/english-app && npm test -- src/App.test.tsx`

Expected: PASS for the two new/updated choice tests. Tile/type tests stay green.

- [ ] **Step 5: Commit only if asked**

```bash
git add kid-workbench/english-app/src/QuestionTypePreview.tsx kid-workbench/english-app/src/App.test.tsx kid-workbench/english-app/src/styles.css
git commit -m "feat(english-app): show instant choice feedback in type practice"
```

---

### Task 2: English runtime quiz generator

**Files:**
- Create: `kid-workbench/english-server/internal/quiz/service.go`
- Create: `kid-workbench/english-server/internal/quiz/service_test.go`

- [ ] **Step 1: Write failing generator tests**

```go
package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/english-server/internal/quiz"
)

func setup(t *testing.T) *quiz.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER)`,
		`CREATE TABLE english_assets (kp_id INTEGER PRIMARY KEY, sense_image_url TEXT, speech_audio_url TEXT)`,
		`INSERT INTO subjects VALUES (1,'english'),(2,'literacy')`,
		`INSERT INTO modules VALUES (1,1,'animals','Animals',0),(2,1,'greetings','Greetings',1),(3,2,'hanzi','汉字',0)`,
		`INSERT INTO knowledge_points VALUES
			(101,1,'apple','{"meaningZh":"苹果"}',1),
			(102,1,'banana','{"meaningZh":"香蕉"}',2),
			(103,1,'dog','{"meaningZh":"小狗"}',3),
			(104,1,'bird','{"meaningZh":"小鸟"}',4),
			(201,2,'hello','{"meaningZh":"你好"}',1),
			(202,2,'please','{"meaningZh":"请"}',2),
			(999,3,'山','{"meaningZh":"mountain"}',1)`,
		`INSERT INTO english_assets VALUES
			(101,'english/senses/101.png','english/speech/101.mp3'),
			(102,'english/senses/102.png','english/speech/102.mp3'),
			(103,'english/senses/103.png','english/speech/103.mp3'),
			(104,'','english/speech/104.mp3')`,
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	return quiz.NewService(db)
}

func TestGenerateListenUsesSameModuleAndSpeech(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	require.Equal(t, "listen", question.Type)
	require.Len(t, question.Options, 4)
	require.Equal(t, question.TargetID, question.Options[question.AnswerIndex].ID)
	require.Contains(t, []int64{101, 102, 103, 104}, question.TargetID)
	require.NotContains(t, []int64{201, 999}, question.TargetID)
	require.Contains(t, question.SpeechURL, "/api/v1/english/words/")
	require.Contains(t, question.SpeechURL, "/speech.mp3")
}

func TestGenerateLookSkipsAbstractGreetings(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "look", nil)
	require.NoError(t, err)
	require.Equal(t, "look", question.Type)
	require.Equal(t, "word", question.Visual.Kind)
	require.NotEmpty(t, question.Visual.Text)
	require.NotContains(t, []int64{201, 202}, question.TargetID)
}

func TestGenerateHonorsExcludeTargetIds(t *testing.T) {
	service := setup(t)
	first, err := service.Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "listen", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "blend", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}

func TestGenerateIgnoresOtherSubjects(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	for _, option := range question.Options {
		require.NotEqual(t, int64(999), option.ID)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/quiz -count=1`

Expected: FAIL because package `quiz` does not exist.

- [ ] **Step 3: Implement the generator**

Create `kid-workbench/english-server/internal/quiz/service.go`:

```go
package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrInvalidType = errors.New("无效的英语题型")
	ErrNoMaterial  = errors.New("没有足够的英语单词生成题目")
)

var abstractLook = map[string]struct{}{
	"hello": {}, "hi": {}, "please": {}, "sorry": {}, "thanks": {}, "thank": {},
	"goodbye": {}, "bye": {}, "yes": {}, "no": {}, "morning": {}, "noon": {},
	"evening": {}, "night": {}, "yesterday": {}, "today": {}, "tomorrow": {},
}

type Visual struct {
	Kind     string `json:"kind"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Option struct {
	ID       int64  `json:"id"`
	Label    string `json:"label,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

type Question struct {
	InstanceID  string   `json:"instanceId"`
	Type        string   `json:"type"`
	Stem        string   `json:"stem"`
	TargetID    int64    `json:"targetId"`
	SpeechText  string   `json:"speechText,omitempty"`
	SpeechURL   string   `json:"speechUrl,omitempty"`
	Visual      Visual   `json:"visual"`
	Options     []Option `json:"options"`
	AnswerIndex int      `json:"answerIndex"`
}

type wordRow struct {
	KpID      int64
	Word      string
	Payload   string
	Module    string
	HasSense  bool
	HasSpeech bool
}

type meaning struct {
	MeaningZh string `json:"meaningZh"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, quizType string, excluded []int64) (Question, error) {
	quizType = strings.TrimSpace(quizType)
	if quizType != "listen" && quizType != "look" {
		return Question{}, ErrInvalidType
	}
	rows, err := s.loadWords(ctx)
	if err != nil {
		return Question{}, err
	}
	candidates := make([]wordRow, 0, len(rows))
	for _, row := range rows {
		if meaningZh(row) == "" {
			continue
		}
		if quizType == "look" {
			if _, skip := abstractLook[strings.ToLower(row.Word)]; skip {
				continue
			}
		}
		if contains(excluded, row.KpID) {
			continue
		}
		candidates = append(candidates, row)
	}
	if len(candidates) == 0 {
		return Question{}, ErrNoMaterial
	}
	target := candidates[randIndex(len(candidates))]
	distractors := pickDistractors(target, rows, excluded)
	if len(distractors) < 3 {
		return Question{}, ErrNoMaterial
	}
	options := make([]Option, 0, 4)
	for _, row := range append([]wordRow{target}, distractors...) {
		option := Option{ID: row.KpID, Label: meaningZh(row)}
		if row.HasSense {
			option.ImageURL = fmt.Sprintf("/api/v1/english/words/%d/sense.png", row.KpID)
		}
		options = append(options, option)
	}
	shuffle(options)
	question := Question{
		InstanceID: newID(), Type: quizType, TargetID: target.KpID,
		Options: options, AnswerIndex: indexOf(options, target.KpID),
		SpeechText: target.Word,
	}
	if target.HasSpeech {
		question.SpeechURL = fmt.Sprintf("/api/v1/english/words/%d/speech.mp3", target.KpID)
	}
	if quizType == "listen" {
		question.Stem = "听一听，选出你听到的单词"
		question.Visual = Visual{Kind: "sound"}
	} else {
		question.Stem = "哪一张图是这个单词？"
		question.Visual = Visual{Kind: "word", Text: target.Word}
	}
	return question, nil
}

func (s *Service) loadWords(ctx context.Context) ([]wordRow, error) {
	var rows []wordRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT kp.id AS kp_id, kp.title AS word, kp.payload, m.code AS module,
			CASE WHEN COALESCE(ea.sense_image_url,'')<>'' THEN 1 ELSE 0 END AS has_sense,
			CASE WHEN COALESCE(ea.speech_audio_url,'')<>'' THEN 1 ELSE 0 END AS has_speech
		FROM knowledge_points kp
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN english_assets ea ON ea.kp_id = kp.id
		WHERE s.code = 'english'
		ORDER BY m.order_no, kp.order_no, kp.id`).Scan(&rows).Error
	return rows, err
}

func pickDistractors(target wordRow, all []wordRow, excluded []int64) []wordRow {
	same, other := []wordRow{}, []wordRow{}
	for _, row := range all {
		if row.KpID == target.KpID || contains(excluded, row.KpID) || strings.EqualFold(row.Word, target.Word) || meaningZh(row) == "" {
			continue
		}
		if row.Module == target.Module {
			same = append(same, row)
		} else {
			other = append(other, row)
		}
	}
	shuffle(same)
	shuffle(other)
	out := append(same, other...)
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

func meaningZh(row wordRow) string {
	var meta meaning
	_ = json.Unmarshal([]byte(row.Payload), &meta)
	return strings.TrimSpace(meta.MeaningZh)
}

func contains(ids []int64, id int64) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func indexOf(options []Option, id int64) int {
	for i, option := range options {
		if option.ID == id {
			return i
		}
	}
	return 0
}

func shuffle[T any](items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j := randIndex(i + 1)
		items[i], items[j] = items[j], items[i]
	}
}

func randIndex(n int) int {
	if n <= 1 {
		return 0
	}
	v, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

func newID() string {
	buf := make([]byte, 12)
	_, _ = cryptorand.Read(buf)
	return hex.EncodeToString(buf)
}
```

SQLite stores booleans as 0/1; keep `HasSense`/`HasSpeech` as `bool` so GORM scans them.

- [ ] **Step 4: Run generator tests**

Run: `cd kid-workbench/english-server && go test ./internal/quiz -count=1`

Expected: PASS

- [ ] **Step 5: Commit only if asked**

```bash
git add kid-workbench/english-server/internal/quiz
git commit -m "feat(english-server): generate listen and look quizzes from the word bank"
```

---

### Task 3: Expose quiz generate over HTTP

**Files:**
- Create: `kid-workbench/english-server/internal/http/handler_quiz.go`
- Modify: `kid-workbench/english-server/internal/http/router.go`
- Modify: `kid-workbench/english-server/internal/http/router_test.go`
- Modify: `kid-workbench/english-server/cmd/server/main.go`

- [ ] **Step 1: Add a failing route test**

Append to `router_test.go`:

```go
type stubQuiz struct{}

func (stubQuiz) Generate(_ context.Context, quizType string, _ []int64) (quiz.Question, error) {
	if quizType != "listen" {
		return quiz.Question{}, quiz.ErrInvalidType
	}
	return quiz.Question{InstanceID: "q1", Type: "listen", TargetID: 101, Options: []quiz.Option{{ID: 101}}, AnswerIndex: 0}, nil
}

func TestGenerateQuizRoute(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{Quiz: stubQuiz{}})
	body, err := json.Marshal(map[string]any{"type": "listen", "excludeTargetIds": []int64{1}})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	bad, err := json.Marshal(map[string]any{"type": "blend"})
	require.NoError(t, err)
	request = httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGenerateQuizUnavailable(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Deps{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/english/quiz/generate", bytes.NewReader([]byte(`{"type":"listen"}`)))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
```

Add imports: `bytes`, `context`, `encoding/json`, `"github.com/conchi/english-server/internal/quiz"`.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-server && go test ./internal/http -run TestGenerateQuiz -count=1`

Expected: FAIL because `Deps.Quiz` and the route do not exist.

- [ ] **Step 3: Add handler, route, and wiring**

`handler_quiz.go` copies pinyin’s handler, swapping error codes and importing `github.com/conchi/english-server/internal/quiz`.

In `router.go` add `Quiz Quiz` to `Deps`. After the catalog block:

```go
if deps.Quiz != nil {
	router.POST("/api/v1/english/quiz/generate", generateQuiz(deps.Quiz))
} else {
	router.POST("/api/v1/english/quiz/generate", func(c *gin.Context) {
		writeError(c, stdhttp.StatusServiceUnavailable, "dependency_unavailable", "出题服务尚未就绪")
	})
}
```

Always register the path so a missing service returns JSON 503, not nginx HTML.

In `cmd/server/main.go` add `Quiz: quiz.NewService(database)` to `Deps`.

- [ ] **Step 4: Run HTTP and full server tests**

Run: `cd kid-workbench/english-server && go test ./... -count=1`

Expected: PASS

- [ ] **Step 5: Commit only if asked**

```bash
git add kid-workbench/english-server/internal/http kid-workbench/english-server/cmd/server/main.go
git commit -m "feat(english-server): expose english quiz generate endpoint"
```

---

### Task 4: Frontend quiz client and store

**Files:**
- Modify: `kid-workbench/english-app/src/api/client.ts`
- Modify: `kid-workbench/english-app/src/api/types.ts`
- Create: `kid-workbench/english-app/src/api/english.ts`
- Create: `kid-workbench/english-app/src/api/english.test.ts`
- Create: `kid-workbench/english-app/src/store/liveQuizStore.ts`

- [ ] **Step 1: Write client tests**

```ts
import { afterEach, expect, test, vi } from 'vitest'
import { api } from './client'
import { generateEnglishQuizSet } from './english'

afterEach(() => vi.unstubAllGlobals())

test('non-json responses become 无法识别的内容', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
  await expect(api('/api/v1/english/quiz/generate', { method: 'POST', body: '{}' })).rejects.toThrow('服务返回了无法识别的内容')
})

test('generateEnglishQuizSet asks for four unique targets', async () => {
  const seen: number[][] = []
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body))
    seen.push(body.excludeTargetIds ?? [])
    const n = seen.length
    return new Response(JSON.stringify({
      data: {
        instanceId: `q${n}`, type: body.type, stem: 'stem', targetId: n,
        speechText: 'apple', visual: { kind: 'sound' },
        options: [{ id: n, label: '苹果' }, { id: 10 + n, label: '香蕉' }, { id: 20 + n, label: '小狗' }, { id: 30 + n, label: '小鸟' }],
        answerIndex: 0,
      },
      error: null,
    }))
  }))
  const questions = await generateEnglishQuizSet('listen')
  expect(questions).toHaveLength(4)
  expect(seen[0]).toEqual([])
  expect(seen[3]).toEqual([1, 2, 3])
})
```

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- src/api/english.test.ts`

Expected: FAIL because `generateEnglishQuizSet` and the non-JSON error do not exist.

- [ ] **Step 3: Implement client, types, helper, store**

Replace `client.ts` with a parse-safe envelope (same behavior as pinyin-app `src/api/client.ts`), exporting `api.get` / `api.post` **or** keep the existing `api(path, init)` helper and wrap `response.json()` in try/catch throwing `APIError('invalid_response', '服务返回了无法识别的内容', status)`. Do not break `PracticePage`, which calls `api<Plan>(...)`.

Extend `types.ts` (pretty-print while touching it) with:

```ts
export type EnglishQuizType = 'listen' | 'look'
export type EnglishGeneratedQuiz = {
  instanceId: string
  type: EnglishQuizType
  stem: string
  targetId: number
  speechText?: string
  speechUrl?: string
  visual: { kind: string; text?: string; imageUrl?: string }
  options: { id: number; label?: string; imageUrl?: string }[]
  answerIndex: number
}
```

`english.ts`:

```ts
import { api } from './client'
import type { EnglishGeneratedQuiz, EnglishQuizType } from './types'

export const englishApi = {
  generateQuiz: (type: EnglishQuizType, excludeTargetIds: number[]) =>
    api<EnglishGeneratedQuiz>('/api/v1/english/quiz/generate', {
      method: 'POST',
      body: JSON.stringify({ type, excludeTargetIds }),
    }),
}

export async function generateEnglishQuizSet(type: EnglishQuizType, count = 4) {
  const questions: EnglishGeneratedQuiz[] = []
  const excludeTargetIds: number[] = []
  for (let i = 0; i < count; i++) {
    const question = await englishApi.generateQuiz(type, excludeTargetIds)
    questions.push(question)
    excludeTargetIds.push(question.targetId)
  }
  return questions
}
```

`liveQuizStore.ts` follows `pinyin-app/src/store/demoQuizStore.ts`, keyed by `'audio-choice' | 'image-text'`. Map `audio-choice → listen`, `image-text → look`. State shape:

```ts
{
  questionsByType: Partial<Record<'audio-choice' | 'image-text', EnglishGeneratedQuiz[]>>
  loadingType: 'audio-choice' | 'image-text' | null
  error: string
  fallback: Partial<Record<'audio-choice' | 'image-text', boolean>>
  ensure, invalidate, useFallback
}
```

On failure set `error` and leave `questionsByType[type]` empty. `useFallback(type)` sets `fallback[type]=true` and clears `error` so the page renders `questionDemos`.

- [ ] **Step 4: Run client tests**

Run: `cd kid-workbench/english-app && npm test -- src/api/english.test.ts`

Expected: PASS

- [ ] **Step 5: Commit only if asked**

```bash
git add kid-workbench/english-app/src/api kid-workbench/english-app/src/store/liveQuizStore.ts
git commit -m "feat(english-app): add live english quiz client"
```

---

### Task 5: Play live listen/look quizzes in the gallery

**Files:**
- Modify: `kid-workbench/english-app/src/App.tsx`
- Modify: `kid-workbench/english-app/src/QuestionTypePreview.tsx`
- Modify: `kid-workbench/english-app/src/App.test.tsx`
- Modify: `kid-workbench/english-app/src/questionDemos.ts` (adapter helper only if needed)

- [ ] **Step 1: Add failing live-quiz tests**

```tsx
test('audio-choice loads four server questions', async () => {
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => new Response(JSON.stringify({
    data: {
      instanceId: 'q1', type: 'listen', stem: '听一听，选出你听到的单词', targetId: 101,
      speechText: 'cat', visual: { kind: 'sound' },
      options: [
        { id: 101, label: '小猫' }, { id: 102, label: '小鱼' },
        { id: 103, label: '太阳' }, { id: 104, label: '书本' },
      ],
      answerIndex: 0,
    },
    error: null,
  }))))
  render(<MemoryRouter initialEntries={['/types/audio-choice']}><App /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: '小猫' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '苹果' })).not.toBeInTheDocument()
})

test('audio-choice can fall back to the local demo bank', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))
  render(<MemoryRouter initialEntries={['/types/audio-choice']}><App /></MemoryRouter>)
  expect(await screen.findByText('服务返回了无法识别的内容')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '用示例题' }))
  expect(await screen.findByRole('button', { name: '苹果' })).toBeInTheDocument()
})
```

Reset `liveQuizStore` in `afterEach`.

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/english-app && npm test -- src/App.test.tsx`

Expected: FAIL because `/types/audio-choice` still always shows the apple demo.

- [ ] **Step 3: Render live questions through the existing preview**

Add a helper that turns `EnglishGeneratedQuiz` into `ChoiceDemo`:

```ts
export function quizToChoiceDemo(question: EnglishGeneratedQuiz): ChoiceDemo {
  return {
    mode: 'choice',
    prompt: question.visual.kind === 'word' ? question.stem : undefined,
    speech: question.speechText,
    options: question.options.map((option) => ({
      id: String(option.id),
      label: option.label ?? '',
      picture: option.imageUrl,
    })),
    answerId: String(question.options[question.answerIndex]?.id ?? ''),
  }
}
```

In `QuestionTypePreviewPage`:

- If `id` is `audio-choice` or `image-text`, `ensure(id)` on mount.
- While loading, show `出题中…`.
- On error, show the error text and `用示例题` (sets `fallback[id]=true` and keeps local `questionDemos`).
- On success, use `quizToChoiceDemo(questions[current-1])`.
- Play `speechUrl` when present; otherwise keep `speak(speechText)`.
- For `look`, show `visual.text` as a large English prompt and do not auto-play.

Pass `speechUrl` into `ListenButton` by extending `ChoicePlay` with `speechUrl?: string` and:

```ts
function playClip(url?: string, text?: string) {
  if (url) {
    const audio = new Audio(url)
    void audio.play().catch(() => speak(text ?? ''))
    return
  }
  if (text) speak(text)
}
```

Do not send `fetch` for `card-builder` / `input-gap` / `reading-qa`.

Clear the live store when the result page clicks 再练一次 so the next round excludes nothing and fetches again.

- [ ] **Step 4: Run frontend tests and build**

Run: `cd kid-workbench/english-app && npm test && npm run build`

Expected: PASS, production build succeeds.

- [ ] **Step 5: Commit only if asked**

```bash
git add kid-workbench/english-app/src
git commit -m "feat(english-app): play live listen and look quizzes in the type gallery"
```

---

### Task 6: Rebuild server image if needed and verify in the browser

**Files:** none new. Runtime: `http://localhost:19132` and `http://localhost:19131`.

- [ ] **Step 1: Confirm quiz generate on the running API**

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"type":"listen","excludeTargetIds":[]}' \
  http://127.0.0.1:19131/api/v1/english/quiz/generate
```

Expected: JSON `{data:{type:"listen", options:[...4], answerIndex:n}, error:null}`.

If this 404s, the Docker `english-server` image is stale. Rebuild only that service:

```bash
cd kid-workbench && docker compose up -d --build english-server
```

Then repeat the curl.

- [ ] **Step 2: Confirm the child app proxies generate**

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"type":"look","excludeTargetIds":[]}' \
  http://localhost:19132/api/v1/english/quiz/generate
```

Expected: JSON, not nginx HTML 502. If 502, the `english-app` container cannot reach `english-server:19131`; fix compose networking before more UI work. Local Vite on `19132` should proxy to `19131` via `vite.config.ts`.

- [ ] **Step 3: Browser-pass the five cards on `http://localhost:19132`**

Viewport 1024×768. For each card: open, answer Q1, confirm instant feedback, finish 4, see result, 再练一次, 回到首页.

| Card | Must see |
| --- | --- |
| 听音选词 | Play button; options from the word bank (not always 苹果/香蕉/小狗/小鸟); 答对了 / 再试一次 |
| 看图选词 | Large English word; four meaning/picture options |
| 组句子 | Tile sentence still works |
| 写单词 | Input still works |
| 读一读 | Passage + choices still works |

Stop `english-server` briefly, open 听音选词, click 用示例题, confirm apple demo still plays, then start the server again.

- [ ] **Step 4: If a card fails, fix and re-verify that card plus home**

Do not stop after the first green path. Home must still show five cards after the live-quiz changes.

- [ ] **Step 5: Commit only if asked**

No source change expected. If docker-compose needed a network fix, commit that file with message `fix(english-app): proxy quiz generate to english-server`.

---

## Self-review

**Spec coverage:** Instant feedback → Task 1. Generator rules / abstract words / subject scope / exclude IDs → Task 2. HTTP endpoint → Task 3. Client + four unique targets + non-JSON error → Task 4. Gallery wiring and fallback → Task 5. Browser five-card pass → Task 6. Plan API left untouched as specified.

**Placeholders:** None. Commands and types are named.

**Type consistency:** API `listen`/`look`; frontend cards `audio-choice`/`image-text`; option `id` is `kpId`; `answerIndex` points at that option.
