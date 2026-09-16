package pinyin_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportHumanPackStoresSoloSyllableAndExampleChars(t *testing.T) {
	svc, store, _ := setupPinyinSpeechService(t)
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "hsk"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "syllabs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-bo1.mp3"), []byte("solo-bo"), 0o644))
	for _, word := range []string{"包", "八", "不", "白", "北"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "hsk", "cmn-"+word+".mp3"), []byte("hsk-"+word), 0o644))
	}

	res, err := svc.ImportHumanPack(context.Background(), "shengmu", root)
	require.NoError(t, err)
	require.Equal(t, 6, res.Generated)
	require.Equal(t, 0, res.Failed)
	require.Equal(t, []byte("solo-bo"), store.files["pinyin/speech/101/solo.mp3"])
	require.Equal(t, []byte("hsk-包"), store.files["pinyin/speech/101/word.mp3"])
	require.Equal(t, []byte("hsk-八"), store.files["pinyin/speech/101/word-1.mp3"])

	list, err := svc.List("table")
	require.NoError(t, err)
	item := list.Items[0]
	require.Contains(t, item.SoloSpeechURL, "/speech/solo.mp3")
	require.Contains(t, item.WordExampleSpeechURLs["八"], "/speech/word-1.mp3")
	require.Contains(t, item.WordExampleSpeechURLs["包"], "/speech/word.mp3")
}

func TestImportHumanPackFallsBackToSyllableWhenHskCharMissing(t *testing.T) {
	svc, store, _ := setupPinyinYunmuService(t)
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "hsk"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "syllabs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-a1.mp3"), []byte("solo-a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-ma1.mp3"), []byte("syl-妈"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "syllabs", "cmn-ba4.mp3"), []byte("syl-爸"), 0o644))
	for _, word := range []string{"他", "大", "马"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "hsk", "cmn-"+word+".mp3"), []byte("hsk-"+word), 0o644))
	}

	res, err := svc.ImportHumanPack(context.Background(), "yunmu", root)
	require.NoError(t, err)
	require.Equal(t, 0, res.Failed)
	require.Equal(t, []byte("solo-a"), store.files["pinyin/speech/201/solo.mp3"])
	require.Equal(t, []byte("syl-妈"), store.files["pinyin/speech/201/word.mp3"])
	require.Equal(t, []byte("syl-爸"), store.files["pinyin/speech/201/word-1.mp3"])
	require.Equal(t, []byte("hsk-他"), store.files["pinyin/speech/201/word-2.mp3"])
}

func TestImportHumanPackRequiresModuleAndDir(t *testing.T) {
	svc, _, _ := setupPinyinSpeechService(t)
	_, err := svc.ImportHumanPack(context.Background(), "", t.TempDir())
	require.Error(t, err)
	_, err = svc.ImportHumanPack(context.Background(), "shengmu", "")
	require.Error(t, err)
}
