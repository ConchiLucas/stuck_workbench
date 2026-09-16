package logiccontent

import (
	"fmt"
	"strings"
)

func RenderSVG(o Object) []byte {
	fill := o.Fill
	if fill == "" {
		fill = "#6b5a86"
	}
	inner := glyphMarkup(o.Glyph, fill, o.Rotate, o.Count)
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">%s</svg>`, inner))
}

func glyphMarkup(glyph, fill string, rotate, count int) string {
	if count < 1 {
		count = 1
	}
	shape := oneShape(glyph, fill)
	if rotate != 0 {
		shape = fmt.Sprintf(`<g transform="rotate(%d 32 32)">%s</g>`, rotate, shape)
	}
	if count == 1 {
		return shape
	}
	if count > 4 {
		count = 4
	}
	parts := []string{}
	positions := [][2]int{{16, 20}, {48, 20}, {32, 46}}
	if count == 2 {
		positions = [][2]int{{20, 32}, {44, 32}}
	}
	if count == 4 {
		positions = [][2]int{{18, 18}, {46, 18}, {18, 46}, {46, 46}}
	}
	for i := 0; i < count; i++ {
		p := positions[i]
		parts = append(parts, fmt.Sprintf(`<g transform="translate(%d %d) scale(0.42) translate(-32 -32)">%s</g>`, p[0], p[1], oneShape(glyph, fill)))
	}
	return strings.Join(parts, "")
}

