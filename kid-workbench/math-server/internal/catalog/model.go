package catalog

type Module struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	OrderNo   int     `json:"orderNo"`
	ItemCount int     `json:"itemCount"`
	Stages    []Stage `json:"stages"`
}

type Stage struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ItemCount int    `json:"itemCount"`
}

type LearningItem struct {
	KpID       int64  `json:"kpId"`
	Title      string `json:"title"`
	ModuleCode string `json:"moduleCode"`
	StageCode  string `json:"stageCode"`
	Kind       string `json:"kind"`
	A          int    `json:"a,omitempty"`
	B          int    `json:"b,omitempty"`
	Shape      string `json:"shape,omitempty"`
}
