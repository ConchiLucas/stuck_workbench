package service

import (
	"encoding/json"
	"fmt"
)

type versionHistory struct{ Title, Code, Stem, Options, Visual, Speech, Answer string }

func frozenHistory(raw string) (versionHistory, error) {
	var s struct {
		SchemaVersion                                                              int
		SubjectCode, TargetText, QuestionType, Prompt, AnswerOptionID, Interaction string
		Stem                                                                       struct {
			Text         string
			Image, Audio *struct{ RevisionID, Kind, SHA256 string }
		}
		Options []struct {
			ID, Text     string
			KpID         int64
			Image, Audio *struct{ RevisionID, Kind, SHA256 string }
		}
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return versionHistory{}, err
	}
	if s.SchemaVersion == 2 && s.SubjectCode == "literacy" && s.QuestionType == "write_char" && s.Interaction == "handwriting" && len(s.Options) == 0 {
		audio := ""
		if s.Stem.Audio != nil {
			audio = "/literacy/api/v1/literacy/material-revisions/" + s.Stem.Audio.RevisionID + "/media/" + s.Stem.Audio.Kind
		}
		speech, _ := json.Marshal(map[string]string{"audio": audio})
		return versionHistory{s.TargetText, s.QuestionType, s.Prompt, "[]", "{}", string(speech), "{}"}, nil
	}
	if (s.SchemaVersion != 1 && s.SchemaVersion != 2) || s.SubjectCode != "literacy" || len(s.Options) != 4 || s.AnswerOptionID == "" || (s.QuestionType != "glyph_sense" && s.QuestionType != "sense_char") {
		return versionHistory{}, fmt.Errorf("generated question snapshot missing or invalid")
	}
	media := func(ref *struct{ RevisionID, Kind, SHA256 string }) string {
		if ref == nil {
			return ""
		}
		return "/literacy/api/v1/literacy/material-revisions/" + ref.RevisionID + "/media/" + ref.Kind
	}
	opts := []map[string]any{}
	answer := -1
	seen := map[string]bool{}
	for i, o := range s.Options {
		if o.ID == "" || seen[o.ID] {
			return versionHistory{}, fmt.Errorf("invalid snapshot options")
		}
		seen[o.ID] = true
		opts = append(opts, map[string]any{"id": o.ID, "kpId": o.KpID, "label": o.Text, "image": media(o.Image), "audio": media(o.Audio)})
		if o.ID == s.AnswerOptionID {
			answer = i
		}
	}
	if answer < 0 {
		return versionHistory{}, fmt.Errorf("snapshot answer missing")
	}
	options, _ := json.Marshal(opts)
	ans, _ := json.Marshal(map[string]int{"index": answer})
	visual, _ := json.Marshal(map[string]string{"kind": "char", "text": s.Stem.Text, "image": media(s.Stem.Image)})
	speech, _ := json.Marshal(map[string]string{"audio": media(s.Stem.Audio)})
	return versionHistory{s.TargetText, s.QuestionType, s.Prompt, string(options), string(visual), string(speech), string(ans)}, nil
}
func orderedHistoryOptions(raw, orderValue string) (string, error) {
	if orderValue == "" {
		return raw, nil
	}
	var original []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &original); err != nil {
		return "", err
	}
	order := parsePicks(orderValue)
	if len(order) != len(original) {
		return "", fmt.Errorf("invalid option order")
	}
	reordered := make([]json.RawMessage, len(order))
	seen := map[int]bool{}
	for i, j := range order {
		if j < 0 || j >= len(original) || seen[j] {
			return "", fmt.Errorf("invalid option order")
		}
		seen[j] = true
		reordered[i] = original[j]
	}
	b, _ := json.Marshal(reordered)
	return string(b), nil
}
