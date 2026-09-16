package asset

import "fmt"

func GlyphKey(kpID int64) string  { return fmt.Sprintf("english/glyphs/%d.png", kpID) }
func SenseKey(kpID int64) string  { return fmt.Sprintf("english/senses/%d.png", kpID) }
func SpeechKey(kpID int64) string { return fmt.Sprintf("english/speech/%d.mp3", kpID) }
