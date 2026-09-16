package catalog

func StageFor(moduleCode string, a, b int) string {
	switch moduleCode {
	case "add10":
		sum := a + b
		switch {
		case a < 1 || b < 1 || sum > 20:
			return ""
		case sum <= 5:
			return "within5"
		case sum <= 10:
			return "within10"
		default:
			return "within20"
		}
	case "sub10":
		switch {
		case a < 1 || a > 20 || b < 0 || b > a:
			return ""
		case a <= 5:
			return "within5"
		case a <= 10:
			return "within10"
		default:
			return "within20"
		}
	case "shape":
		return "basic-shapes"
	default:
		return ""
	}
}

func stageName(code string) string {
	switch code {
	case "within5":
		return "5以内"
	case "within10":
		return "10以内"
	case "within20":
		return "20以内"
	case "basic-shapes":
		return "基础图形"
	default:
		return ""
	}
}

func stageCodes(moduleCode string) []string {
	if moduleCode == "shape" {
		return []string{"basic-shapes"}
	}
	if moduleCode == "add10" || moduleCode == "sub10" {
		return []string{"within5", "within10", "within20"}
	}
	return nil
}
