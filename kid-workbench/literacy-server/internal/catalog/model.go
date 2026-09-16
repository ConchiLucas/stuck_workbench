package catalog

type Module struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	OrderNo   int    `json:"orderNo"`
	ItemCount int    `json:"itemCount"`
}

type Item struct {
	KpID       int64  `json:"kpId"`
	Character  string `json:"character"`
	ModuleCode string `json:"moduleCode"`
	ModuleName string `json:"moduleName"`
	HasGlyph   bool   `json:"hasGlyph"`
	HasSense   bool   `json:"hasSense"`
	HasSpeech  bool   `json:"hasSpeech"`
	OrderNo    int    `json:"orderNo"`
}
