package seed

import (
	"encoding/json"
	"fmt"
)

type englishPayload struct {
	MeaningZh        string `json:"meaningZh"`
	Phonetic         string `json:"phonetic,omitempty"`
	PartOfSpeech     string `json:"partOfSpeech,omitempty"`
	Example          string `json:"example,omitempty"`
	ExampleMeaningZh string `json:"exampleMeaningZh,omitempty"`
}

type englishWordSpec struct {
	Word      string
	MeaningZh string
}

type englishTopicSpec struct {
	Code  string
	Name  string
	Words []englishWordSpec
}

func ew(word, meaningZh string) englishWordSpec {
	return englishWordSpec{Word: word, MeaningZh: meaningZh}
}

func englishModules() []moduleSpec {
	topics := []englishTopicSpec{
		{"animals", "Animals", []englishWordSpec{ew("cat", "猫"), ew("dog", "狗"), ew("bird", "鸟"), ew("fish", "鱼"), ew("rabbit", "兔子"), ew("tiger", "老虎"), ew("lion", "狮子"), ew("bear", "熊"), ew("monkey", "猴子"), ew("panda", "熊猫")}},
		{"colors", "Colors", []englishWordSpec{ew("red", "红色"), ew("blue", "蓝色"), ew("green", "绿色"), ew("yellow", "黄色"), ew("pink", "粉色"), ew("black", "黑色"), ew("white", "白色"), ew("orange", "橙色"), ew("purple", "紫色"), ew("brown", "棕色")}},
		{"numbers", "Numbers", []englishWordSpec{ew("one", "一"), ew("two", "二"), ew("three", "三"), ew("four", "四"), ew("five", "五"), ew("six", "六"), ew("seven", "七"), ew("eight", "八"), ew("nine", "九"), ew("ten", "十")}},
		{"food", "Food", []englishWordSpec{ew("apple", "苹果"), ew("banana", "香蕉"), ew("bread", "面包"), ew("milk", "牛奶"), ew("egg", "鸡蛋"), ew("rice", "米饭"), ew("cake", "蛋糕"), ew("juice", "果汁"), ew("soup", "汤"), ew("candy", "糖果")}},
		{"family", "Family", []englishWordSpec{ew("mom", "妈妈"), ew("dad", "爸爸"), ew("baby", "婴儿"), ew("grandpa", "爷爷"), ew("grandma", "奶奶"), ew("brother", "兄弟"), ew("sister", "姐妹"), ew("uncle", "叔叔"), ew("aunt", "阿姨"), ew("cousin", "堂（表）兄弟姐妹")}},
		{"body", "Body", []englishWordSpec{ew("head", "头"), ew("eye", "眼睛"), ew("ear", "耳朵"), ew("nose", "鼻子"), ew("mouth", "嘴巴"), ew("hand", "手"), ew("foot", "脚"), ew("arm", "手臂"), ew("leg", "腿"), ew("hair", "头发")}},
		{"fruits", "Fruits", []englishWordSpec{ew("grape", "葡萄"), ew("peach", "桃子"), ew("pear", "梨"), ew("orange", "橙子"), ew("lemon", "柠檬"), ew("melon", "瓜"), ew("cherry", "樱桃"), ew("mango", "芒果"), ew("kiwi", "猕猴桃"), ew("berry", "浆果")}},
		{"weather", "Weather", []englishWordSpec{ew("sunny", "晴朗的"), ew("rainy", "下雨的"), ew("cloudy", "多云的"), ew("windy", "有风的"), ew("snowy", "下雪的"), ew("hot", "热的"), ew("cold", "冷的"), ew("warm", "温暖的"), ew("cool", "凉爽的"), ew("storm", "暴风雨")}},
		{"toys", "Toys", []englishWordSpec{ew("ball", "球"), ew("doll", "玩偶"), ew("car", "小汽车"), ew("block", "积木"), ew("kite", "风筝"), ew("train", "火车"), ew("puzzle", "拼图"), ew("robot", "机器人"), ew("balloon", "气球"), ew("slide", "滑梯")}},
		{"school", "School", []englishWordSpec{ew("book", "书"), ew("pen", "钢笔"), ew("pencil", "铅笔"), ew("bag", "书包"), ew("desk", "课桌"), ew("chair", "椅子"), ew("teacher", "老师"), ew("student", "学生"), ew("class", "班级"), ew("school", "学校")}},
		{"clothes", "Clothes", []englishWordSpec{ew("shirt", "衬衫"), ew("pants", "裤子"), ew("dress", "连衣裙"), ew("hat", "帽子"), ew("shoe", "鞋"), ew("sock", "袜子"), ew("coat", "外套"), ew("scarf", "围巾"), ew("glove", "手套"), ew("skirt", "裙子")}},
		{"actions", "Actions", []englishWordSpec{ew("run", "跑"), ew("jump", "跳"), ew("walk", "走"), ew("sit", "坐"), ew("stand", "站"), ew("eat", "吃"), ew("drink", "喝"), ew("sleep", "睡觉"), ew("read", "阅读"), ew("write", "写")}},
		{"places", "Places", []englishWordSpec{ew("home", "家"), ew("park", "公园"), ew("zoo", "动物园"), ew("shop", "商店"), ew("farm", "农场"), ew("beach", "海滩"), ew("library", "图书馆"), ew("museum", "博物馆"), ew("hospital", "医院"), ew("cinema", "电影院")}},
		{"transport", "Transport", []englishWordSpec{ew("bus", "公交车"), ew("bike", "自行车"), ew("plane", "飞机"), ew("boat", "小船"), ew("train", "火车"), ew("taxi", "出租车"), ew("truck", "卡车"), ew("subway", "地铁"), ew("helicopter", "直升机"), ew("ship", "轮船")}},
		{"shapes", "Shapes", []englishWordSpec{ew("circle", "圆形"), ew("square", "正方形"), ew("triangle", "三角形"), ew("star", "星形"), ew("heart", "心形"), ew("oval", "椭圆形"), ew("rectangle", "长方形"), ew("diamond", "菱形"), ew("cross", "十字形"), ew("arrow", "箭头")}},
		{"time", "Time", []englishWordSpec{ew("morning", "早晨"), ew("noon", "中午"), ew("afternoon", "下午"), ew("evening", "傍晚"), ew("night", "夜晚"), ew("today", "今天"), ew("yesterday", "昨天"), ew("tomorrow", "明天"), ew("week", "星期"), ew("year", "年")}},
		{"feelings", "Feelings", []englishWordSpec{ew("happy", "开心的"), ew("sad", "难过的"), ew("angry", "生气的"), ew("tired", "疲倦的"), ew("hungry", "饥饿的"), ew("thirsty", "口渴的"), ew("scared", "害怕的"), ew("brave", "勇敢的"), ew("kind", "友善的"), ew("funny", "有趣的")}},
		{"nature", "Nature", []englishWordSpec{ew("tree", "树"), ew("flower", "花"), ew("grass", "草"), ew("river", "河流"), ew("mountain", "山"), ew("sun", "太阳"), ew("moon", "月亮"), ew("star", "星星"), ew("cloud", "云"), ew("rain", "雨")}},
		{"jobs", "Jobs", []englishWordSpec{ew("doctor", "医生"), ew("nurse", "护士"), ew("chef", "厨师"), ew("pilot", "飞行员"), ew("driver", "司机"), ew("farmer", "农民"), ew("singer", "歌手"), ew("dancer", "舞者"), ew("police", "警察"), ew("firefighter", "消防员")}},
		{"greetings", "Greetings", []englishWordSpec{ew("hello", "你好"), ew("hi", "嗨"), ew("bye", "再见"), ew("thanks", "谢谢"), ew("please", "请"), ew("sorry", "对不起"), ew("yes", "是"), ew("no", "不"), ew("ok", "好的"), ew("welcome", "欢迎")}},
	}

	mods := make([]moduleSpec, 0, len(topics))
	for i, topic := range topics {
		difficulty := 1 + i/7
		if difficulty > 3 {
			difficulty = 3
		}
		kps := make([]kpSpec, 0, len(topic.Words))
		for j, word := range topic.Words {
			payload, err := json.Marshal(englishPayload{MeaningZh: word.MeaningZh})
			if err != nil {
				panic(err)
			}
			kps = append(kps, kpSpec{
				Code: fmt.Sprintf("e%d%03d", i, j+1), Title: word.Word,
				Payload: string(payload), Difficulty: difficulty,
			})
		}
		mods = append(mods, moduleSpec{Code: topic.Code, Name: topic.Name, Kps: kps})
	}
	return mods
}
