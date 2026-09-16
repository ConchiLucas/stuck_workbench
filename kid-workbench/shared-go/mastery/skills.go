package mastery

// 识字题型技能。字的「完全掌握」要求认、写都达到 mastered / review_due。
const (
	SkillGlyphSense = "glyph_sense" // 看字图选义图
	SkillSenseChar  = "sense_char"  // 看义图选字
	SkillWriteChar  = "write_char"  // 按笔顺手写

	// SkillListenGlyph 已下线；历史作答仍可映射展示，但不计入掌握维度。
	SkillListenGlyph = "listen_glyph"
)

// 拼音题型技能；字母三项与音节拼读分别记账。
const (
	SkillPinyinInWord    = "inword" // 听例字选音
	SkillPinyinListen    = "listen" // 听单读选字母
	SkillPinyinShape     = "shape"
	SkillPinyinBlend     = "blend"
	PinyinSyllableModule = "syllables"
)

// 英语题型技能。listen 与拼音共用题码，解析时必须结合学科。
const (
	SkillEnglishListen  = "listen"
	SkillEnglishPicture = "picture"
	SkillEnglishBuild   = "build"
	SkillEnglishType    = "type"
	SkillEnglishRead    = "read"
)

const (
	SkillScienceRecognize = "recognize" // 历史题码，映射为 choice
	SkillScienceChoice    = "choice"
	SkillScienceMatch      = "match"
	SkillScienceSequence   = "sequence"
	SkillScienceLabel      = "label"
)

var ScienceSkills = []string{SkillScienceChoice, SkillScienceMatch, SkillScienceSequence, SkillScienceLabel}

const (
	SkillPoemTitle   = "title"
	SkillPoemFill    = "fill"
	SkillPoemCouplet = "couplet"
	SkillPoemRecite  = "recite"
)

var PoemSkills = []string{SkillPoemTitle, SkillPoemFill, SkillPoemCouplet, SkillPoemRecite}

var LogicSkills = []string{"pattern", "classify", "order", "shape_reason", "diff", "compare"}

const (
	SkillMathCalc  = "calc"
	SkillMathStory = "story"
	SkillMathFind  = "find"
	SkillMathName  = "name"
)

// LiteracySkills 固定顺序，供矩阵展示与灌题使用。
var LiteracySkills = []string{
	SkillGlyphSense,
	SkillSenseChar,
	SkillWriteChar,
}

// PinyinSkills is the subject's candidate set, not each point's required set.
var PinyinSkills = []string{SkillPinyinListen, SkillPinyinInWord, SkillPinyinShape, SkillPinyinBlend}
var PinyinLetterSkills = []string{SkillPinyinListen, SkillPinyinInWord, SkillPinyinShape}
var PinyinSyllableSkills = []string{SkillPinyinBlend}

var EnglishSkills = []string{
	SkillEnglishListen,
	SkillEnglishPicture,
	SkillEnglishBuild,
	SkillEnglishType,
	SkillEnglishRead,
}

const (
	SkillPhraseListenZh = "listen_zh"
	SkillPhraseListenEn = "listen_en"
	SkillPhraseScene    = "scene"
	SkillPhraseReply    = "reply"
)

var PhraseSkills = []string{SkillPhraseListenZh, SkillPhraseListenEn, SkillPhraseScene, SkillPhraseReply}
var PhraseCoreSkills = []string{SkillPhraseListenZh, SkillPhraseListenEn, SkillPhraseScene}

const (
	SkillChengyuMeaning = "meaning"
	SkillChengyuPick    = "pick"
	SkillChengyuPinyin  = "pinyin"
	SkillChengyuExample = "example"
)

var ChengyuSkills = []string{SkillChengyuMeaning, SkillChengyuPick, SkillChengyuPinyin, SkillChengyuExample}

var MathArithmeticSkills = []string{SkillMathCalc, SkillMathStory}
var MathShapeSkills = []string{SkillMathFind, SkillMathName}

