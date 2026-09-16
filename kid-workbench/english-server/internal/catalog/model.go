package catalog

type Module struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	OrderNo   int    `json:"orderNo"`
	WordCount int    `json:"wordCount"`
}

type Word struct {
	KpID             int64  `json:"kpId"`
	Word             string `json:"word"`
	MeaningZh        string `json:"meaningZh"`
	Phonetic         string `json:"phonetic,omitempty"`
	PartOfSpeech     string `json:"partOfSpeech,omitempty"`
	Example          string `json:"example,omitempty"`
	ExampleMeaningZh string `json:"exampleMeaningZh,omitempty"`
	ModuleCode       string `json:"moduleCode"`
	ModuleName       string `json:"moduleName"`
	HasGlyph         bool   `json:"hasGlyph"`
	HasSense         bool   `json:"hasSense"`
	HasSpeech        bool   `json:"hasSpeech"`
	OrderNo          int    `json:"orderNo"`
}
