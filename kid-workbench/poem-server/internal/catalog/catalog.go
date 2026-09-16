package catalog

type Pavilion struct {
	Code      string `json:"code"`
	Title     string `json:"title"`
	KidTitle  string `json:"kidTitle"`
	Kind      string `json:"kind"`
	OrderNo   int    `json:"orderNo"`
	ClueCount int    `json:"clueCount"`
	Line      string `json:"line"`
}

type Option struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Visual string `json:"visual"`
}

type Clue struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Prompt   string   `json:"prompt"`
	Line     string   `json:"line,omitempty"`
	Speech   string   `json:"speech"`
	AnswerID string   `json:"answerId"`
	Options  []Option `json:"options"`
	Sequence []string `json:"sequence,omitempty"`
}

func Pavilions() []Pavilion {
	rows := []Pavilion{
		{Code: "moon", Title: "月下亭", KidTitle: "望月", Kind: "choice", OrderNo: 1, Line: "床前明月光"},
		{Code: "dawn", Title: "春晓径", KidTitle: "闻鸟", Kind: "choice", OrderNo: 2, Line: "处处闻啼鸟"},
		{Code: "goose", Title: "白鹅池", KidTitle: "诵句", Kind: "recite", OrderNo: 3, Line: "鹅鹅鹅"},
		{Code: "farm", Title: "田园井", KidTitle: "补字", Kind: "fill", OrderNo: 4, Line: "锄禾日当午"},
		{Code: "tower", Title: "鹳楼台", KidTitle: "对句", Kind: "couplet", OrderNo: 5, Line: "欲穷千里目"},
		{Code: "snow", Title: "寒江矶", KidTitle: "寻影", Kind: "choice", OrderNo: 6, Line: "独钓寒江雪"},
		{Code: "scroll", Title: "画中卷", KidTitle: "读画", Kind: "choice", OrderNo: 7, Line: "近听水无声"},
	}
	for i := range rows {
		rows[i].ClueCount = len(Clues(rows[i].Code))
	}
	return rows
}

func PavilionByCode(code string) (Pavilion, bool) {
	for _, row := range Pavilions() {
		if row.Code == code {
			return row, true
		}
	}
	return Pavilion{}, false
}

