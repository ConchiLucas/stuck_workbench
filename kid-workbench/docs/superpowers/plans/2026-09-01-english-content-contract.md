# English Content Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every English knowledge point reviewed Chinese meaning metadata, store stable knowledge-point references in English question options, and track `listen` and `picture` as separate mastery skills.

**Architecture:** Keep course semantics in `knowledge_points.payload`, generated media in `english_assets`, and stable option references in `questions.options`. Extend the shared mastery mapping with subject-aware question-code resolution because `listen` is used by both pinyin and English.

**Tech Stack:** Go 1.26, GORM, PostgreSQL, SQLite tests, JSON

**Prerequisite:** The current `shared-go` extraction and migrations `006_plan_subject` and `007_plan_option_order` must be completed and committed before executing this plan.

---

### Task 1: Lock the English Metadata Contract

**Files:**
- Create: `kid-workbench/parent-dashboard/backend/internal/seed/catalog_english.go`
- Create: `kid-workbench/parent-dashboard/backend/internal/seed/catalog_english_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/catalog.go`

- [ ] **Step 1: Write the failing metadata coverage test**

```go
func TestEnglishCatalogHasReviewedMeaningForEveryWord(t *testing.T) {
    modules := englishModules()
    require.Len(t, modules, 20)
    count := 0
    for _, module := range modules {
        require.Len(t, module.Kps, 10, module.Code)
        for _, kp := range module.Kps {
            var payload englishPayload
            require.NoError(t, json.Unmarshal([]byte(kp.Payload), &payload), kp.Title)
            require.NotEmpty(t, payload.MeaningZh, kp.Title)
            count++
        }
    }
    require.Equal(t, 200, count)
}
```

- [ ] **Step 2: Run the test to verify failure**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/seed -run TestEnglishCatalogHasReviewedMeaningForEveryWord -count=1`

Expected: FAIL because `englishPayload` and reviewed payloads do not exist.

- [ ] **Step 3: Create the focused English catalog file**

Define:

```go
type englishPayload struct {
    MeaningZh       string `json:"meaningZh"`
    Phonetic        string `json:"phonetic,omitempty"`
    PartOfSpeech    string `json:"partOfSpeech,omitempty"`
    Example         string `json:"example,omitempty"`
    ExampleMeaningZh string `json:"exampleMeaningZh,omitempty"`
}