// SkillsFor 返回学科、模块需要按技能记账的 code 列表；返回值可安全修改。
func SkillsFor(subjectCode, moduleCode string) []string {
	switch subjectCode {
	case "literacy":
		return append([]string{}, LiteracySkills...)
	case "pinyin":
		if moduleCode == PinyinSyllableModule {
			return append([]string{}, PinyinSyllableSkills...)
		}
		return append([]string{}, PinyinLetterSkills...)
	case "english":
		return append([]string{}, EnglishSkills...)
	case "science":
		return nil
	case "logic":
		return nil
	case "poem":
		return append([]string{}, PoemSkills...)
	case "phrase":
		return append([]string{}, PhraseCoreSkills...)
	case "chengyu":
		return append([]string{}, ChengyuSkills...)
	case "math":
		switch moduleCode {
		case "add10", "sub10":
			return append([]string{}, MathArithmeticSkills...)
		case "shape":
			return append([]string{}, MathShapeSkills...)
		}
	}
	return nil
}

// SkillIsTracked 表示该技能可以写入作答与分项账本。
// 短句「问与答」会计入 reply 分项，但不在 SkillsFor 里，因此不参与完全掌握。
func SkillIsTracked(subjectCode, moduleCode, skillCode string) bool {
	if skillCode == "" {
		return false
	}
	for _, code := range SkillsFor(subjectCode, moduleCode) {
		if code == skillCode {
			return true
		}
	}
	return subjectCode == "phrase" && skillCode == SkillPhraseReply ||
		subjectCode == "science" && scienceSkill(skillCode) ||
		subjectCode == "poem" && poemSkill(skillCode) ||
		subjectCode == "logic" && logicSkill(skillCode)
}

// SkillsForSubject 返回该学科需要按技能记账的 code 列表；空表示按 KP 单行掌握。
func SkillsForSubject(subjectCode string) []string {
	if subjectCode == "pinyin" {
		return append([]string{}, PinyinSkills...)
	}
	return SkillsFor(subjectCode, "")
}

// SkillDisplayName 是进度总览/知识库共用的题型名称。
func SkillDisplayName(subjectCode, skillCode string) string {
	switch subjectCode {
	case "english":
		switch skillCode {
		case SkillEnglishListen:
			return "听音选词"
		case SkillEnglishPicture:
			return "看图选词"
		case SkillEnglishBuild:
			return "组句子"
		case SkillEnglishType:
			return "写单词"
		case SkillEnglishRead:
			return "读一读"
		}
	case "literacy":
		switch skillCode {
		case SkillGlyphSense:
			return "看字选义"
		case SkillSenseChar:
			return "看义选字"
		case SkillWriteChar:
			return "手写笔顺"
		}
	case "pinyin":
		switch skillCode {
		case SkillPinyinListen:
			return "听音选字母"
		case SkillPinyinInWord:
			return "字中找拼音"
		case SkillPinyinShape:
			return "看形认读"
		case SkillPinyinBlend:
			return "音节拼读"
		}
	case "math":
		switch skillCode {
		case SkillMathCalc:
			return "算式计算"
		case SkillMathStory:
			return "情境应用"
		case SkillMathFind:
			return "听音找图形"
		case SkillMathName:
			return "看图认名称"
		}
	case "logic":
		switch skillCode {
		case "pattern":
			return "找规律"
		case "classify":
			return "分类"
		case "order":
			return "排序"
		case "shape_reason":
			return "图形推理"
		case "diff":
			return "找不同"
		case "compare":
			return "比较"
		}
	case "science":
		switch skillCode {
		case SkillScienceChoice:
			return "选择题"
		case SkillScienceMatch:
			return "连线题"
		case SkillScienceSequence:
			return "排序题"
		case SkillScienceLabel:
			return "结构标注题"
		case SkillScienceRecognize:
			return "科普辨认"
		}
	case "poem":
		switch skillCode {
		case SkillPoemTitle:
			return "选诗名"
		case SkillPoemFill:
			return "补字"
		case SkillPoemCouplet:
			return "选下一句"
		case SkillPoemRecite:
			return "排顺序"
		}
	case "phrase":
		switch skillCode {
		case SkillPhraseListenZh:
			return "听一听"
		case SkillPhraseListenEn:
			return "选句子"
		case SkillPhraseScene:
			return "什么时候说"
		case SkillPhraseReply:
			return "问与答"
		}
	case "chengyu":
		switch skillCode {
		case SkillChengyuMeaning:
			return "听释义"
		case SkillChengyuPick:
			return "选成语"
		case SkillChengyuPinyin:
			return "看拼音"
		case SkillChengyuExample:
			return "看句子"
		}
	}
	if skillCode == "" {
		return "题型未记录"
	}
	return skillCode
}

