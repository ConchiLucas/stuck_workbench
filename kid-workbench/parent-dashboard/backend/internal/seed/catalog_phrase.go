package seed

import (
	"fmt"
)

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

func phraseModules() []moduleSpec {
	topics := []struct {
		code   string
		name   string
		prefix string
		items  []phraseItem
	}{
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
		{"class", "课堂", "pc", []phraseItem{
			ph("Sit down, please.", "请坐下。", []string{"请站起来。", "请举手。", "请安静。"}, "老师让大家坐下", ""),
			ph("Stand up, please.", "请站起来。", []string{"请坐下。", "请打开书。", "请合上书。"}, "老师让大家站起来", ""),
			ph("Listen to me.", "听我说。", []string{"看着我。", "跟我读。", "请安静。"}, "老师要开始讲课", ""),
			ph("Look at me.", "看着我。", []string{"听我说。", "举手。", "坐下。"}, "老师要你看着她", ""),
			ph("Open your book.", "打开书。", []string{"合上书本。", "站起来。", "坐下。"}, "开始读书了", ""),
			ph("Close your book.", "合上书本。", []string{"打开书。", "举手。", "站起来。"}, "书读完了", ""),
			ph("Raise your hand.", "举手。", []string{"坐下。", "站起来。", "安静。"}, "你想发言", ""),
			ph("Let's begin.", "我们开始吧。", []string{"再见。", "休息吧。", "请坐下。"}, "课要开始了", ""),
		}},
		{"daily", "日常", "pd", []phraseItem{
			ph("Thank you.", "谢谢你。", []string{"不客气。", "对不起。", "再见。"}, "别人帮了你", ""),
			ph("You're welcome.", "不客气。", []string{"谢谢你。", "对不起。", "再见。"}, "别人跟你说谢谢", "Thank you."),
			ph("Excuse me.", "对不起/打扰一下。", []string{"谢谢你。", "再见。", "你好。"}, "想借过或打扰别人", ""),
			ph("I'm sorry.", "我很抱歉。", []string{"谢谢你。", "不客气。", "没关系。"}, "不小心撞到别人", ""),
			ph("May I come in?", "我可以进来吗？", []string{"请坐下。", "请出去。", "请安静。"}, "想进教室", ""),
			ph("What's your name?", "你叫什么名字？", []string{"你好吗？", "再见。", "早上好。"}, "想认识新朋友", ""),
			ph("My name is Lily.", "我的名字是莉莉。", []string{"你叫什么名字？", "再见。", "早上好。"}, "别人问你叫什么", "What's your name?"),
			ph("See you tomorrow.", "明天见。", []string{"晚安。", "再见。", "早上好。"}, "放学要回家了", ""),
		}},
		{"feel", "感受", "pf", []phraseItem{
			ph("I like it.", "我喜欢。", []string{"我不喜欢。", "我饿了。", "我累了。"}, "看到喜欢的东西", ""),
			ph("I don't like it.", "我不喜欢。", []string{"我喜欢。", "我很好。", "我很高兴。"}, "不想吃某种食物", ""),
			ph("I'm hungry.", "我饿了。", []string{"我渴了。", "我累了。", "我很好。"}, "肚子咕咕叫", ""),
			ph("I'm thirsty.", "我渴了。", []string{"我饿了。", "我累了。", "我很好。"}, "口很干", ""),
			ph("Let's play.", "我们一起玩吧。", []string{"安静。", "坐下。", "再见。"}, "想约小朋友玩", ""),
			ph("Come here.", "过来。", []string{"等等我。", "再见。", "坐下。"}, "叫小朋友过来", ""),
			ph("Wait for me.", "等等我。", []string{"过来。", "再见。", "开始吧。"}, "你跑得比较慢", ""),
			ph("Be quiet.", "安静。", []string{"大声点。", "一起玩。", "站起来。"}, "教室太吵了", ""),
		}},
	}

	mods := make([]moduleSpec, 0, len(topics))
	for _, t := range topics {
		kps := make([]kpSpec, 0, len(t.items))
		for i, it := range t.items {
			diff := it.Diff
			if diff == 0 {
				diff = 1
			}
			kps = append(kps, kpSpec{
				Code: fmt.Sprintf("%s%03d", t.prefix, i+1), Title: it.En, Difficulty: diff,
				Payload: mustPayload(map[string]any{
					"kind": "phrase", "zh": it.Zh, "wrong": it.Wrong,
					"scene": it.Scene, "replyTo": it.ReplyTo,
				}),
			})
		}
		mods = append(mods, moduleSpec{Code: t.code, Name: t.name, Kps: kps})
	}
	return mods
}