func oneShape(glyph, fill string) string {
	switch glyph {
	case "circle":
		return fmt.Sprintf(`<circle cx="32" cy="32" r="18" fill="%s"/>`, fill)
	case "square":
		return fmt.Sprintf(`<rect x="14" y="14" width="36" height="36" rx="4" fill="%s"/>`, fill)
	case "triangle":
		return fmt.Sprintf(`<polygon points="32,12 52,50 12,50" fill="%s"/>`, fill)
	case "star":
		return fmt.Sprintf(`<polygon points="32,10 38,26 56,26 42,36 48,52 32,42 16,52 22,36 8,26 26,26" fill="%s"/>`, fill)
	case "diamond":
		return fmt.Sprintf(`<polygon points="32,10 54,32 32,54 10,32" fill="%s"/>`, fill)
	case "heart":
		return fmt.Sprintf(`<path fill="%s" d="M32 52 C14 38 10 24 18 16 c6-6 14-2 14 6 0-8 8-12 14-6 8 8 4 22-14 36z"/>`, fill)
	case "apple":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="32" cy="36" rx="16" ry="18"/><path d="M32 18c0-8 8-10 10-4" fill="none" stroke="#3f7a3a" stroke-width="3"/><ellipse cx="28" cy="22" rx="6" ry="3" fill="#3f7a3a"/></g>`, fill)
	case "banana":
		return fmt.Sprintf(`<path fill="%s" d="M16 18c18 2 34 18 30 34-16-8-28-20-30-34z"/>`, fill)
	case "pear":
		return fmt.Sprintf(`<g fill="%s"><circle cx="32" cy="24" r="10"/><ellipse cx="32" cy="42" rx="16" ry="14"/><path d="M32 12v8" fill="none" stroke="#3f7a3a" stroke-width="3"/></g>`, fill)
	case "grape":
		return fmt.Sprintf(`<g fill="%s"><circle cx="24" cy="28" r="8"/><circle cx="40" cy="28" r="8"/><circle cx="32" cy="42" r="8"/></g>`, fill)
	case "carrot":
		return fmt.Sprintf(`<g><polygon points="32,18 48,52 16,52" fill="%s"/><path d="M24 16h16" stroke="#3f7a3a" stroke-width="4"/></g>`, fill)
	case "cat":
		return fmt.Sprintf(`<g fill="%s"><circle cx="32" cy="36" r="16"/><polygon points="16,28 18,12 28,24"/><polygon points="48,28 46,12 36,24"/><circle cx="26" cy="34" r="3" fill="#fff"/><circle cx="38" cy="34" r="3" fill="#fff"/></g>`, fill)
	case "dog":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="32" cy="36" rx="16" ry="14"/><ellipse cx="18" cy="28" rx="8" ry="10"/><ellipse cx="46" cy="28" rx="8" ry="10"/><circle cx="26" cy="36" r="3" fill="#fff"/><circle cx="38" cy="36" r="3" fill="#fff"/></g>`, fill)
	case "bird":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="30" cy="34" rx="16" ry="10"/><polygon points="46,34 58,28 46,40"/><circle cx="22" cy="32" r="3" fill="#fff"/></g>`, fill)
	case "fish":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="28" cy="32" rx="16" ry="10"/><polygon points="44,32 58,20 58,44"/><circle cx="20" cy="30" r="3" fill="#fff"/></g>`, fill)
	case "car":
		return fmt.Sprintf(`<g fill="%s"><rect x="10" y="28" width="44" height="16" rx="4"/><rect x="18" y="18" width="24" height="12" rx="3"/><circle cx="20" cy="46" r="6" fill="#333"/><circle cx="44" cy="46" r="6" fill="#333"/></g>`, fill)
	case "bus":
		return fmt.Sprintf(`<g fill="%s"><rect x="8" y="18" width="48" height="26" rx="4"/><rect x="14" y="22" width="10" height="10" fill="#fff"/><rect x="28" y="22" width="10" height="10" fill="#fff"/><rect x="42" y="22" width="8" height="10" fill="#fff"/><circle cx="20" cy="48" r="6" fill="#333"/><circle cx="46" cy="48" r="6" fill="#333"/></g>`, fill)
	case "bike":
		return fmt.Sprintf(`<g fill="none" stroke="%s" stroke-width="3"><circle cx="18" cy="42" r="10"/><circle cx="46" cy="42" r="10"/><path d="M18 42 L32 22 L46 42 M32 22 L28 16"/></g>`, fill)
	case "skate":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="32" cy="28" rx="16" ry="10"/><rect x="12" y="40" width="40" height="6" rx="3"/><circle cx="20" cy="50" r="4"/><circle cx="32" cy="50" r="4"/><circle cx="44" cy="50" r="4"/></g>`, fill)
	case "tree":
		return fmt.Sprintf(`<g><circle cx="32" cy="26" r="16" fill="%s"/><rect x="28" y="38" width="8" height="16" fill="#8b5a2b"/></g>`, fill)
	case "flower":
		return fmt.Sprintf(`<g fill="%s"><circle cx="32" cy="20" r="8"/><circle cx="20" cy="32" r="8"/><circle cx="44" cy="32" r="8"/><circle cx="32" cy="44" r="8"/><circle cx="32" cy="32" r="6" fill="#f4d35e"/></g>`, fill)
	case "sun":
		return fmt.Sprintf(`<g fill="%s"><circle cx="32" cy="32" r="12"/><g stroke="%s" stroke-width="3"><path d="M32 8v8M32 48v8M8 32h8M48 32h8M14 14l6 6M44 44l6 6M14 50l6-6M44 20l6-6"/></g></g>`, fill, fill)
	case "leaf":
		return fmt.Sprintf(`<path fill="%s" d="M32 12c16 10 20 28 0 40C12 40 16 22 32 12z"/>`, fill)
	case "snow":
		return fmt.Sprintf(`<g fill="none" stroke="%s" stroke-width="3"><path d="M32 10v44M14 20l36 24M50 20 14 44M18 32h28"/></g>`, fill)
	case "arrow":
		// Tip at +x so rotate 0 reads as 向右; 90 clockwise is 向下.
		return fmt.Sprintf(`<polygon points="52,32 24,16 24,24 8,24 8,40 24,40 24,48" fill="%s"/>`, fill)
	case "giraffe":
		return fmt.Sprintf(`<g fill="%s"><rect x="28" y="10" width="8" height="28" rx="3"/><circle cx="36" cy="12" r="8"/><ellipse cx="30" cy="48" rx="16" ry="10"/></g>`, fill)
	case "mouse":
		return fmt.Sprintf(`<g fill="%s"><circle cx="22" cy="24" r="10"/><circle cx="42" cy="24" r="10"/><ellipse cx="32" cy="38" rx="16" ry="12"/><circle cx="26" cy="36" r="3" fill="#fff"/><circle cx="38" cy="36" r="3" fill="#fff"/></g>`, fill)
	case "rabbit":
		return fmt.Sprintf(`<g fill="%s"><ellipse cx="24" cy="16" rx="6" ry="16"/><ellipse cx="40" cy="16" rx="6" ry="16"/><circle cx="32" cy="38" r="16"/></g>`, fill)
	case "one":
		return numberGlyph("1", fill)
	case "two":
		return numberGlyph("2", fill)
	case "three":
		return numberGlyph("3", fill)
	case "four":
		return numberGlyph("4", fill)
	default:
		return fmt.Sprintf(`<circle cx="32" cy="32" r="18" fill="%s"/>`, fill)
	}
}

func numberGlyph(text, fill string) string {
	return fmt.Sprintf(`<text x="32" y="42" text-anchor="middle" font-size="32" font-family="system-ui,sans-serif" font-weight="700" fill="%s">%s</text>`, fill, text)
}
