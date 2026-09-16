package quiz_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/english-server/internal/quiz"
)

func setup(t *testing.T) *quiz.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		`CREATE TABLE subjects (id INTEGER PRIMARY KEY, code TEXT)`,
		`CREATE TABLE modules (id INTEGER PRIMARY KEY, subject_id INTEGER, code TEXT, name TEXT, order_no INTEGER)`,
		`CREATE TABLE knowledge_points (id INTEGER PRIMARY KEY, module_id INTEGER, title TEXT, payload TEXT, order_no INTEGER)`,
		`CREATE TABLE english_assets (kp_id INTEGER PRIMARY KEY, sense_image_url TEXT, speech_audio_url TEXT)`,
		`INSERT INTO subjects VALUES (1,'english'),(2,'literacy')`,
		`INSERT INTO modules VALUES (1,1,'animals','Animals',0),(2,1,'greetings','Greetings',1),(3,2,'hanzi','汉字',0)`,
		`INSERT INTO knowledge_points VALUES
			(101,1,'apple','{"meaningZh":"苹果"}',1),
			(102,1,'banana','{"meaningZh":"香蕉"}',2),
			(103,1,'dog','{"meaningZh":"小狗"}',3),
			(104,1,'bird','{"meaningZh":"小鸟"}',4),
			(201,2,'hello','{"meaningZh":"你好"}',1),
			(202,2,'please','{"meaningZh":"请"}',2),
			(999,3,'山','{"meaningZh":"mountain"}',1)`,
		`INSERT INTO english_assets VALUES
			(101,'english/senses/101.png','english/speech/101.mp3'),
			(102,'english/senses/102.png','english/speech/102.mp3'),
			(103,'english/senses/103.png','english/speech/103.mp3'),
			(104,'','english/speech/104.mp3')`,
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	return quiz.NewService(db)
}

func TestGenerateListenUsesSameModuleAndSpeech(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	require.Equal(t, "listen", question.Type)
	require.Len(t, question.Options, 4)
	require.Equal(t, question.TargetID, question.Options[question.AnswerIndex].ID)
	require.Contains(t, []int64{101, 102, 103, 104}, question.TargetID)
	require.NotContains(t, []int64{201, 999}, question.TargetID)
	require.Contains(t, question.SpeechURL, "/api/v1/english/words/")
	require.Contains(t, question.SpeechURL, "/speech.mp3")
}

func TestGenerateLookSkipsAbstractGreetings(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "look", nil)
	require.NoError(t, err)
	require.Equal(t, "look", question.Type)
	require.Equal(t, "word", question.Visual.Kind)
	require.NotEmpty(t, question.Visual.Text)
	require.NotContains(t, []int64{201, 202}, question.TargetID)
}

func TestGenerateHonorsExcludeTargetIds(t *testing.T) {
	service := setup(t)
	first, err := service.Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	second, err := service.Generate(context.Background(), "listen", []int64{first.TargetID})
	require.NoError(t, err)
	require.NotEqual(t, first.TargetID, second.TargetID)
}

func TestGenerateRejectsUnknownType(t *testing.T) {
	_, err := setup(t).Generate(context.Background(), "blend", nil)
	require.ErrorIs(t, err, quiz.ErrInvalidType)
}

func TestGenerateIgnoresOtherSubjects(t *testing.T) {
	question, err := setup(t).Generate(context.Background(), "listen", nil)
	require.NoError(t, err)
	for _, option := range question.Options {
		require.NotEqual(t, int64(999), option.ID)
	}
}
