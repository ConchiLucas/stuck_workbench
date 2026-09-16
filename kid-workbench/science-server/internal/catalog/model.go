package catalog

type Module struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Order int    `json:"order" gorm:"column:order_no"`
	Total int    `json:"total" gorm:"column:item_count"`
}

type Item struct {
	KpID            int64  `json:"kpId"`
	Title           string `json:"title"`
	ModuleCode      string `json:"moduleCode"`
	ModuleName      string `json:"moduleName"`
	Difficulty      int    `json:"difficulty"`
	Summary         string `json:"summary"`
	Explanation     string `json:"explanation"`
	FunFact         string `json:"funFact"`
	ContentVersion  int    `json:"contentVersion"`
	SenseImageURL   string `json:"senseImageUrl,omitempty"`
	GlyphImageURL   string `json:"glyphImageUrl,omitempty"`
	SpeechURL       string `json:"speechUrl,omitempty"`
	HasPractice     bool   `json:"hasPractice"`
	OrderNo         int    `json:"orderNo"`
	HasSenseImage   bool   `json:"-"`
	HasGlyphImage   bool   `json:"-"`
	HasSpeechAudio  bool   `json:"-"`
	GlyphObjectKey  string `json:"-"`
	SenseObjectKey  string `json:"-"`
	SpeechObjectKey string `json:"-"`
}
