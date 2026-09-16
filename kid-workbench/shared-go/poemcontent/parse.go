package poemcontent

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"strings"
	"unicode"
)

func ParseWork(kpID int64, code, title, payload string) Work {
	var raw struct {
		Kind     string          `json:"kind"`
		WorkID   string          `json:"workId"`
		Author   string          `json:"author"`
		Dynasty  string          `json:"dynasty"`
		Edition  string          `json:"edition"`
		Citation string          `json:"citation"`
		Line1    string          `json:"line1"`
		Line2    string          `json:"line2"`
		Lines    json.RawMessage `json:"lines"`
	}
	_ = json.Unmarshal([]byte(payload), &raw)
	workID := strings.TrimSpace(raw.WorkID)
	if workID == "" {
		workID = strings.TrimSpace(code)
	}
	w := Work{
		WorkID: workID, KpID: kpID, Code: strings.TrimSpace(code), Title: strings.TrimSpace(title),
		Author: strings.TrimSpace(raw.Author), Dynasty: strings.TrimSpace(raw.Dynasty),
		Edition: strings.TrimSpace(raw.Edition), Citation: strings.TrimSpace(raw.Citation),
	}
	w.Lines = parseLines(workID, raw.Lines, raw.Line1, raw.Line2)
	return NormalizeWork(w)
}

func NormalizeWork(w Work) Work {
	w.WorkID = strings.TrimSpace(w.WorkID)
	if w.WorkID == "" {
		w.WorkID = strings.TrimSpace(w.Code)
	}
	if w.Edition == "" {
		w.Edition = Edition
	}
	lines := make([]Line, 0, len(w.Lines))
	for i, line := range w.Lines {
		text := strings.TrimSpace(line.Text)
		if text == "" {
			continue
		}
		ord := line.Ord
		if ord <= 0 {
			ord = i + 1
		}
		id := strings.TrimSpace(line.ID)
		if id == "" && w.WorkID != "" {
			id = LineID(w.WorkID, ord)
		}
		lines = append(lines, Line{ID: id, Ord: ord, Text: text})
	}
	w.Lines = lines
	return w
}

func parseLines(workID string, raw json.RawMessage, line1, line2 string) []Line {
	if len(raw) > 0 && string(raw) != "null" {
		var objects []Line
		if json.Unmarshal(raw, &objects) == nil && len(objects) > 0 && strings.TrimSpace(objects[0].Text) != "" {
			return objects
		}
		var texts []string
		if json.Unmarshal(raw, &texts) == nil {
			out := make([]Line, 0, len(texts))
			for i, t := range texts {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}
				ord := i + 1
				lines := out
				_ = lines
				out = append(out, Line{ID: LineID(workID, ord), Ord: ord, Text: t})
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	out := []Line{}
	if t := strings.TrimSpace(line1); t != "" {
		out = append(out, Line{ID: LineID(workID, 1), Ord: 1, Text: t})
	}
	if t := strings.TrimSpace(line2); t != "" {
		out = append(out, Line{ID: LineID(workID, 2), Ord: 2, Text: t})
	}
	return out
}

func sha256Short(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func LineOrd(id string) int {
	i := strings.LastIndex(id, ":L")
	if i < 0 {
		return 1
	}
	n := 0
	for _, r := range id[i+2:] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	if n < 1 {
		return 1
	}
	return n
}

func SpeechWAV(seed string) []byte {
	const sampleRate = 8000
	n := 2400
	h := sha256.Sum256([]byte(seed))
	freq := 220 + int(h[0])*2 + int(h[1])
	data := make([]byte, n*2)
	for i := 0; i < n; i++ {
		phase := 2 * 3.14159265 * float64(freq) * float64(i) / sampleRate
		amp := 0.18
		if i < 80 || i > n-80 {
			amp *= float64(min(i, n-i)) / 80
		}
		v := int16(amp * 30000 * sinApprox(phase))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(v))
	}
	buf := make([]byte, 44+len(data))
	copy(buf[0:], []byte("RIFF"))
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+len(data)))
	copy(buf[8:], []byte("WAVEfmt "))
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	binary.LittleEndian.PutUint16(buf[22:], 1)
	binary.LittleEndian.PutUint32(buf[24:], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:], sampleRate*2)
	binary.LittleEndian.PutUint16(buf[32:], 2)
	binary.LittleEndian.PutUint16(buf[34:], 16)
	copy(buf[36:], []byte("data"))
	binary.LittleEndian.PutUint32(buf[40:], uint32(len(data)))
	copy(buf[44:], data)
	return buf
}

func sinApprox(x float64) float64 {
	for x > 3.14159265 {
		x -= 6.2831853
	}
	for x < -3.14159265 {
		x += 6.2831853
	}
	x2 := x * x
	return x * (1 - x2/6 + x2*x2/120)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func FirstHan(s string) string {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return string(r)
		}
	}
	return s
}