type englishWordSpec struct {
    Word string
    MeaningZh string
}
```

Move `englishModules` from `catalog.go` into this file. Build each `kpSpec.Payload` with `json.Marshal(englishPayload{MeaningZh: word.MeaningZh})`. Keep existing module codes, names, word order, KP codes and difficulty calculation unchanged.

Use these reviewed first-version meanings:

```text
animals: cat 猫, dog 狗, bird 鸟, fish 鱼, rabbit 兔子, tiger 老虎, lion 狮子, bear 熊, monkey 猴子, panda 熊猫
colors: red 红色, blue 蓝色, green 绿色, yellow 黄色, pink 粉色, black 黑色, white 白色, orange 橙色, purple 紫色, brown 棕色
numbers: one 一, two 二, three 三, four 四, five 五, six 六, seven 七, eight 八, nine 九, ten 十
food: apple 苹果, banana 香蕉, bread 面包, milk 牛奶, egg 鸡蛋, rice 米饭, cake 蛋糕, juice 果汁, soup 汤, candy 糖果
family: mom 妈妈, dad 爸爸, baby 婴儿, grandpa 爷爷, grandma 奶奶, brother 兄弟, sister 姐妹, uncle 叔叔, aunt 阿姨, cousin 堂（表）兄弟姐妹
body: head 头, eye 眼睛, ear 耳朵, nose 鼻子, mouth 嘴巴, hand 手, foot 脚, arm 手臂, leg 腿, hair 头发
fruits: grape 葡萄, peach 桃子, pear 梨, orange 橙子, lemon 柠檬, melon 瓜, cherry 樱桃, mango 芒果, kiwi 猕猴桃, berry 浆果
weather: sunny 晴朗的, rainy 下雨的, cloudy 多云的, windy 有风的, snowy 下雪的, hot 热的, cold 冷的, warm 温暖的, cool 凉爽的, storm 暴风雨
toys: ball 球, doll 玩偶, car 小汽车, block 积木, kite 风筝, train 火车, puzzle 拼图, robot 机器人, balloon 气球, slide 滑梯
school: book 书, pen 钢笔, pencil 铅笔, bag 书包, desk 课桌, chair 椅子, teacher 老师, student 学生, class 班级, school 学校
clothes: shirt 衬衫, pants 裤子, dress 连衣裙, hat 帽子, shoe 鞋, sock 袜子, coat 外套, scarf 围巾, glove 手套, skirt 裙子
actions: run 跑, jump 跳, walk 走, sit 坐, stand 站, eat 吃, drink 喝, sleep 睡觉, read 阅读, write 写
places: home 家, park 公园, zoo 动物园, shop 商店, farm 农场, beach 海滩, library 图书馆, museum 博物馆, hospital 医院, cinema 电影院
transport: bus 公交车, bike 自行车, plane 飞机, boat 小船, train 火车, taxi 出租车, truck 卡车, subway 地铁, helicopter 直升机, ship 轮船
shapes: circle 圆形, square 正方形, triangle 三角形, star 星形, heart 心形, oval 椭圆形, rectangle 长方形, diamond 菱形, cross 十字形, arrow 箭头
time: morning 早晨, noon 中午, afternoon 下午, evening 傍晚, night 夜晚, today 今天, yesterday 昨天, tomorrow 明天, week 星期, year 年
feelings: happy 开心的, sad 难过的, angry 生气的, tired 疲倦的, hungry 饥饿的, thirsty 口渴的, scared 害怕的, brave 勇敢的, kind 友善的, funny 有趣的
nature: tree 树, flower 花, grass 草, river 河流, mountain 山, sun 太阳, moon 月亮, star 星星, cloud 云, rain 雨
jobs: doctor 医生, nurse 护士, chef 厨师, pilot 飞行员, driver 司机, farmer 农民, singer 歌手, dancer 舞者, police 警察, firefighter 消防员
greetings: hello 你好, hi 嗨, bye 再见, thanks 谢谢, please 请, sorry 对不起, yes 是, no 不, ok 好的, welcome 欢迎
```

- [ ] **Step 4: Run seed tests**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/seed -run 'TestEnglishCatalog|TestCatalog' -count=1`

Expected: PASS with 200 English knowledge points and unchanged codes/order.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/internal/seed/catalog.go kid-workbench/parent-dashboard/backend/internal/seed/catalog_english.go kid-workbench/parent-dashboard/backend/internal/seed/catalog_english_test.go
git commit -m "feat: add reviewed english meanings"
```

### Task 2: Add Stable English Option References

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/quiz.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/english.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/quiz/quiz_test.go`
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/questions.go`

- [ ] **Step 1: Write failing English option-contract tests**

Update the English test fixture with sibling IDs and assert:

```go
require.Equal(t, kp.ID, listen.Options[listen.AnswerIndex].KpID)
require.Equal(t, "cat", listen.Options[listen.AnswerIndex].Label)
require.Equal(t, kp.ID, picture.Options[picture.AnswerIndex].KpID)
require.Equal(t, "sense", picture.Options[picture.AnswerIndex].AssetKind)
require.Empty(t, picture.Options[picture.AnswerIndex].Emoji)
```

Also assert all four English options have distinct, non-zero `KpID` values.

- [ ] **Step 2: Run the focused test to verify failure**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz -run TestEnglish -count=1`

Expected: FAIL because `Option.KpID`, `Option.AssetKind` and sibling IDs are undefined.

- [ ] **Step 3: Extend only the generic transport structs**

```go
type Kp struct {
    ID int64
    Title string
    Payload string
    Difficulty int
    SubjectCode string
    ModuleCode string
    Siblings []string
    SiblingIDs map[string]int64
}

type Option struct {
    KpID int64 `json:"kpId,omitempty"`
    Label string `json:"label,omitempty"`
    Emoji string `json:"emoji,omitempty"`
    Shape string `json:"shape,omitempty"`
    AssetKind string `json:"assetKind,omitempty"`
}
```

Do not change non-English option generation.

- [ ] **Step 4: Pass module title-to-ID maps from the seed**

Build `siblingIDs map[int64]map[string]int64` keyed by module ID while loading rows. Pass the module map into `quiz.Kp.SiblingIDs` for every generated question.

- [ ] **Step 5: Generate referenced English options**

