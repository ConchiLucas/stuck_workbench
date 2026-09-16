package asset

import (
	"fmt"
	"strconv"
	"strings"
)

func GlyphKey(kpID int64) string {
	return fmt.Sprintf("pinyin/glyphs/%d.png", kpID)
}

func SpeechKey(kpID int64, kind string) string {
	return fmt.Sprintf("pinyin/speech/%d/%s.mp3", kpID, kind)
}

func ValidSpeechKind(kind string) bool {
	if kind == "solo" || kind == "word" {
		return true
	}
	rest, ok := strings.CutPrefix(kind, "word-")
	if !ok {
		return false
	}
	n, err := strconv.Atoi(rest)
	return err == nil && n >= 1 && n <= 4 && strconv.Itoa(n) == rest
}