// SkillFromQuestionCode 结合学科把题目 code 映射到技能。
// 题码并非全局唯一（例如 pinyin 与 english 都有 listen），不能脱离学科解析。
func SkillFromQuestionCode(subjectCode, questionCode string) string {
	for _, skill := range SkillsForSubject(subjectCode) {
		if skill == questionCode {
			return skill
		}
	}
	if subjectCode == "math" {
		for _, skill := range append(MathArithmeticSkills, MathShapeSkills...) {
			if skill == questionCode {
				return skill
			}
		}
	}
	if subjectCode == "phrase" {
		for _, skill := range PhraseSkills {
			if skill == questionCode {
				return skill
			}
		}
	}
	if subjectCode == "science" {
		if questionCode == SkillScienceRecognize {
			return SkillScienceChoice
		}
		if scienceSkill(questionCode) {
			return questionCode
		}
	}
	if subjectCode == "poem" {
		if questionCode == "nextline" {
			return SkillPoemCouplet
		}
		if poemSkill(questionCode) {
			return questionCode
		}
	}
	if subjectCode == "logic" && logicSkill(questionCode) {
		return questionCode
	}
	if subjectCode == "chengyu" {
		for _, skill := range ChengyuSkills {
			if skill == questionCode {
				return skill
			}
		}
	}
	return ""
}

func scienceSkill(code string) bool {
	for _, skill := range ScienceSkills {
		if skill == code {
			return true
		}
	}
	return false
}

func poemSkill(code string) bool {
	for _, skill := range PoemSkills {
		if skill == code {
			return true
		}
	}
	return false
}

func logicSkill(code string) bool {
	for _, skill := range LogicSkills {
		if skill == code {
			return true
		}
	}
	return false
}

// RollupSkills 把字下各技能状态收成一个字级状态。
//
// 规则（从严）：
//  1. 空 → not_started
//  2. 任一 shaky → shaky
//  3. 全部为 mastered 或 review_due → 有 review_due 则 review_due，否则 mastered
//  4. 有任何进展但未全过 → learning
//  5. 否则 not_started
func RollupSkills(statuses []Status) Status {
	if len(statuses) == 0 {
		return StatusNotStarted
	}

	anyShaky := false
	anyDue := false
	anyProgress := false
	doneCount := 0

	for _, st := range statuses {
		switch st {
		case StatusShaky:
			anyShaky = true
			anyProgress = true
		case StatusLearning:
			anyProgress = true
		case StatusReviewDue:
			anyDue = true
			anyProgress = true
			doneCount++
		case StatusMastered:
			anyProgress = true
			doneCount++
		}
	}

	if anyShaky {
		return StatusShaky
	}
	if doneCount >= len(statuses) {
		if anyDue {
			return StatusReviewDue
		}
		return StatusMastered
	}
	if anyProgress {
		return StatusLearning
	}
	return StatusNotStarted
}

// IsSkillDone 技能是否算「过关」（掌握或待复习）。
func IsSkillDone(st Status) bool {
	return st == StatusMastered || st == StatusReviewDue
}