Replace English `labelOptions` use with a focused helper that produces `{KpID, Label}`. Replace picture emoji options with `{KpID, AssetKind: "sense"}` while retaining the existing ambiguity allow-list to decide which words may participate.

- [ ] **Step 6: Run all quiz and seed tests**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/quiz ./internal/seed -count=1`

Expected: PASS; non-English serialized option fixtures remain unchanged.

- [ ] **Step 7: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/internal/quiz kid-workbench/parent-dashboard/backend/internal/seed/questions.go
git commit -m "feat: add stable english question references"
```

### Task 3: Add Subject-Aware English Skills

**Files:**
- Modify: `kid-workbench/shared-go/mastery/skills.go`
- Modify: `kid-workbench/shared-go/mastery/skills_test.go`
- Modify: `kid-workbench/shared-go/learning/service.go`
- Modify: `kid-workbench/shared-go/learning/service_test.go`

- [ ] **Step 1: Write failing skill mapping tests**

```go
func TestEnglishSkills(t *testing.T) {
    require.Equal(t, []string{"listen", "picture"}, SkillsForSubject("english"))
    require.Equal(t, "listen", SkillFromQuestionCode("english", "listen"))
    require.Equal(t, "picture", SkillFromQuestionCode("english", "picture"))
    require.Equal(t, "listen", SkillFromQuestionCode("pinyin", "listen"))
    require.Empty(t, SkillFromQuestionCode("literacy", "listen"))
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd kid-workbench/shared-go && go test ./mastery ./learning -run 'TestEnglishSkills|TestApplyOneAggregatesEnglishSkills' -count=1`

Expected: FAIL because English has no skill set and the resolver is not subject-aware.

- [ ] **Step 3: Make question-code mapping subject-aware**

Define `EnglishSkills = []string{"listen", "picture"}` and change the resolver to:

```go
func SkillFromQuestionCode(subjectCode, questionCode string) string
```

Resolve codes inside a subject switch so the shared `listen` code remains valid for both pinyin and English without becoming valid for literacy.

- [ ] **Step 4: Update learning service resolution order**

In `ApplyOne`, load the subject before mapping the question code, then call:

```go
skillCode = mastery.SkillFromQuestionCode(subjectCode, question.Code)
skillSet := mastery.SkillsForSubject(subjectCode)
```

Keep the rest of the transaction and roll-up behavior unchanged.

- [ ] **Step 5: Add English roll-up coverage**

Seed one English KP with `listen` and `picture` questions. Apply the configured `BaseMasterStreak` number of correct attempts to each skill, using a unique `clientId` for every attempt. Assert two mastered `mastery_skills` rows plus one mastered rolled-up `mastery_states` row.

- [ ] **Step 6: Run shared and parent regression suites**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Expected: both PASS.

- [ ] **Step 7: Commit**

```bash
git add kid-workbench/shared-go
git commit -m "feat: track english mastery skills"
```

### Task 4: Verify Persisted English Contracts

**Files:**
- Modify: `kid-workbench/parent-dashboard/backend/internal/seed/questions_test.go`
- Modify: `kid-workbench/parent-dashboard/README.md`

- [ ] **Step 1: Add an end-to-end seed assertion**

Seed catalog and questions in SQLite. Query all English questions and assert every `listen` option has non-zero `kpId` plus `label`, every `picture` option has non-zero `kpId` plus `assetKind=sense`, and no question exposes a correct answer inside its options.

- [ ] **Step 2: Run the persistence test**

Run: `cd kid-workbench/parent-dashboard/backend && go test ./internal/seed -run 'TestStoredEnglishQuestionContract|TestStoredQuestionsAreAnswerable' -count=1`

Expected: PASS.

- [ ] **Step 3: Document the English contract**

Add a short README subsection naming `meaningZh`, optional metadata fields, English skills, and the `{kpId,label}` / `{kpId,assetKind}` option contracts.

- [ ] **Step 4: Run final checks**

Run: `cd kid-workbench/shared-go && go test ./... -count=1`

Run: `cd kid-workbench/parent-dashboard/backend && go test ./... -count=1`

Expected: both PASS.

- [ ] **Step 5: Commit**

```bash
git add kid-workbench/parent-dashboard/backend/internal/seed/questions_test.go kid-workbench/parent-dashboard/README.md
git commit -m "test: verify english content contract"
```
