package asset

import "fmt"

func GlyphKey(kpID int64) string {
	return fmt.Sprintf("literacy/glyphs/%d.png", kpID)
}

func SenseKey(kpID int64) string {
	return fmt.Sprintf("literacy/senses/%d.png", kpID)
}

func SpeechKey(kpID int64) string {
	return fmt.Sprintf("literacy/speech/%d.mp3", kpID)
}
