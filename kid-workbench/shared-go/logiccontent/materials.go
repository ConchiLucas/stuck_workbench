package logiccontent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KnowledgePoint struct {
	ID         int64  `gorm:"primaryKey"`
	ModuleID   int64  `gorm:"column:module_id"`
	Code       string `gorm:"column:code"`
	Title      string `gorm:"column:title"`
	OrderNo    int    `gorm:"column:order_no"`
	Difficulty int    `gorm:"column:difficulty"`
	Payload    string `gorm:"column:payload"`
}

func (KnowledgePoint) TableName() string { return "knowledge_points" }

type QuestionRow struct {
	ID      int64  `gorm:"primaryKey"`
	KpID    int64  `gorm:"column:kp_id"`
	Code    string `gorm:"column:code"`
	Stem    string `gorm:"column:stem"`
	Options string `gorm:"column:options"`
	Answer  string `gorm:"column:answer"`
	Visual  string `gorm:"column:visual"`
	Speech  string `gorm:"column:speech"`
}

func (QuestionRow) TableName() string { return "questions" }

func EnsureMaterials(db *gorm.DB) error {
	if err := MigrateMedia(db); err != nil {
		return err
	}
	var subjectID, moduleID int64
	if err := db.Raw(`SELECT id FROM subjects WHERE code = ?`, SubjectCode).Scan(&subjectID).Error; err != nil {
		return err
	}
	if subjectID == 0 {
		return fmt.Errorf("缺少 logic 学科")
	}
	if err := db.Raw(`SELECT id FROM modules WHERE subject_id = ? AND code = 'playground'`, subjectID).Scan(&moduleID).Error; err != nil {
		return err
	}
	if moduleID == 0 {
		if err := db.Exec(`INSERT INTO modules (subject_id, code, name, order_no) VALUES (?, 'playground', '逻辑练习', 80)`, subjectID).Error; err != nil {
			return err
		}
		if err := db.Raw(`SELECT id FROM modules WHERE subject_id = ? AND code = 'playground'`, subjectID).Scan(&moduleID).Error; err != nil {
			return err
		}
	}
	if moduleID == 0 {
		return fmt.Errorf("缺少 logic playground 模块")
	}
	specs := CuratedMaterials()
	for i, spec := range specs {
		if err := Validate(spec.Example); err != nil {
			return fmt.Errorf("%s: %w", spec.Code, err)
		}
		payload := EncodeJSON(map[string]any{
			"kind": spec.Example.Kind, "rule": spec.Example.Rule, "example": spec.Example, "glyphVersion": GlyphVersion,
		})
		var kpID int64
		if err := db.Raw(`SELECT id FROM knowledge_points WHERE module_id = ? AND code = ?`, moduleID, spec.Code).Scan(&kpID).Error; err != nil {
			return err
		}
		if kpID == 0 {
			if err := db.Exec(`INSERT INTO knowledge_points (module_id, code, title, payload, difficulty, order_no) VALUES (?, ?, ?, ?, 1, ?)`,
				moduleID, spec.Code, spec.Title, payload, 200+i).Error; err != nil {
				return err
			}
			if err := db.Raw(`SELECT id FROM knowledge_points WHERE module_id = ? AND code = ?`, moduleID, spec.Code).Scan(&kpID).Error; err != nil {
				return err
			}
		} else if err := db.Exec(`UPDATE knowledge_points SET title = ?, payload = ?, order_no = ? WHERE id = ?`, spec.Title, payload, 200+i, kpID).Error; err != nil {
			return err
		}
		example := spec.Example
		example.KpID = kpID
		example.GlyphVersion = GlyphVersion
		example.ImageURLs = LiveImageURLs(kpID, example.Objects)
		if err := upsertLiveMedia(db, kpID, example.Objects); err != nil {
			return err
		}
		visual := EncodeJSON(map[string]any{"example": example, "items": captions(example, example.Sequence)})
		options := EncodeJSON(choiceOptions(example))
		answer := EncodeJSON(map[string]any{"answerId": example.AnswerID, "correctSequence": example.CorrectSequence})
		var qid int64
		if err := db.Raw(`SELECT id FROM questions WHERE kp_id = ? AND code = ?`, kpID, spec.Example.Kind).Scan(&qid).Error; err != nil {
			return err
		}
		if qid == 0 {
			if err := db.Exec(`INSERT INTO questions (kp_id, code, type, stem, options, answer, visual, difficulty) VALUES (?, ?, 'practice', ?, ?, ?, ?, 1)`,
				kpID, spec.Example.Kind, example.Prompt, options, answer, visual).Error; err != nil {
				return err
			}
		} else if err := db.Exec(`UPDATE questions SET stem = ?, options = ?, answer = ?, visual = ? WHERE id = ?`, example.Prompt, options, answer, visual, qid).Error; err != nil {
			return err
		}
	}
	return nil
}