func Clues(code string) []Clue {
	switch code {
	case "moon":
		return []Clue{
			choice("moon-light", "诗句里写了什么？", "床前明月光", "moon",
				opt("sun", "太阳", "sun"), opt("lamp", "灯", "lamp"), opt("moon", "月亮", "moon"), opt("star", "星星", "star")),
			choice("moon-frost", "地上像什么？", "疑是地上霜", "frost",
				opt("frost", "霜", "frost"), opt("snow", "雪", "snow"), opt("flower", "花", "flower"), opt("rain", "雨", "rain")),
      title("moon-title", "这首诗叫什么？", "床前明月光", "jingyesi",
				opt("chunxiao", "春晓", "char"), opt("yonge", "咏鹅", "char"), opt("jingyesi", "静夜思", "char"), opt("minnong", "悯农", "char")),
			author("moon-author", "这首诗是谁写的？", "床前明月光", "libai",
				opt("dufu", "杜甫", "char"), opt("libai", "李白", "char"), opt("wangwei", "王维", "char"), opt("menghaoran", "孟浩然", "char")),
		}
	case "dawn":
		return []Clue{
			choice("dawn-bird", "诗句里听见了什么？", "处处闻啼鸟", "bird",
				opt("bell", "钟", "bell"), opt("horse", "马", "horse"), opt("bird", "鸟", "bird"), opt("drum", "鼓", "drum")),
			choice("dawn-rain", "夜里是什么声音？", "夜来风雨声", "rain",
				opt("rain", "风雨", "rain"), opt("song", "踏歌", "song"), opt("drum", "战鼓", "drum"), opt("bell", "钟", "bell")),
			title("dawn-title", "这首诗叫什么？", "春眠不觉晓", "chunxiao",
				opt("jingyesi", "静夜思", "char"), opt("chunxiao", "春晓", "char"), opt("jiangxue", "江雪", "char"), opt("yonge", "咏鹅", "char")),
			author("dawn-author", "这首诗是谁写的？", "春眠不觉晓", "menghaoran",
				opt("libai", "李白", "char"), opt("wangzhihuan", "王之涣", "char"), opt("menghaoran", "孟浩然", "char"), opt("liuzongyuan", "柳宗元", "char")),
		}
	case "goose":
		return []Clue{
			{
				ID: "goose-recite", Kind: "recite", Prompt: "按顺序点出这四句", Line: "鹅鹅鹅", Speech: "鹅鹅鹅，曲项向天歌，白毛浮绿水，红掌拨清波",
				AnswerID: "done", Sequence: []string{"鹅鹅鹅", "曲项向天歌", "白毛浮绿水", "红掌拨清波"},
				Options: []Option{opt("鹅鹅鹅", "鹅鹅鹅", "line"), opt("曲项向天歌", "曲项向天歌", "line"), opt("红掌拨清波", "红掌拨清波", "line"), opt("白毛浮绿水", "白毛浮绿水", "line")},
			},
			couplet("goose-next", "下一句是哪一句？", "鹅鹅鹅", "曲项向天歌",
				opt("曲项向天歌", "曲项向天歌", "line"), opt("白毛浮绿水", "白毛浮绿水", "line"), opt("床前明月光", "床前明月光", "line"), opt("红掌拨清波", "红掌拨清波", "line")),
			fill("goose-water", "缺的字是哪个？", "白毛浮绿□", "水",
				opt("山", "山", "char"), opt("水", "水", "char"), opt("天", "天", "char"), opt("月", "月", "char")),
			choice("goose-palm", "红掌是什么颜色？", "红掌拨清波", "red",
				opt("green", "绿", "green"), opt("red", "红", "red"), opt("blue", "蓝", "blue"), opt("white", "白", "white")),
		}
	case "farm":
		return []Clue{
			fill("farm-noon", "缺的字是哪个？", "锄禾日当□", "午",
				opt("天", "天", "char"), opt("午", "午", "char"), opt("月", "月", "char"), opt("鹅", "鹅", "char")),
			fill("farm-soil", "缺的字是哪个？", "汗滴禾下□", "土",
				opt("土", "土", "char"), opt("水", "水", "char"), opt("火", "火", "char"), opt("石", "石", "char")),
			fill("farm-meal", "缺的字是哪个？", "谁知盘中□", "餐",
				opt("饭", "饭", "char"), opt("菜", "菜", "char"), opt("餐", "餐", "char"), opt("茶", "茶", "char")),
			title("farm-title", "这首诗叫什么？", "粒粒皆辛苦", "minnong",
				opt("chunxiao", "春晓", "char"), opt("yonge", "咏鹅", "char"), opt("minnong", "悯农", "char"), opt("jiangxue", "江雪", "char")),
		}
	case "tower":
		return []Clue{
			couplet("tower-sun", "下一句是哪一句？", "白日依山尽", "黄河入海流",
				opt("黄河入海流", "黄河入海流", "line"), opt("处处闻啼鸟", "处处闻啼鸟", "line"), opt("疑是地上霜", "疑是地上霜", "line"), opt("红掌拨清波", "红掌拨清波", "line")),
			couplet("tower-climb", "下一句是哪一句？", "欲穷千里目", "更上一层楼",
				opt("低头思故乡", "低头思故乡", "line"), opt("更上一层楼", "更上一层楼", "line"), opt("花落知多少", "花落知多少", "line"), opt("春风吹又生", "春风吹又生", "line")),
			choice("tower-river", "入海的是哪条河？", "黄河入海流", "yellow-river",
				opt("yangtze", "长江", "yangtze"), opt("yellow-river", "黄河", "yellow-river"), opt("pond", "小池", "pond"), opt("well", "水井", "well")),
			title("tower-title", "这首诗叫什么？", "欲穷千里目", "dengguanquelou",
				opt("jingyesi", "静夜思", "char"), opt("chunxiao", "春晓", "char"), opt("dengguanquelou", "登鹳雀楼", "char"), opt("hua", "画", "char")),
		}
	case "snow":
		return []Clue{
			choice("snow-bird", "这句里的鸟怎么样了？", "千山鸟飞绝", "gone",
				opt("many", "很多鸟", "bird"), opt("goose", "有一只鹅", "goose"), opt("gone", "飞光了", "gone"), opt("cicada", "有蝉", "cicada")),
			choice("snow-boat", "江上有什么？", "孤舟蓑笠翁", "boat",
				opt("boat", "孤舟", "boat"), opt("ship", "大船", "ship"), opt("cart", "马车", "cart"), opt("kite", "纸鸢", "kite")),
			choice("snow-weather", "下着什么？", "独钓寒江雪", "snow",
				opt("rain", "雨", "rain"), opt("sun", "日", "sun"), opt("snow", "雪", "snow"), opt("flower", "花", "flower")),
			title("snow-title", "这首诗叫什么？", "千山鸟飞绝", "jiangxue",
				opt("yonge", "咏鹅", "char"), opt("chunxiao", "春晓", "char"), opt("minnong", "悯农", "char"), opt("jiangxue", "江雪", "char")),
		}
	case "scroll":
		return []Clue{
			choice("scroll-mountain", "远看的山怎么样？", "远看山有色", "color",
				opt("silent", "没有声音", "silent"), opt("color", "有颜色", "color"), opt("gone", "看不见", "gone"), opt("snow", "全是雪", "snow")),
			choice("scroll-water", "近处听水，水怎么样？", "近听水无声", "silent",
				opt("loud", "很响", "loud"), opt("sing", "在唱歌", "sing"), opt("silent", "没有声音", "silent"), opt("hot", "是热水", "hot")),
			title("scroll-title", "这首诗叫什么？", "远看山有色", "hua",
				opt("jiangxue", "江雪", "char"), opt("hua", "画", "char"), opt("chunxiao", "春晓", "char"), opt("yonge", "咏鹅", "char")),
			choice("scroll-bird", "人走过来，画里的鸟会怎样？", "人来鸟不惊", "still",
				opt("fly", "飞走", "fly"), opt("sing", "唱歌", "sing"), opt("swim", "游泳", "swim"), opt("still", "还停着", "still")),
		}
	default:
		return nil
	}
}

func opt(id, label, visual string) Option {
	return Option{ID: id, Label: label, Visual: visual}
}

func choice(id, prompt, line, answer string, options ...Option) Clue {
	return Clue{ID: id, Kind: "choice", Prompt: prompt, Line: line, Speech: line, AnswerID: answer, Options: options}
}

func title(id, prompt, line, answer string, options ...Option) Clue {
	return Clue{ID: id, Kind: "title", Prompt: prompt, Line: line, Speech: line, AnswerID: answer, Options: options}
}

func author(id, prompt, line, answer string, options ...Option) Clue {
	return Clue{ID: id, Kind: "author", Prompt: prompt, Line: line, Speech: line, AnswerID: answer, Options: options}
}

func fill(id, prompt, line, answer string, options ...Option) Clue {
	return Clue{ID: id, Kind: "fill", Prompt: prompt, Line: line, Speech: line, AnswerID: answer, Options: options}
}

func couplet(id, prompt, line, answer string, options ...Option) Clue {
	return Clue{ID: id, Kind: "couplet", Prompt: prompt, Line: line, Speech: line, AnswerID: answer, Options: options}
}
