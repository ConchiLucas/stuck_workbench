package mastery_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-learning/mastery"
)

func TestLiteracySkillCodes(t *testing.T) {
	require.Equal(t, []string{
		mastery.SkillGlyphSense,
		mastery.SkillSenseChar,
		mastery.SkillWriteChar,
	}, mastery.LiteracySkills)
}

func TestPinyinSkillCodes(t *testing.T) {
	require.Equal(t, []string{
		mastery.SkillPinyinListen,
		mastery.SkillPinyinInWord,
		mastery.SkillPinyinShape,
		mastery.SkillPinyinBlend,
	}, mastery.PinyinSkills)
	require.Equal(t, mastery.PinyinSkills, mastery.SkillsForSubject("pinyin"))
	require.Equal(t, mastery.LiteracySkills, mastery.SkillsForSubject("literacy"))
	require.Nil(t, mastery.SkillsForSubject("math"))
}

func TestScienceSkillCodes(t *testing.T) {
	require.Equal(t, []string{
		mastery.SkillScienceChoice,
		mastery.SkillScienceMatch,
		mastery.SkillScienceSequence,
		mastery.SkillScienceLabel,
	}, mastery.ScienceSkills)
	require.Nil(t, mastery.SkillsForSubject("science"))
	require.Equal(t, mastery.SkillScienceChoice, mastery.SkillFromQuestionCode("science", "recognize"))
	require.Equal(t, mastery.SkillScienceChoice, mastery.SkillFromQuestionCode("science", "choice"))
	require.Equal(t, mastery.SkillScienceMatch, mastery.SkillFromQuestionCode("science", "match"))
	require.Equal(t, mastery.SkillScienceSequence, mastery.SkillFromQuestionCode("science", "sequence"))
	require.Equal(t, mastery.SkillScienceLabel, mastery.SkillFromQuestionCode("science", "label"))
	require.Equal(t, "选择题", mastery.SkillDisplayName("science", "choice"))
	require.Equal(t, "连线题", mastery.SkillDisplayName("science", "match"))
	require.Equal(t, "排序题", mastery.SkillDisplayName("science", "sequence"))
	require.Equal(t, "结构标注题", mastery.SkillDisplayName("science", "label"))
	require.True(t, mastery.SkillIsTracked("science", "observe", "match"))
	require.True(t, mastery.SkillIsTracked("science", "animal", "choice"))
	require.False(t, mastery.SkillIsTracked("science", "observe", "listen"))
	require.Equal(t, mastery.PoemSkills, mastery.SkillsForSubject("poem"))
	require.Equal(t, "选诗名", mastery.SkillDisplayName("poem", "title"))
	require.Equal(t, "补字", mastery.SkillDisplayName("poem", "fill"))
	require.Equal(t, "选下一句", mastery.SkillDisplayName("poem", "couplet"))
	require.Equal(t, "排顺序", mastery.SkillDisplayName("poem", "recite"))
	require.Equal(t, mastery.SkillPoemCouplet, mastery.SkillFromQuestionCode("poem", "nextline"))
	require.True(t, mastery.SkillIsTracked("poem", "poem50", "recite"))
	require.False(t, mastery.SkillIsTracked("poem", "poem50", "choice"))
	require.Empty(t, mastery.SkillsFor("logic", "playground"))
	require.True(t, mastery.SkillIsTracked("logic", "playground", "pattern"))
	require.True(t, mastery.SkillIsTracked("logic", "playground", "order"))
	require.False(t, mastery.SkillIsTracked("logic", "playground", "choice"))
	require.Equal(t, "找规律", mastery.SkillDisplayName("logic", "pattern"))
	require.Equal(t, "pattern", mastery.SkillFromQuestionCode("logic", "pattern"))
}

func TestSkillDisplayNameMatchesCurrentEnglishTypes(t *testing.T) {
	require.Equal(t, "听音选词", mastery.SkillDisplayName("english", "listen"))
	require.Equal(t, "看图选词", mastery.SkillDisplayName("english", "picture"))
	require.Equal(t, "组句子", mastery.SkillDisplayName("english", "build"))
	require.Equal(t, "写单词", mastery.SkillDisplayName("english", "type"))
	require.Equal(t, "读一读", mastery.SkillDisplayName("english", "read"))
	require.NotEqual(t, "单词认读", mastery.SkillDisplayName("english", "type"))
}

func TestEnglishSkillCodesAreSubjectAware(t *testing.T) {
	require.Equal(t, []string{"listen", "picture", "build", "type", "read"}, mastery.EnglishSkills)
	require.Equal(t, mastery.EnglishSkills, mastery.SkillsForSubject("english"))
	require.Equal(t, mastery.SkillEnglishListen, mastery.SkillFromQuestionCode("english", "listen"))
	require.Equal(t, mastery.SkillEnglishPicture, mastery.SkillFromQuestionCode("english", "picture"))
	require.Equal(t, mastery.SkillEnglishBuild, mastery.SkillFromQuestionCode("english", "build"))
	require.Equal(t, mastery.SkillEnglishType, mastery.SkillFromQuestionCode("english", "type"))
	require.Equal(t, mastery.SkillEnglishRead, mastery.SkillFromQuestionCode("english", "read"))
	require.Equal(t, mastery.SkillPinyinListen, mastery.SkillFromQuestionCode("pinyin", "listen"))
	require.Empty(t, mastery.SkillFromQuestionCode("literacy", "listen"))
}