func LiveImageURLs(kpID int64, objects []Object) map[string]string {
	out := map[string]string{}
	for _, o := range objects {
		out[o.ID] = fmt.Sprintf(LiveGlyphPath, kpID, o.ID)
	}
	return out
}

func upsertLiveMedia(db *gorm.DB, kpID int64, objects []Object) error {
	for _, o := range objects {
		data := RenderSVG(o)
		sum := sha256.Sum256(data)
		row := ItemMedia{KpID: kpID, ObjectID: o.ID, SHA256: hex.EncodeToString(sum[:]), MIME: "image/svg+xml", Data: data}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "kp_id"}, {Name: "object_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"sha256", "mime", "data"}),
		}).Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func captions(e LogicExample, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if o, ok := ObjectByID(e.Objects, id); ok {
			out = append(out, o.Caption)
		}
	}
	return out
}

func choiceOptions(e LogicExample) []map[string]any {
	if e.Kind == "order" {
		var out []map[string]any
		for i, id := range e.DisplayOrder {
			o, _ := ObjectByID(e.Objects, id)
			out = append(out, map[string]any{"id": id, "label": o.Caption, "index": i})
		}
		return out
	}
	var out []map[string]any
	for i, id := range e.Options {
		o, _ := ObjectByID(e.Objects, id)
		out = append(out, map[string]any{"id": id, "label": o.Caption, "index": i})
	}
	return out
}

type MaterialSpec struct {
	Code    string
	Title   string
	Example LogicExample
}

func obj(id, caption, glyph, fill string, scale int, attrs map[string]string) Object {
	return Object{ID: id, Caption: caption, Glyph: glyph, Fill: fill, Scale: scale, Attrs: attrs}
}

func objR(id, caption, glyph, fill string, rotate int, attrs map[string]string) Object {
	return Object{ID: id, Caption: caption, Glyph: glyph, Fill: fill, Scale: 2, Rotate: rotate, Attrs: attrs}
}

