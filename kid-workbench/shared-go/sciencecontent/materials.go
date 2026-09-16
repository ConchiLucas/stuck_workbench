package sciencecontent

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed habitats/*.png
var habitatFiles embed.FS

const ObserveModule = "observe"

type MaterialSpec struct {
	Code        string
	Title       string
	Kind        string
	Prompt      string
	Explanation string
	Tip         string
	Rationale   string
	Citation    string
	Payload     map[string]any
	Question    storedQuestion
}

type storedQuestion struct {
	Code    string
	Stem    string
	Options any
	Answer  any
	Visual  any
}

func MaterialSpecs() []MaterialSpec {
	plantTargets := []LabelTarget{
		{ID: "flower", Label: "花", X: 0.78, Y: 0.14},
		{ID: "leaf", Label: "叶", X: 0.18, Y: 0.48},
		{ID: "root", Label: "根", X: 0.50, Y: 0.90},
	}
	return []MaterialSpec{
		mediaSpec("sc-hab-rainforest", "热带雨林", "rainforest"),
		mediaSpec("sc-hab-ice", "冰原", "ice"),
		mediaSpec("sc-hab-desert", "沙漠", "desert"),
		{
			Code: "sc-choice-duck", Title: "鸭子的脚掌", Kind: KindChoice,
			Prompt:      "哪种动物的脚掌最适合在水里游泳？",
			Explanation: "鸭子的脚趾之间有蹼，像小扇子，划水更有力。",
			Tip:         "先看脚趾之间，有没有像小扇子一样连起来。",
			Rationale:   "蹼足增加划水面积。本题问最适合在水里游泳的脚掌，鸭子成立，猫和兔子没有蹼。",
			Citation:    "小学科学观察：水禽脚趾间有蹼。",
			Question: storedQuestion{Code: KindChoice, Stem: "哪种动物的脚掌最适合在水里游泳？",
				Options: []Choice{{ID: "cat", Label: "猫", Icon: "cat"}, {ID: "duck", Label: "鸭子", Icon: "duck"}, {ID: "rabbit", Label: "兔子", Icon: "rabbit"}},
				Answer:  map[string]any{"id": "duck"}, Visual: map[string]any{"kind": "icon", "key": "duck"}},
		},
		{
			Code: "sc-choice-sun", Title: "植物需要阳光", Kind: KindChoice,
			Prompt:      "在大自然里，绿叶制造养分主要靠哪一种光？",
			Explanation: "绿叶靠太阳光制造养分，植物才能慢慢长高。",
			Tip:         "想想叶子白天总是朝着哪里。",
			Rationale:   "光合作用的能量来自太阳辐射。题干限定「在大自然里」，不把教室灯光或萤火虫当作成立答案。",
			Citation:    "光合作用利用可见光，自然界主要来源是太阳。",
			Question: storedQuestion{Code: KindChoice, Stem: "在大自然里，绿叶制造养分主要靠哪一种光？",
				Options: []Choice{{ID: "lamp", Label: "灯光", Icon: "lamp"}, {ID: "sun", Label: "太阳", Icon: "sun"}, {ID: "firefly", Label: "萤火虫", Icon: "firefly"}},
				Answer:  map[string]any{"id": "sun"}, Visual: map[string]any{"kind": "icon", "key": "sun"}},
		},
		{
			Code: "sc-choice-ice", Title: "冰遇热融化", Kind: KindChoice,
			Prompt:      "冰变成水，通常是因为什么？",
			Explanation: "冰吸收热量后会融化成水。",
			Tip:         "把冰块放在手上，会慢慢变成什么？",
			Rationale:   "熔化需要吸收热量。变冷、变重都不是冰变水的原因。",
			Citation:    "物质三态：固态吸热熔化为液态。",
			Question: storedQuestion{Code: KindChoice, Stem: "冰变成水，通常是因为什么？",
				Options: []Choice{{ID: "heat", Label: "变热了", Icon: "heat"}, {ID: "cold", Label: "变冷了", Icon: "cold"}, {ID: "heavy", Label: "变重了", Icon: "heavy"}},
				Answer:  map[string]any{"id": "heat"}, Visual: map[string]any{"kind": "icon", "key": "ice"}},
		},
		{
			Code: "sc-choice-gills", Title: "鱼用鳃呼吸", Kind: KindChoice,
			Prompt:      "哪种动物用鳃在水里呼吸？",
			Explanation: "金鱼用鳃在水里交换气体。",
			Tip:         "看看谁一直生活在水里。",
			Rationale:   "鱼类用鳃从水中获取氧气。猫和兔子用肺在空气中呼吸。",
			Citation:    "鱼类用鳃呼吸。",
			Question: storedQuestion{Code: KindChoice, Stem: "哪种动物用鳃在水里呼吸？",
				Options: []Choice{{ID: "cat", Label: "猫", Icon: "cat"}, {ID: "goldfish", Label: "金鱼", Icon: "fish"}, {ID: "rabbit", Label: "兔子", Icon: "rabbit"}},
				Answer:  map[string]any{"id": "goldfish"}, Visual: map[string]any{"kind": "icon", "key": "fish"}},
		},
		{
			Code: "sc-match-homes", Title: "动物和生活环境", Kind: KindMatch,
			Prompt:      "把下面三种动物和它们最常见的生活环境连起来。",
			Explanation: "青蛙常见于潮湿的树林和水边，北极熊生活在冰原，骆驼适应沙漠。",
			Tip:         "看看皮毛、皮肤和脚掌，再想它们适合哪里。",
			Rationale:   "题干限定给定三种动物的最常见环境，做一一对应。青蛙也可生活在池塘等湿地，本题不把复杂栖息地网络做成多对多。",
			Citation:    "北极熊依赖海冰与冰原；骆驼是沙漠动物；树蛙常见于热带潮湿森林。",
			Question: storedQuestion{Code: KindMatch, Stem: "把下面三种动物和它们最常见的生活环境连起来。",
				Options: map[string]any{
					"sourceGroup": "动物", "targetGroup": "环境",
					"sources": []Node{{ID: "frog", Label: "青蛙", Icon: "frog"}, {ID: "bear", Label: "北极熊", Icon: "bear"}, {ID: "camel", Label: "骆驼", Icon: "camel"}},
					"targets": []Node{{ID: "forest", Label: "热带雨林", Icon: "rainforest"}, {ID: "ice", Label: "冰原", Icon: "ice"}, {ID: "desert", Label: "沙漠", Icon: "desert"}},
					"answers": map[string]string{"frog": "forest", "bear": "ice", "camel": "desert"},
				},
				Answer: map[string]any{"pairs": map[string]string{"frog": "forest", "bear": "ice", "camel": "desert"}},
				Visual: map[string]any{"kind": "match"}},
		},
		{
			Code: "sc-match-organs", Title: "器官和本领", Kind: KindMatch,
			Prompt:      "把器官和它们的本领连在一起。",
			Explanation: "眼睛用来看，耳朵用来听，鼻子用来闻。",
			Tip:         "先想这个器官帮我们做什么。",
			Rationale:   "小学观察课把眼、耳、鼻与看、听、闻一一对应。不把大脑综合感觉做成一对多。",
			Citation:    "感觉器官的基本功能。",
			Question: storedQuestion{Code: KindMatch, Stem: "把器官和它们的本领连在一起。",
				Options: map[string]any{
					"sourceGroup": "器官", "targetGroup": "本领",
					"sources": []Node{{ID: "ear", Label: "耳朵", Icon: "ear"}, {ID: "eye", Label: "眼睛", Icon: "eye"}, {ID: "nose", Label: "鼻子", Icon: "nose"}},
					"targets": []Node{{ID: "see", Label: "看", Icon: "see"}, {ID: "hear", Label: "听", Icon: "hear"}, {ID: "smell", Label: "闻", Icon: "smell"}},
					"answers": map[string]string{"eye": "see", "ear": "hear", "nose": "smell"},
				},
				Answer: map[string]any{"pairs": map[string]string{"eye": "see", "ear": "hear", "nose": "smell"}},
				Visual: map[string]any{"kind": "match"}},
		},
		{
			Code: "sc-match-food", Title: "动物爱吃的东西", Kind: KindMatch,
			Prompt:      "把下面三种动物和它们最爱吃的食物连起来。",
			Explanation: "熊猫主要吃竹子，蜜蜂采花蜜，长颈鹿吃树叶。",
			Tip:         "想想它们平时用什么喂饱自己。",
			Rationale:   "题干限定「最爱吃」。大熊猫野外食物几乎全是竹子；蜜蜂采集花蜜；长颈鹿取食高处树叶。不把完整食物网做成强制 1:1。",
			Citation:    "大熊猫以竹为主食；蜜蜂采集花蜜；长颈鹿是典型的食叶者。",
			Question: storedQuestion{Code: KindMatch, Stem: "把下面三种动物和它们最爱吃的食物连起来。",
				Options: map[string]any{
					"sourceGroup": "动物", "targetGroup": "食物",
					"sources": []Node{{ID: "bee", Label: "蜜蜂", Icon: "bee"}, {ID: "panda", Label: "熊猫", Icon: "panda"}, {ID: "giraffe", Label: "长颈鹿", Icon: "giraffe"}},
					"targets": []Node{{ID: "bamboo", Label: "竹子", Icon: "bamboo"}, {ID: "nectar", Label: "花蜜", Icon: "nectar"}, {ID: "leaves", Label: "树叶", Icon: "leaf"}},
					"answers": map[string]string{"panda": "bamboo", "bee": "nectar", "giraffe": "leaves"},
				},
				Answer: map[string]any{"pairs": map[string]string{"panda": "bamboo", "bee": "nectar", "giraffe": "leaves"}},
				Visual: map[string]any{"kind": "match"}},
		},
		{
			Code: "sc-seq-plant", Title: "植物生长顺序", Kind: KindSequence,
			Prompt:      "把植物从种子长成幼苗的过程排成正确顺序。",
			Explanation: "种子吸水后先发芽，再长成幼苗。起点是种子。",
			Tip:         "先找到还没发芽的那一步。",
			Rationale:   "种子萌发：吸水→胚根/胚芽伸出（发芽）→长成幼苗。判定以种子为起点的唯一顺序。",
			Citation:    "种子萌发过程。",
			Question: storedQuestion{Code: KindSequence, Stem: "把植物从种子长成幼苗的过程排成正确顺序。",
				Options: map[string]any{"items": []Node{{ID: "sprout", Label: "发芽", Icon: "sprout"}, {ID: "seed", Label: "种子", Icon: "seed"}, {ID: "seedling", Label: "幼苗", Icon: "seedling"}}},
				Answer:  map[string]any{"order": []string{"seed", "sprout", "seedling"}, "startId": "seed", "loop": false},
				Visual:  map[string]any{"kind": "sequence"}},
		},
		{
			Code: "sc-seq-butterfly", Title: "蝴蝶生长顺序", Kind: KindSequence,
			Prompt:      "把蝴蝶从卵到成虫的主要阶段排好（本题不包含蛹）。",
			Explanation: "卵先变成毛毛虫，再变成蝴蝶。本题故意省略蛹，方便先建立先后关系。",
			Tip:         "先找到最小的起点。",
			Rationale:   "完全变态实际是卵→幼虫→蛹→成虫。题干写明不包含蛹，三步是教学简化，不是完整生活史。",
			Citation:    "鳞翅目完全变态；教学简化须标明省略蛹。",
			Question: storedQuestion{Code: KindSequence, Stem: "把蝴蝶从卵到成虫的主要阶段排好（本题不包含蛹）。",
				Options: map[string]any{"items": []Node{{ID: "caterpillar", Label: "毛毛虫", Icon: "caterpillar"}, {ID: "egg", Label: "卵", Icon: "egg"}, {ID: "butterfly", Label: "蝴蝶", Icon: "butterfly"}}},
				Answer:  map[string]any{"order": []string{"egg", "caterpillar", "butterfly"}, "startId": "egg", "loop": false},
				Visual:  map[string]any{"kind": "sequence"}},
		},
		{
			Code: "sc-seq-water", Title: "水的受热变化", Kind: KindSequence,
			Prompt:      "冰块受热之后会怎样变化？从固态开始排。",
			Explanation: "冰遇热先变成水，再变成水汽。这是加热路径，不是水循环的唯一起点。",
			Tip:         "先找到还是硬邦邦的那一步。",
			Rationale:   "加热：冰（固）→水（液）→水汽（气）。水循环可从任何一态开始，故题干指定从固态、受热。",
			Citation:    "水的三态变化：熔化与汽化需要吸热。",
			Question: storedQuestion{Code: KindSequence, Stem: "冰块受热之后会怎样变化？从固态开始排。",
				Options: map[string]any{"items": []Node{{ID: "water", Label: "水", Icon: "water"}, {ID: "ice", Label: "冰", Icon: "ice"}, {ID: "vapor", Label: "汽", Icon: "vapor"}}},
				Answer:  map[string]any{"order": []string{"ice", "water", "vapor"}, "startId": "ice", "loop": false},
				Visual:  map[string]any{"kind": "sequence"}},
		},
		labelSpec("sc-label-root", "标注根", "把“根”标到植物的正确位置。", "根藏在土壤下，负责吸收水分。", "看一看植物最下面、埋在土里的部分。", "root", "根", plantTargets),
		labelSpec("sc-label-leaf", "标注叶", "把“叶”标到植物的正确位置。", "叶子长在茎上，用来接收阳光。", "看看植物身体两边伸出来的绿色部分。", "leaf", "叶", plantTargets),
		labelSpec("sc-label-flower", "标注花", "把“花”标到植物的正确位置。", "花开在最上面，能结出种子。", "看看植物最高、最鲜艳的部分。", "flower", "花", plantTargets),
	}
}

func labelSpec(code, title, prompt, explanation, tip, tokenID, tokenLabel string, targets []LabelTarget) MaterialSpec {
	return MaterialSpec{
		Code: code, Title: title, Kind: KindLabel, Prompt: prompt, Explanation: explanation, Tip: tip,
		Rationale: "结构图部位与标签必须对应：根在地下、叶在茎侧、花在上部。坐标为相对图幅 0–1，随容器缩放。",
		Citation:  "被子植物营养器官与生殖器官的基本位置。",
		Question: storedQuestion{Code: KindLabel, Stem: prompt,
			Options: map[string]any{
				"diagramKey": DiagramPlant, "diagramVersion": DiagramVer,
				"targets": targets, "tokens": []Node{{ID: tokenID, Label: tokenLabel}},
			},
			Answer: map[string]any{"labels": map[string]string{tokenID: tokenID}},
			Visual: map[string]any{"kind": "diagram", "key": DiagramPlant, "version": DiagramVer}},
	}
}

func mediaSpec(code, title, habitat string) MaterialSpec {
	return MaterialSpec{
		Code: code, Title: title + "环境图", Kind: "media",
		Prompt: title, Explanation: "连线题使用的真实环境照片。",
		Rationale: "环境图必须支持题意，不能用无关装饰图。",
		Citation:  "照片仅表示该环境的典型外观。",
		Payload:   map[string]any{"kind": "media", "habitat": habitat},
	}
}

func HabitatFile(id string) ([]byte, error) {
	name := map[string]string{"forest": "rainforest", "rainforest": "rainforest", "ice": "ice", "desert": "desert"}[id]
	if name == "" {
		name = id
	}
	return habitatFiles.ReadFile("habitats/habitat-" + name + ".png")
}

func EnsureMaterials(db *gorm.DB) error {
	if err := MigrateMedia(db); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var subjectID int64
		if err := tx.Raw(`SELECT id FROM subjects WHERE code = ?`, "science").Scan(&subjectID).Error; err != nil {
			return err
		}
		if subjectID == 0 {
			return fmt.Errorf("science subject missing")
		}
		var moduleID int64
		if err := tx.Raw(`SELECT id FROM modules WHERE subject_id = ? AND code = ?`, subjectID, ObserveModule).Scan(&moduleID).Error; err != nil {
			return err
		}
		if moduleID == 0 {
			if err := tx.Exec(`INSERT INTO modules (subject_id, code, name, order_no) VALUES (?, ?, ?, ?)`, subjectID, ObserveModule, "观察与过程", 80).Error; err != nil {
				return err
			}
			if err := tx.Raw(`SELECT id FROM modules WHERE subject_id = ? AND code = ?`, subjectID, ObserveModule).Scan(&moduleID).Error; err != nil {
				return err
			}
		}
		specs := MaterialSpecs()
		for i, spec := range specs {
			payload := spec.Payload
			if payload == nil {
				payload = map[string]any{
					"kind": spec.Kind, "prompt": spec.Prompt, "explanation": spec.Explanation, "tip": spec.Tip,
					"rationale": spec.Rationale, "citation": spec.Citation,
				}
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			var kpID int64
			if err := tx.Raw(`SELECT id FROM knowledge_points WHERE module_id = ? AND code = ?`, moduleID, spec.Code).Scan(&kpID).Error; err != nil {
				return err
			}
			if kpID == 0 {
				if err := tx.Exec(`INSERT INTO knowledge_points (module_id, code, title, payload, difficulty, order_no) VALUES (?, ?, ?, ?, 1, ?)`,
					moduleID, spec.Code, spec.Title, string(raw), i+1).Error; err != nil {
					return err
				}
				if err := tx.Raw(`SELECT id FROM knowledge_points WHERE module_id = ? AND code = ?`, moduleID, spec.Code).Scan(&kpID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Exec(`UPDATE knowledge_points SET title = ?, payload = ?, order_no = ? WHERE id = ?`, spec.Title, string(raw), i+1, kpID).Error; err != nil {
					return err
				}
			}
			if err := upsertAsset(tx, kpID, spec, i+1); err != nil {
				return err
			}
			if spec.Kind == "media" {
				habitat := "ice"
				if spec.Payload != nil {
					if v, ok := spec.Payload["habitat"].(string); ok {
						habitat = v
					}
				}
				if err := storeHabitat(tx, kpID, habitat); err != nil {
					return err
				}
				continue
			}
			if err := upsertQuestion(tx, kpID, spec); err != nil {
				return err
			}
		}
		return attachMatchHabitatURLs(tx, moduleID)
	})
}

func upsertAsset(tx *gorm.DB, kpID int64, spec MaterialSpec, order int) error {
	now := time.Now().UTC()
	row := map[string]any{
		"kp_id": kpID, "title": spec.Title, "module_code": ObserveModule, "module_name": "观察与过程",
		"module_order": 80, "kp_order": order, "needs_sense_image": false, "review_status": "published",
		"summary": spec.Prompt, "explanation": spec.Explanation, "fun_fact": spec.Citation,
		"content_version": 1, "synced_at": now, "updated_at": now,
	}
	return tx.Table("science_assets").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kp_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "module_code", "module_name", "summary", "explanation", "fun_fact", "review_status", "updated_at"}),
	}).Create(row).Error
}

func upsertQuestion(tx *gorm.DB, kpID int64, spec MaterialSpec) error {
	opt, err := json.Marshal(spec.Question.Options)
	if err != nil {
		return err
	}
	ans, err := json.Marshal(spec.Question.Answer)
	if err != nil {
		return err
	}
	vis, err := json.Marshal(spec.Question.Visual)
	if err != nil {
		return err
	}
	var qid int64
	if err := tx.Raw(`SELECT id FROM questions WHERE kp_id = ? AND code = ?`, kpID, spec.Question.Code).Scan(&qid).Error; err != nil {
		return err
	}
	if qid == 0 {
		return tx.Exec(`INSERT INTO questions (kp_id, code, type, stem, options, answer, visual, difficulty) VALUES (?, ?, 'practice', ?, ?, ?, ?, 1)`,
			kpID, spec.Question.Code, spec.Question.Stem, string(opt), string(ans), string(vis)).Error
	}
	return tx.Exec(`UPDATE questions SET stem = ?, options = ?, answer = ?, visual = ? WHERE id = ?`,
		spec.Question.Stem, string(opt), string(ans), string(vis), qid).Error
}

func storeHabitat(tx *gorm.DB, kpID int64, habitat string) error {
	data, err := HabitatFile(habitat)
	if err != nil {
		return err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	row := ItemMedia{KpID: kpID, Kind: "diagram", SHA256: hash, MIME: "image/png", Data: data}
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func attachMatchHabitatURLs(tx *gorm.DB, moduleID int64) error {
	type kp struct {
		ID   int64
		Code string
	}
	var rows []kp
	if err := tx.Raw(`SELECT id, code FROM knowledge_points WHERE module_id = ?`, moduleID).Scan(&rows).Error; err != nil {
		return err
	}
	byCode := map[string]int64{}
	for _, r := range rows {
		byCode[r.Code] = r.ID
	}
	habitat := map[string]int64{
		"forest": byCode["sc-hab-rainforest"],
		"ice":    byCode["sc-hab-ice"],
		"desert": byCode["sc-hab-desert"],
	}
	var match struct {
		ID      int64
		Options string
	}
	if err := tx.Raw(`SELECT q.id, q.options FROM questions q JOIN knowledge_points kp ON kp.id = q.kp_id WHERE kp.code = ?`, "sc-match-homes").Scan(&match).Error; err != nil {
		return err
	}
	if match.ID == 0 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal([]byte(match.Options), &body) != nil {
		return nil
	}
	targets, _ := body["targets"].([]any)
	for i, raw := range targets {
		t, _ := raw.(map[string]any)
		id, _ := t["id"].(string)
		if kpID := habitat[id]; kpID > 0 {
			t["imageUrl"] = fmt.Sprintf("/api/v1/science/items/%d/diagram.png", kpID)
			targets[i] = t
		}
	}
	body["targets"] = targets
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return tx.Exec(`UPDATE questions SET options = ? WHERE id = ?`, string(raw), match.ID).Error
}

func HabitatKpIDs(db *gorm.DB) (map[string]int64, error) {
	type row struct {
		ID   int64
		Code string
	}
	var rows []row
	if err := db.Raw(`SELECT kp.id, kp.code FROM knowledge_points kp JOIN modules m ON m.id = kp.module_id JOIN subjects s ON s.id = m.subject_id WHERE s.code = 'science' AND kp.code LIKE 'sc-hab-%'`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		switch r.Code {
		case "sc-hab-rainforest":
			out["forest"] = r.ID
		case "sc-hab-ice":
			out["ice"] = r.ID
		case "sc-hab-desert":
			out["desert"] = r.ID
		}
	}
	return out, nil
}

func ContentHash(spec MaterialSpec) string {
	raw, _ := json.Marshal(spec.Question)
	return fmt.Sprintf("%x", sha256.Sum256(append([]byte(spec.Code), raw...)))[:16]
}

func SourceHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", sum[:])[:16]
}

func QuestionCodes() []string {
	return []string{KindChoice, KindMatch, KindSequence, KindLabel}
}

func InteractionSpecs() []MaterialSpec {
	var out []MaterialSpec
	for _, spec := range MaterialSpecs() {
		if spec.Kind != "media" {
			out = append(out, spec)
		}
	}
	return out
}

func FilterKind(kind string) []MaterialSpec {
	var out []MaterialSpec
	for _, spec := range InteractionSpecs() {
		if spec.Kind == kind {
			out = append(out, spec)
		}
	}
	return out
}

func ExampleFromSpec(spec MaterialSpec, habitatByCode map[string]int64) (ScienceExample, error) {
	opt, _ := json.Marshal(spec.Question.Options)
	ans, _ := json.Marshal(spec.Question.Answer)
	vis, _ := json.Marshal(spec.Question.Visual)
	payload, _ := json.Marshal(map[string]any{
		"explanation": spec.Explanation, "tip": spec.Tip, "rationale": spec.Rationale,
	})
	return exampleFromStored(spec.Kind, LiveQuestion{
		Code: spec.Kind, Stem: spec.Question.Stem, Options: string(opt), Answer: string(ans),
		Visual: string(vis), Payload: string(payload), HabitatByCode: habitatByCode,
	})
}

func ShuffleExample(e ScienceExample, seed int64) ScienceExample {
	switch KindForSkill(e.Kind) {
	case KindChoice:
		ids := make([]string, len(e.Options))
		for i, o := range e.Options {
			ids[i] = o.ID
		}
		order := ShuffleIDs(ids, seed)
		byID := map[string]Choice{}
		for _, o := range e.Options {
			byID[o.ID] = o
		}
		next := make([]Choice, 0, len(order))
		for _, id := range order {
			next = append(next, byID[id])
		}
		e.Options, e.OptionOrder = next, order
	case KindMatch:
		e.MatchSourceOrder = ShuffleIDs(nodeIDs(e.MatchSources), seed)
		e.MatchTargetOrder = ShuffleIDs(nodeIDs(e.MatchTargets), seed+17)
		e.MatchSources = reorderNodes(e.MatchSources, e.MatchSourceOrder)
		e.MatchTargets = reorderNodes(e.MatchTargets, e.MatchTargetOrder)
	case KindSequence:
		e.SequenceDisplayOrder = ShuffleIDs(nodeIDs(e.SequenceItems), seed)
		e.SequenceItems = reorderNodes(e.SequenceItems, e.SequenceDisplayOrder)
	case KindLabel:
		e.LabelTokens = reorderNodes(e.LabelTokens, ShuffleIDs(nodeIDs(e.LabelTokens), seed))
	}
	return e
}
