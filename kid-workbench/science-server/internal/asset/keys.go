package asset

import "fmt"

func GlyphKey(kpID int64) string {
	return fmt.Sprintf("science/glyphs/%d.png", kpID)
}

func GlyphVersionKey(kpID int64, version int) string {
	return fmt.Sprintf("science/glyphs/%d-v%d.png", kpID, version)
}

func SenseKey(kpID int64) string  { return fmt.Sprintf("science/senses/%d.png", kpID) }
func SpeechKey(kpID int64) string { return fmt.Sprintf("science/speech/%d.mp3", kpID) }

func SenseVersionKey(kpID int64, version int) string {
	return fmt.Sprintf("science/senses/%d-v%d.png", kpID, version)
}

func SpeechVersionKey(kpID int64, version int) string {
	return fmt.Sprintf("science/speech/%d-v%d.mp3", kpID, version)
}
