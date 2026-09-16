package quiz

import "encoding/json"

type logicPayload struct {
	Kind   string   `json:"kind"`
	Seq    []string `json:"seq"`
	A      string   `json:"a"`
	Wrong  []string `json:"wrong"`
	Prompt string   `json:"prompt"`
	Speech string   `json:"speech"`
}

func logicSpecs(kp Kp) []Spec {
	var p logicPayload
	if err := json.Unmarshal([]byte(kp.Payload), &p); err != nil || p.A == "" {
		return nil
	}
	if len(p.Wrong) < optionCount-1 {
		return nil
	}

	prompt := p.Prompt
	if prompt == "" {
		prompt = "选一选"
	}
	speech := p.Speech
	if speech == "" {
		speech = prompt
	}

	// emoji 选项：答案和干扰项都是单个 emoji 时，用 Emoji 字段让按钮更大。
	useEmoji := looksLikeEmoji(p.A)
	for _, w := range p.Wrong {
		if !looksLikeEmoji(w) {
			useEmoji = false
			break
		}
	}

	specs := make([]Spec, 0, 2)
	for i, code := range []string{"pick1", "pick2"} {
		rng := rngFor(kp.ID, i+1)
		var opts []Option
		var idx int
		if useEmoji {
			picked := pickTiered([][]string{p.Wrong}, optionCount-1, func(s string) bool { return s == p.A }, rng)
			ds := make([]Option, 0, len(picked))
			for _, w := range picked {
				ds = append(ds, emojiOption(w))
			}
			opts, idx = buildOptions(emojiOption(p.A), ds, rng)
		} else {
			opts, idx = labelOptions(p.A, [][]string{p.Wrong}, rng)
		}

		sp := Spec{
			Code: code, Stem: prompt,
			Options: opts, AnswerIndex: idx,
			Speech: Speech{Text: speech, Lang: LangZH},
		}
		if len(p.Seq) > 0 {
			sp.Visual = Visual{Kind: "seq", Items: p.Seq}
		}
		specs = append(specs, sp)
	}
	return specs
}

// 粗判：不含常见汉字/字母数字、长度较短的当成 emoji 选项。
// 排序题的「1 → 2 → 3」会走文字分支。
func looksLikeEmoji(s string) bool {
	if s == "" || len([]rune(s)) > 4 {
		return false
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return false
		}
		if r >= 0x4e00 && r <= 0x9fff {
			return false
		}
	}
	return true
}

func emojiOption(s string) Option {
	name := emojiName(s)
	if name == "" {
		name = s
	}
	return Option{Emoji: s, Label: name}
}

func emojiName(s string) string {
	names := map[string]string{
		"🔴": "红", "🔵": "蓝", "🟢": "绿", "🟡": "黄", "⬛": "黑块", "⬜": "白块",
		"🔺": "三角", "⭐": "星星", "🍎": "苹果", "🍌": "香蕉", "🍇": "葡萄", "🍊": "橘子",
		"🍐": "梨", "🌙": "月亮", "☀️": "太阳", "☁️": "云", "🌈": "彩虹",
		"😊": "笑", "😢": "哭", "😡": "生气", "😴": "睡", "😄": "笑", "😁": "笑",
		"🐶": "狗", "🐱": "猫", "🐭": "老鼠", "🐰": "兔子", "🐻": "熊", "🐦": "鸟",
		"🦋": "蝴蝶", "🐕": "狗", "✈️": "飞机", "🚗": "小车", "🚌": "公交车",
		"🚕": "出租车", "🚲": "自行车", "🛴": "滑板车", "🛼": "溜冰鞋",
		"🌸": "花", "🌿": "草", "🌳": "树", "⭕": "空心圆", "⚫": "黑圆", "⚪": "白圆",
		"🟥": "红块", "🟧": "橙块", "🟨": "黄块", "🟩": "绿块", "🟦": "蓝块", "❤️": "红心",
		"•": "点", "➡️": "右", "⬇️": "下", "⬅️": "左", "⬆️": "上",
		"👉": "右", "👆": "上", "👇": "下", "👈": "左", "🍞": "面包", "🧀": "奶酪",
		"👟": "鞋子", "🏠": "房子", "🍕": "披萨", "🌧️": "雨", "❄️": "雪",
		"🎩": "帽子", "👀": "眼睛", "👂": "耳朵", "👃": "鼻子", "🍦": "冰淇淋",
		"✏️": "铅笔", "📏": "尺子", "📕": "书", "🔥": "火", "🧊": "冰",
		"💧": "水滴", "🐟": "鱼", "🐔": "鸡", "🐄": "牛", "🌅": "早上", "🌤️": "白天",
		"🌱": "小苗", "🧦": "袜子", "🙌": "手", "🧼": "肥皂", "🍂": "秋天",
		"👶": "宝宝", "👧": "小孩", "👩": "大人", "🍚": "饭", "🧽": "海绵",
		"🥚": "蛋", "🐣": "破壳", "🐥": "小鸡", "◼️": "方块", "🌺": "花", "🌹": "玫瑰",
		"🌞": "太阳", "🌑": "黑夜", "🌕": "月亮", "🌝": "圆月", "🔹": "小蓝",
		"🦒": "长颈鹿", "🐘": "大象", "🚀": "火箭", "🐢": "乌龟", "🐌": "蜗牛",
		"🚶": "走路", "🪶": "羽毛", "🎈": "气球", "🍃": "叶子", "🐍": "蛇",
		"🐛": "毛毛虫", "🐞": "瓢虫", "🌶️": "辣椒", "🍀": "四叶草", "🌊": "大海",
		"🫧": "泡泡", "💦": "水花", "🏀": "篮球", "🎾": "网球", "⚾": "棒球",
		"🍒": "樱桃", "🐜": "蚂蚁",
	}
	return names[s]
}