func CuratedMaterials() []MaterialSpec {
	red, blue, yellow, green := "#e11d48", "#2563eb", "#eab308", "#16a34a"
	return []MaterialSpec{
		patternSpec("logic-pattern-ab", "红蓝交替", "下一个是哪个？", "AB", 2, "color",
			[]Object{obj("r1", "红", "circle", red, 2, map[string]string{"color": "red"}), obj("b1", "蓝", "circle", blue, 2, map[string]string{"color": "blue"}), obj("r2", "红", "circle", red, 2, map[string]string{"color": "red"}), obj("g1", "绿", "circle", green, 2, map[string]string{"color": "green"})},
			[]string{"r1", "b1", "r2"}, "b1", []string{"r1", "b1", "g1"}),
		patternSpec("logic-pattern-aabb", "成对水果", "下一个是哪个？", "AABB", 2, "kind",
			[]Object{obj("a1", "苹果", "apple", red, 2, map[string]string{"kind": "apple"}), obj("a2", "苹果", "apple", red, 2, map[string]string{"kind": "apple"}), obj("bn1", "香蕉", "banana", yellow, 2, map[string]string{"kind": "banana"}), obj("bn2", "香蕉", "banana", yellow, 2, map[string]string{"kind": "banana"}), obj("p1", "梨", "pear", green, 2, map[string]string{"kind": "pear"})},
			[]string{"a1", "a2", "bn1"}, "bn2", []string{"a1", "bn2", "p1"}),
		patternSpec("logic-pattern-abc", "形状轮换", "下一个是哪个？", "ABC", 3, "shape",
			[]Object{obj("c1", "圆", "circle", blue, 2, map[string]string{"shape": "circle"}), obj("s1", "方", "square", blue, 2, map[string]string{"shape": "square"}), obj("t1", "三角", "triangle", blue, 2, map[string]string{"shape": "triangle"}), obj("st1", "星", "star", blue, 2, map[string]string{"shape": "star"})},
			[]string{"c1", "s1", "t1"}, "c1", []string{"c1", "s1", "st1"}),
		patternSpec("logic-pattern-size", "大小递进", "下一个是哪个？", "grow", 0, "scale",
			[]Object{obj("n1", "小圆", "circle", blue, 1, map[string]string{"size": "1"}), obj("n2", "中圆", "circle", blue, 2, map[string]string{"size": "2"}), obj("n3", "大圆", "circle", blue, 3, map[string]string{"size": "3"}), obj("n4", "更大", "circle", blue, 4, map[string]string{"size": "4"})},
			[]string{"n1", "n2", "n3"}, "n4", []string{"n2", "n4", "n1"}),

		classifySpec("logic-classify-animal", "动物里的例外", "哪个不属于这一类？", "kingdom", "animal",
			[]Object{obj("cat", "猫", "cat", "#f59e0b", 2, map[string]string{"category": "animal"}), obj("dog", "狗", "dog", "#92400e", 2, map[string]string{"category": "animal"}), obj("bird", "鸟", "bird", "#0ea5e9", 2, map[string]string{"category": "animal"}), obj("car", "汽车", "car", red, 2, map[string]string{"category": "vehicle"})},
			[]string{"cat", "dog", "bird", "car"}, "car"),
		classifySpec("logic-classify-fruit", "水果里的例外", "哪个不属于这一类？", "kind", "fruit",
			[]Object{obj("ap", "苹果", "apple", red, 2, map[string]string{"category": "fruit"}), obj("ba", "香蕉", "banana", yellow, 2, map[string]string{"category": "fruit"}), obj("pe", "梨", "pear", green, 2, map[string]string{"category": "fruit"}), obj("ca", "胡萝卜", "carrot", "#f97316", 2, map[string]string{"category": "vegetable"})},
			[]string{"ap", "ba", "pe", "ca"}, "ca"),
		classifySpec("logic-classify-red", "红色里的例外", "哪个不属于这一类？", "color", "red",
			[]Object{obj("ra", "红苹果", "apple", red, 2, map[string]string{"category": "red"}), obj("rh", "红心", "heart", red, 2, map[string]string{"category": "red"}), obj("rc", "红圆", "circle", red, 2, map[string]string{"category": "red"}), obj("bl", "蓝圆", "circle", blue, 2, map[string]string{"category": "blue"})},
			[]string{"ra", "rh", "rc", "bl"}, "bl"),
		classifySpec("logic-classify-round", "圆形里的例外", "哪个不属于这一类？", "shape", "round",
			[]Object{obj("ci", "圆", "circle", blue, 2, map[string]string{"category": "round"}), obj("ap2", "苹果", "apple", red, 2, map[string]string{"category": "round"}), obj("gr", "葡萄", "grape", "#7c3aed", 2, map[string]string{"category": "round"}), obj("tr", "三角", "triangle", yellow, 2, map[string]string{"category": "pointy"})},
			[]string{"ci", "ap2", "gr", "tr"}, "tr"),

		orderSpec("logic-order-count", "数量从小到大", "按数量从小到大点一排", "count", "asc",
			[]Object{obj("o1", "1", "one", blue, 2, map[string]string{"count": "1"}), obj("o2", "2", "two", blue, 2, map[string]string{"count": "2"}), obj("o3", "3", "three", blue, 2, map[string]string{"count": "3"})},
			[]string{"o1", "o2", "o3"}, []string{"o2", "o3", "o1"}),
		orderSpec("logic-order-season", "季节先后", "按春夏秋冬点一排", "season", "asc",
			[]Object{obj("sp", "花", "flower", red, 2, map[string]string{"order": "1"}), obj("su", "太阳", "sun", yellow, 2, map[string]string{"order": "2"}), obj("au", "叶", "leaf", "#ea580c", 2, map[string]string{"order": "3"}), obj("wi", "雪", "snow", "#38bdf8", 2, map[string]string{"order": "4"})},
			[]string{"sp", "su", "au", "wi"}, []string{"au", "sp", "wi", "su"}),
		orderSpec("logic-order-size", "从小到大", "按从小到大点一排", "visualSize", "asc",
			[]Object{obj("sml", "小", "circle", blue, 1, map[string]string{"size": "1"}), obj("mid", "中", "circle", blue, 2, map[string]string{"size": "2"}), obj("big", "大", "circle", blue, 3, map[string]string{"size": "3"})},
			[]string{"sml", "mid", "big"}, []string{"big", "sml", "mid"}),
		orderSpec("logic-order-dup", "相同图形仍分实例", "按 1-2-1 的编号点一排", "label", "asc",
			[]Object{obj("d1", "红一", "circle", red, 2, map[string]string{"n": "1"}), obj("d2", "蓝二", "circle", blue, 2, map[string]string{"n": "2"}), obj("d3", "红三", "circle", red, 2, map[string]string{"n": "3"})},
			[]string{"d1", "d2", "d3"}, []string{"d3", "d1", "d2"}),

		shapeSpec("logic-shape-color", "颜色轮换", "下一个图形是哪个？", "ABC-color", "color",
			[]Object{obj("sc1", "红圆", "circle", red, 2, map[string]string{"color": "red"}), obj("sc2", "蓝圆", "circle", blue, 2, map[string]string{"color": "blue"}), obj("sc3", "黄圆", "circle", yellow, 2, map[string]string{"color": "yellow"}), obj("sc4", "绿圆", "circle", green, 2, map[string]string{"color": "green"})},
			[]string{"sc1", "sc2", "sc3"}, "sc1", []string{"sc1", "sc4", "sc2"}),
		shapeSpec("logic-shape-form", "形状轮换", "下一个图形是哪个？", "ABC-shape", "shape",
			[]Object{obj("sf1", "圆", "circle", blue, 2, map[string]string{"shape": "circle"}), obj("sf2", "方", "square", blue, 2, map[string]string{"shape": "square"}), obj("sf3", "三角", "triangle", blue, 2, map[string]string{"shape": "triangle"}), obj("sf4", "菱形", "diamond", blue, 2, map[string]string{"shape": "diamond"})},
			[]string{"sf1", "sf2", "sf3"}, "sf1", []string{"sf1", "sf4", "sf2"}),
		shapeSpec("logic-shape-turn", "方向递进", "下一个图形是哪个？", "rotate", "direction",
			[]Object{objR("t0", "向右", "arrow", blue, 0, map[string]string{"dir": "0"}), objR("t90", "向下", "arrow", blue, 90, map[string]string{"dir": "90"}), objR("t180", "向左", "arrow", blue, 180, map[string]string{"dir": "180"}), objR("t270", "向上", "arrow", blue, 270, map[string]string{"dir": "270"})},
			[]string{"t0", "t90", "t180"}, "t270", []string{"t270", "t0", "t90"}),
		shapeSpec("logic-shape-count", "数量递进", "下一个图形是哪个？", "count", "count",
			[]Object{
				{ID: "c1", Caption: "一个点", Glyph: "circle", Fill: blue, Scale: 2, Count: 1, Attrs: map[string]string{"n": "1"}},
				{ID: "c2", Caption: "两个点", Glyph: "circle", Fill: blue, Scale: 2, Count: 2, Attrs: map[string]string{"n": "2"}},
				{ID: "c3", Caption: "三个点", Glyph: "circle", Fill: blue, Scale: 2, Count: 3, Attrs: map[string]string{"n": "3"}},
				{ID: "c4", Caption: "四个点", Glyph: "circle", Fill: blue, Scale: 2, Count: 4, Attrs: map[string]string{"n": "4"}},
			},
			[]string{"c1", "c2", "c3"}, "c4", []string{"c2", "c4", "c1"}),

		diffSpec("logic-diff-color", "颜色不同", "哪一个和其他不一样？", "color",
			[]Object{obj("dc1", "蓝圆", "circle", blue, 2, map[string]string{"color": "blue"}), obj("dc2", "蓝圆", "circle", blue, 2, map[string]string{"color": "blue"}), obj("dc3", "红圆", "circle", red, 2, map[string]string{"color": "red"}), obj("dc4", "蓝圆", "circle", blue, 2, map[string]string{"color": "blue"})},
			[]string{"dc1", "dc2", "dc3", "dc4"}, "dc3"),
		diffSpec("logic-diff-shape", "形状不同", "哪一个和其他不一样？", "shape",
			[]Object{obj("ds1", "圆", "circle", blue, 2, map[string]string{"shape": "circle"}), obj("ds2", "圆", "circle", blue, 2, map[string]string{"shape": "circle"}), obj("ds3", "方", "square", blue, 2, map[string]string{"shape": "square"}), obj("ds4", "圆", "circle", blue, 2, map[string]string{"shape": "circle"})},
			[]string{"ds1", "ds2", "ds3", "ds4"}, "ds3"),
		diffSpec("logic-diff-dir", "方向不同", "哪一个和其他不一样？", "direction",
			[]Object{objR("dr1", "向右", "arrow", blue, 0, map[string]string{"direction": "right"}), objR("dr2", "向右", "arrow", blue, 0, map[string]string{"direction": "right"}), objR("dr3", "向左", "arrow", blue, 180, map[string]string{"direction": "left"}), objR("dr4", "向右", "arrow", blue, 0, map[string]string{"direction": "right"})},
			[]string{"dr1", "dr2", "dr3", "dr4"}, "dr3"),
		diffSpec("logic-diff-fruit-color", "水果颜色不同", "哪一个和其他不一样？", "color",
			[]Object{obj("df1", "红苹果", "apple", red, 2, map[string]string{"color": "red"}), obj("df2", "红心", "heart", red, 2, map[string]string{"color": "red"}), obj("df3", "绿梨", "pear", green, 2, map[string]string{"color": "green"}), obj("df4", "红圆", "circle", red, 2, map[string]string{"color": "red"})},
			[]string{"df1", "df2", "df3", "df4"}, "df3"),

		compareSpec("logic-compare-visual", "画面更大", "哪个更大？", "visualSize", "max",
			[]Object{obj("bus", "公交车", "bus", "#2563eb", 4, map[string]string{"visualSize": "4"}), obj("skate", "溜冰鞋", "skate", "#db2777", 1, map[string]string{"visualSize": "1"}), obj("bike", "自行车", "bike", "#16a34a", 2, map[string]string{"visualSize": "2"})},
			[]string{"bus", "skate", "bike"}, "bus"),
		compareSpec("logic-compare-height", "谁更高", "哪一个更高？", "visualSize", "max",
			[]Object{obj("gir", "长颈鹿", "giraffe", "#d97706", 4, map[string]string{"visualSize": "4"}), obj("mou", "老鼠", "mouse", "#9ca3af", 1, map[string]string{"visualSize": "1"}), obj("rab", "兔子", "rabbit", "#a3a3a3", 2, map[string]string{"visualSize": "2"})},
			[]string{"gir", "mou", "rab"}, "gir"),
		compareSpec("logic-compare-count", "数量最多", "哪一堆更多？", "count", "max",
			[]Object{
				{ID: "k1", Caption: "一个", Glyph: "circle", Fill: blue, Scale: 2, Count: 1, Attrs: map[string]string{"count": "1"}},
				{ID: "k2", Caption: "两个", Glyph: "circle", Fill: blue, Scale: 2, Count: 2, Attrs: map[string]string{"count": "2"}},
				{ID: "k3", Caption: "三个", Glyph: "circle", Fill: blue, Scale: 2, Count: 3, Attrs: map[string]string{"count": "3"}},
			},
			[]string{"k1", "k2", "k3"}, "k3"),
		compareSpec("logic-compare-speed", "谁更快（交通工具）", "在交通工具里，哪个更快？", "speed", "max",
			[]Object{obj("car2", "汽车", "car", red, 2, map[string]string{"speed": "3"}), obj("bike2", "自行车", "bike", green, 2, map[string]string{"speed": "2"}), obj("skate2", "溜冰鞋", "skate", "#db2777", 2, map[string]string{"speed": "1"})},
			[]string{"car2", "bike2", "skate2"}, "car2"),
	}
}

