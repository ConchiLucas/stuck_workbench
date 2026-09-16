package pinyin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/pinyin"
)

func setupQuizService(t *testing.T) (*pinyin.Service, int64) {
	t.Helper()
	gdb, err := db.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))

	items := []pinyin.Asset{
		{KpID: 1, Letter: "b", ModuleCode: "initials", ModuleName: "声母", SoloText: "波", WordText: "爸", WordExamples: `["爸","八","不","白","北"]`, SoloSpeechURL: "/b-solo", WordSpeechURL: "/b-word", GlyphImageURL: "/b.png"},
		{KpID: 2, Letter: "p", ModuleCode: "initials", ModuleName: "声母", SoloText: "坡", WordText: "怕", SoloSpeechURL: "/p-solo", WordSpeechURL: "/p-word", GlyphImageURL: "/p.png"},
		{KpID: 3, Letter: "m", ModuleCode: "initials", ModuleName: "声母", SoloText: "摸", WordText: "妈", SoloSpeechURL: "/m-solo", WordSpeechURL: "/m-word", GlyphImageURL: "/m.png"},
		{KpID: 4, Letter: "f", ModuleCode: "initials", ModuleName: "声母", SoloText: "佛", WordText: "飞", SoloSpeechURL: "/f-solo", WordSpeechURL: "/f-word", GlyphImageURL: "/f.png"},
	}
	require.NoError(t, gdb.Create(&items).Error)

	require.NoError(t, gdb.Exec("UPDATE pinyin_syllable_assets SET speech_url = '/recording.mp3'").Error)
	var syllables int64
	require.NoError(t, gdb.Table("pinyin_syllable_assets").Where("enabled = ?", true).Count(&syllables).Error)
	require.GreaterOrEqual(t, syllables, int64(4))
	return pinyin.NewService(gdb, nil, nil, nil, nil), syllables
}

func TestGenerateQuizBuildsEveryTypeFromDatabaseMaterial(t *testing.T) {
	service, _ := setupQuizService(t)

	for _, quizType := range []string{"listen", "inword", "shape", "blend"} {
		t.Run(quizType, func(t *testing.T) {
			question, err := service.GenerateQuiz(context.Background(), quizType, nil)
			require.NoError(t, err)
			require.Equal(t, quizType, question.Type)
			require.NotEmpty(t, question.InstanceID)
			require.Len(t, question.Options, 4)
			require.GreaterOrEqual(t, question.AnswerIndex, 0)
			require.Less(t, question.AnswerIndex, 4)
			require.Equal(t, question.TargetID, question.Options[question.AnswerIndex].ID)

			seen := map[int64]bool{}
			for _, option := range question.Options {
				require.False(t, seen[option.ID], "option IDs must be unique")
				seen[option.ID] = true
			}
			if quizType == "shape" || quizType == "blend" {
				for _, option := range question.Options {
					require.True(t, option.SpeechURL != "" || option.SpeechText != "")
				}
			}
		})
	}
}

func TestGenerateQuizAvoidsExcludedTargetWhenAnotherIsAvailable(t *testing.T) {
	service, _ := setupQuizService(t)
	first, err := service.GenerateQuiz(context.Background(), "listen", nil)
	require.NoError(t, err)

	second, err := service.GenerateQuiz(context.Background(), "listen", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateQuizRejectsUnknownType(t *testing.T) {
	service, _ := setupQuizService(t)
	_, err := service.GenerateQuiz(context.Background(), "unknown", nil)
	require.ErrorIs(t, err, pinyin.ErrInvalidQuizType)
}

func TestGenerateInwordQuizPicksFromExampleWords(t *testing.T) {
	service, _ := setupQuizService(t)
	allowed := map[string]bool{"爸": true, "八": true, "不": true, "白": true, "北": true}
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		question, err := service.GenerateQuiz(context.Background(), "inword", []int64{2, 3, 4})
		require.NoError(t, err)
		require.Equal(t, int64(1), question.TargetID)
		require.True(t, allowed[question.Visual.Text], question.Visual.Text)
		require.Equal(t, question.Visual.Text, question.SpeechText)
		require.Contains(t, question.SpeechURL, "/speech/")
		if question.Visual.Text == "爸" {
			require.Contains(t, question.SpeechURL, "/speech/word.mp3")
		} else {
			require.Regexp(t, `/speech/word-[1-4]\.mp3`, question.SpeechURL)
		}
		seen[question.Visual.Text] = true
	}
	require.Greater(t, len(seen), 1)
}
