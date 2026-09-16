package pinyin

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Reading provides Solo (standalone pronunciation) and Words (simple example characters) for TTS and 字中找拼音.
type Reading struct {
	Solo  string
	Words []string
}

func (r Reading) Word() string {
	if len(r.Words) == 0 {
		return ""
	}
	return r.Words[0]
}

// Readings maps pinyin letter (kp title) → Solo/Words. Each letter has 5 simple 例字; the first is the primary TTS word.
var Readings = map[string]Reading{
	"b":  {Solo: "波", Words: []string{"包", "八", "不", "白", "北"}},
	"p":  {Solo: "坡", Words: []string{"怕", "皮", "平", "跑", "片"}},
	"m":  {Solo: "摸", Words: []string{"吗", "马", "米", "木", "门"}},
	"f":  {Solo: "佛", Words: []string{"飞", "风", "饭", "分", "发"}},
	"d":  {Solo: "得", Words: []string{"大", "地", "的", "东", "对"}},
	"t":  {Solo: "特", Words: []string{"题", "天", "土", "头", "听"}},
	"n":  {Solo: "呢", Words: []string{"你", "女", "牛", "年", "南"}},
	"l":  {Solo: "勒", Words: []string{"来", "了", "六", "里", "老"}},
	"g":  {Solo: "哥", Words: []string{"高", "个", "工", "光", "给"}},
	"k":  {Solo: "科", Words: []string{"看", "口", "开", "可", "快"}},
	"h":  {Solo: "喝", Words: []string{"好", "火", "花", "和", "河"}},
	"j":  {Solo: "基", Words: []string{"家", "几", "九", "见", "叫"}},
	"q":  {Solo: "欺", Words: []string{"去", "七", "气", "前", "青"}},
	"x":  {Solo: "希", Words: []string{"小", "下", "西", "学", "星"}},
	"zh": {Solo: "知", Words: []string{"只", "中", "这", "住", "找"}},
	"ch": {Solo: "吃", Words: []string{"车", "出", "长", "吃", "成"}},
	"sh": {Solo: "诗", Words: []string{"书", "是", "十", "手", "上"}},
	"r":  {Solo: "日", Words: []string{"肉", "人", "日", "入", "热"}},
	"z":  {Solo: "资", Words: []string{"走", "在", "字", "早", "坐"}},
	"c":  {Solo: "次", Words: []string{"菜", "草", "从", "层", "次"}},
	"s":  {Solo: "思", Words: []string{"四", "三", "岁", "色", "送"}},
	"y":  {Solo: "衣", Words: []string{"羊", "一", "有", "雨", "云"}},
	"w":  {Solo: "乌", Words: []string{"我", "五", "晚", "外", "问"}},

	"a":   {Solo: "啊", Words: []string{"妈", "爸", "他", "大", "马"}},
	"o":   {Solo: "喔", Words: []string{"我", "波", "坡", "摸", "哦"}},
	"e":   {Solo: "鹅", Words: []string{"鹅", "哥", "河", "车", "的"}},
	"i":   {Solo: "衣", Words: []string{"米", "一", "你", "七", "西"}},
	"u":   {Solo: "乌", Words: []string{"鼓", "五", "不", "土", "书"}},
	"ü":   {Solo: "鱼", Words: []string{"鱼", "女", "雨", "绿", "去"}},
	"ai":  {Solo: "哀", Words: []string{"白", "来", "太", "开", "海"}},
	"ei":  {Solo: "诶", Words: []string{"飞", "北", "美", "给", "黑"}},
	"ui":  {Solo: "威", Words: []string{"水", "对", "会", "回", "岁"}},
	"ao":  {Solo: "熬", Words: []string{"猫", "好", "高", "老", "包"}},
	"ou":  {Solo: "欧", Words: []string{"头", "口", "走", "手", "豆"}},
	"iu":  {Solo: "优", Words: []string{"牛", "六", "九", "油", "球"}},
	"ie":  {Solo: "耶", Words: []string{"叶", "姐", "写", "夜", "爷"}},
	"üe":  {Solo: "约", Words: []string{"月", "学", "雪", "约", "觉"}},
	"er":  {Solo: "儿", Words: []string{"儿", "耳", "二", "而", "尔"}},
	"an":  {Solo: "安", Words: []string{"山", "看", "三", "半", "男"}},
	"en":  {Solo: "恩", Words: []string{"门", "人", "本", "分", "们"}},
	"in":  {Solo: "因", Words: []string{"心", "林", "金", "民", "音"}},
	"un":  {Solo: "温", Words: []string{"春", "村", "孙", "困", "寸"}},
	"ün":  {Solo: "晕", Words: []string{"云", "军", "群", "训", "匀"}},
	"ang": {Solo: "昂", Words: []string{"羊", "忙", "方", "长", "帮"}},
	"eng": {Solo: "", Words: []string{"灯", "风", "正", "生", "朋"}},
}

func EncodeWordExamples(words []string) string {
	if len(words) == 0 {
		return ""
	}
	raw, err := json.Marshal(words)
	if err != nil {
		return strings.Join(words, "")
	}
	return string(raw)
}

func ParseWordExamples(raw, fallback string) []string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var words []string
		if err := json.Unmarshal([]byte(raw), &words); err == nil && len(words) > 0 {
			return words
		}
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return []string{fallback}
	}
	return nil
}

func EncodeWordSpeechURLs(urls map[string]string) string {
	if len(urls) == 0 {
		return ""
	}
	raw, err := json.Marshal(urls)
	if err != nil {
		return ""
	}
	return string(raw)
}

func ParseWordSpeechURLs(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}
	}
	var urls map[string]string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil || urls == nil {
		return map[string]string{}
	}
	return urls
}

func SpeechKindForExampleIndex(index int) string {
	if index <= 0 {
		return "word"
	}
	return "word-" + strconv.Itoa(index)
}

func ParseExampleSpeechKind(kind string) (int, bool) {
	kind = strings.TrimSpace(kind)
	if kind == "word" {
		return 0, true
	}
	rest, ok := strings.CutPrefix(kind, "word-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 || n > 4 || strconv.Itoa(n) != rest {
		return 0, false
	}
	return n, true
}