func TestSkillsForMathModules(t *testing.T) {
	require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "add10"))
	require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "sub10"))
	require.Equal(t, []string{"find", "name"}, mastery.SkillsFor("math", "shape"))
	require.Empty(t, mastery.SkillsFor("math", "unknown"))
}

func TestSkillsForReturnsCopy(t *testing.T) {
	got := mastery.SkillsFor("math", "add10")
	got[0] = "changed"
	require.Equal(t, []string{"calc", "story"}, mastery.SkillsFor("math", "add10"))
}

func TestMathQuestionCodesMapToSkills(t *testing.T) {
	for _, code := range []string{"calc", "story", "find", "name"} {
		require.Equal(t, code, mastery.SkillFromQuestionCode("math", code))
	}
}

func TestPhraseSkillCodes(t *testing.T) {
	require.Equal(t, []string{"listen_zh", "listen_en", "scene", "reply"}, mastery.PhraseSkills)
	require.Equal(t, []string{"listen_zh", "listen_en", "scene"}, mastery.PhraseCoreSkills)
	require.Equal(t, mastery.PhraseCoreSkills, mastery.SkillsForSubject("phrase"))
	require.Equal(t, mastery.SkillPhraseReply, mastery.SkillFromQuestionCode("phrase", "reply"))
	require.Equal(t, "听一听", mastery.SkillDisplayName("phrase", "listen_zh"))
	require.Equal(t, "选句子", mastery.SkillDisplayName("phrase", "listen_en"))
	require.Equal(t, "什么时候说", mastery.SkillDisplayName("phrase", "scene"))
	require.Equal(t, "问与答", mastery.SkillDisplayName("phrase", "reply"))
	require.Equal(t, mastery.SkillPhraseListenZh, mastery.SkillFromQuestionCode("phrase", "listen_zh"))
	require.Empty(t, mastery.SkillFromQuestionCode("english", "listen_zh"))
	require.True(t, mastery.SkillIsTracked("phrase", "greet", "listen_zh"))
	require.True(t, mastery.SkillIsTracked("phrase", "greet", "reply"))
	require.False(t, mastery.SkillIsTracked("phrase", "greet", "listen"))
	require.False(t, mastery.SkillIsTracked("math", "shape", "calc"))
}

func TestChengyuSkillCodes(t *testing.T) {
	require.Equal(t, []string{"meaning", "pick", "pinyin", "example"}, mastery.ChengyuSkills)
	require.Equal(t, mastery.ChengyuSkills, mastery.SkillsForSubject("chengyu"))
	require.Equal(t, "听释义", mastery.SkillDisplayName("chengyu", "meaning"))
	require.Equal(t, "选成语", mastery.SkillDisplayName("chengyu", "pick"))
	require.Equal(t, "看拼音", mastery.SkillDisplayName("chengyu", "pinyin"))
	require.Equal(t, "看句子", mastery.SkillDisplayName("chengyu", "example"))
	require.Equal(t, mastery.SkillChengyuMeaning, mastery.SkillFromQuestionCode("chengyu", "meaning"))
	require.Empty(t, mastery.SkillFromQuestionCode("pinyin", "meaning"))
}

func TestSkillFromQuestionCode(t *testing.T) {
	require.Equal(t, "", mastery.SkillFromQuestionCode("literacy", "listen_glyph"))
	require.Equal(t, "", mastery.SkillFromQuestionCode("literacy", "listen1"))
	require.Equal(t, mastery.SkillGlyphSense, mastery.SkillFromQuestionCode("literacy", "glyph_sense"))
	require.Equal(t, mastery.SkillSenseChar, mastery.SkillFromQuestionCode("literacy", "sense_char"))
	require.Equal(t, mastery.SkillWriteChar, mastery.SkillFromQuestionCode("literacy", "write_char"))
	require.Equal(t, mastery.SkillPinyinInWord, mastery.SkillFromQuestionCode("pinyin", "inword"))
	require.Equal(t, mastery.SkillPinyinListen, mastery.SkillFromQuestionCode("pinyin", "listen"))
	require.Equal(t, mastery.SkillMathCalc, mastery.SkillFromQuestionCode("math", "calc"))
}

func TestRollupAllMastered(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusReviewDue,
	})
	require.Equal(t, mastery.StatusReviewDue, got)
}

func TestRollupAllMasteredNoDue(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusMastered,
	})
	require.Equal(t, mastery.StatusMastered, got)
}

func TestRollupAnyShaky(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusShaky,
	})
	require.Equal(t, mastery.StatusShaky, got)
}

func TestRollupLearningBeatsNotStarted(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusNotStarted,
		mastery.StatusLearning,
	})
	require.Equal(t, mastery.StatusLearning, got)
}

func TestRollupEmptyIsNotStarted(t *testing.T) {
	require.Equal(t, mastery.StatusNotStarted, mastery.RollupSkills(nil))
}

func TestRollupIncompleteNotFullyMastered(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusNotStarted,
	})
	require.Equal(t, mastery.StatusLearning, got)
}

func TestPinyinRequiredSkillsByModule(t *testing.T) {
	require.Equal(t, []string{"listen", "inword", "shape"}, mastery.SkillsFor("pinyin", "shengmu"))
	require.Equal(t, []string{"blend"}, mastery.SkillsFor("pinyin", "syllables"))
	require.False(t, mastery.IsSkillDone(mastery.RollupSkills([]mastery.Status{mastery.StatusMastered, mastery.StatusReviewDue, mastery.StatusNotStarted})))
}