func patternSpec(code, title, prompt, ruleType string, period int, dim string, objects []Object, seq []string, answer string, options []string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "pattern", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: ruleType, Period: period, Dimension: dim, Explain: title + "：按已展示序列继续。"},
		Objects: objects, Sequence: seq, Options: options, AnswerID: answer,
	}}
}

func classifySpec(code, title, prompt, dim, inGroup string, objects []Object, options []string, answer string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "classify", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: "odd-one-out", Dimension: dim, InGroup: inGroup, Explain: title},
		Objects: objects, Options: options, AnswerID: answer,
	}}
}

func orderSpec(code, title, prompt, dim, dir string, objects []Object, correct, display []string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "order", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: "order", Dimension: dim, Direction: dir, Explain: title},
		Objects: objects, CorrectSequence: correct, DisplayOrder: display,
	}}
}

func shapeSpec(code, title, prompt, ruleType, dim string, objects []Object, seq []string, answer string, options []string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "shape_reason", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: ruleType, Dimension: dim, Explain: title + "：图形按该维度变化。"},
		Objects: objects, Sequence: seq, Options: options, AnswerID: answer,
	}}
}

func diffSpec(code, title, prompt, dim string, objects []Object, options []string, answer string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "diff", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: "unique-odd", Dimension: dim, Explain: title + "：只有一项在该维度上不同。"},
		Objects: objects, Options: options, AnswerID: answer,
	}}
}

func compareSpec(code, title, prompt, dim, dir string, objects []Object, options []string, answer string) MaterialSpec {
	return MaterialSpec{Code: code, Title: title, Example: LogicExample{
		Kind: "compare", Prompt: prompt, GlyphVersion: GlyphVersion,
		Rule: Rule{Type: "compare", Dimension: dim, Direction: dir, Explain: title},
		Objects: objects, Options: options, AnswerID: answer,
	}}
}
