package catalog

type Module struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	OrderNo   int    `json:"orderNo"`
	ItemCount int    `json:"itemCount"`
}

type Item struct {
	KpID          int64  `json:"kpId"`
	Letter        string `json:"letter"`
	ModuleCode    string `json:"moduleCode"`
	ModuleName    string `json:"moduleName"`
	SoloText      string `json:"soloText"`
	WordText      string `json:"wordText"`
	HasSoloSpeech bool   `json:"hasSoloSpeech"`
	HasWordSpeech bool   `json:"hasWordSpeech"`
	HasGlyph      bool   `json:"hasGlyph"`
	OrderNo       int    `json:"orderNo"`
}
