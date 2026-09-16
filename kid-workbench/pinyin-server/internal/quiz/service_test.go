package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/pinyin-server/internal/quiz"
)

func setupQuizService(t *testing.T) *quiz.Service { return quiz.NewService(setupQuizDB(t)) }

func setupQuizDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, statement := range []string{
		`CREATE TABLE pinyin_assets (
			kp_id INTEGER PRIMARY KEY, letter TEXT, module_code TEXT, module_name TEXT,
			module_order INTEGER, kp_order INTEGER, solo_text TEXT, word_text TEXT, word_examples TEXT,
			solo_speech_url TEXT, word_speech_url TEXT, glyph_image_url TEXT
		)`,
		`CREATE TABLE pinyin_syllable_assets (
			id INTEGER PRIMARY KEY AUTOINCREMENT, initial_text TEXT, final_text TEXT, tone INTEGER,
			syllable_text TEXT, speech_text TEXT, speech_url TEXT, enabled INTEGER
		)`,
		`INSERT INTO pinyin_assets VALUES
			(1, 'b', 'shengmu', '声母', 1, 1, '波', '爸', '["爸","八","不","白","北"]', '/b-solo', '/b-word', '/b.png'),
			(2, 'p', 'shengmu', '声母', 1, 2, '坡', '怕', '', '/p-solo', '/p-word', '/p.png'),
			(3, 'm', 'shengmu', '声母', 1, 3, '摸', '妈', '', '/m-solo', '/m-word', '/m.png'),
			(4, 'f', 'shengmu', '声母', 1, 4, '佛', '飞', '', '/f-solo', '/f-word', '/f.png')`,
		`INSERT INTO pinyin_syllable_assets (initial_text, final_text, tone, syllable_text, speech_text, speech_url, enabled) VALUES
			('b', 'ā', 1, 'bā', '八', '/api/v1/pinyin/syllables/1/speech.mp3', 1), ('p', 'ā', 1, 'pā', '趴', '/api/v1/pinyin/syllables/2/speech.mp3', 1),
			('m', 'ā', 1, 'mā', '妈', '/api/v1/pinyin/syllables/3/speech.mp3', 1), ('f', 'ā', 1, 'fā', '发', '/api/v1/pinyin/syllables/4/speech.mp3', 1)`,
	} {
		require.NoError(t, database.Exec(statement).Error)
	}
	return database
}

func TestGenerateBuildsEveryTypeFromDatabaseMaterial(t *testing.T) {
	service := setupQuizService(t)
	for _, quizType := range []string{"listen", "inword", "shape", "blend"} {
		t.Run(quizType, func(t *testing.T) {
			question, err := service.Generate(context.Background(), quizType, nil)
			require.NoError(t, err)
			require.Equal(t, quizType, question.Type)
			require.NotEmpty(t, question.InstanceID)
			require.Len(t, question.Options, 4)
			require.GreaterOrEqual(t, question.AnswerIndex, 0)
			require.Less(t, question.AnswerIndex, 4)
			require.Equal(t, question.TargetID, question.Options[question.AnswerIndex].ID)
			if quizType == "shape" || quizType == "blend" {
				for _, option := range question.Options {
					require.True(t, option.SpeechURL != "" || option.SpeechText != "")
				}
			}
			if quizType == "listen" {
				require.Contains(t, question.SpeechURL, "/speech/solo.mp3")
			}
		})
	}
}

func TestGenerateAvoidsExcludedTargetWhenAnotherIsAvailable(t *testing.T) {
	service := setupQuizService(t)
	first, err := service.Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "listen", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	service := setupQuizService(t)
	_, err := service.Generate(context.Background(), "unknown", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}

func TestGenerateInwordPicksFromExampleWords(t *testing.T) {
	service := setupQuizService(t)
	allowed := map[string]bool{"爸": true, "八": true, "不": true, "白": true, "北": true}
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		question, err := service.Generate(context.Background(), "inword", []int64{2, 3, 4})
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

func TestBlendRejectsUnrecordedMaterials(t *testing.T) {
	database := setupQuizDB(t)
	require.NoError(t, database.Exec("UPDATE pinyin_syllable_assets SET speech_url='' WHERE id=1").Error)
	_, err := quiz.NewService(database).Generate(context.Background(), "blend", nil)
	require.ErrorIs(t, err, quiz.ErrNoMaterial)
}
