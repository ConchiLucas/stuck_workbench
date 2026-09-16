package knowledge

import (
	"encoding/json"
	"time"

	"github.com/conchi/study-learning/chengyucontent"
	"github.com/conchi/study-learning/englishcontent"
	"github.com/conchi/study-learning/logiccontent"
	"github.com/conchi/study-learning/mathcontent"
	"github.com/conchi/study-learning/phrasecontent"
	"github.com/conchi/study-learning/poemcontent"
	"github.com/conchi/study-learning/sciencecontent"
)

type Filter struct {
	From, To, MasteredOn                                          string
	Subject, Module, Q, State, Skill, Cursor, View, FollowUpState string
	KpID                                                          int64
	Limit                                                         int
	WrongOnly                                                     bool
}
type Fault struct {
	Status        int
	Code, Message string
}

func (e *Fault) Error() string   { return e.Message }
func bad(code, msg string) error { return &Fault{400, code, msg} }

type Coverage struct {
	Level       string   `json:"level"`
	ReasonCodes []string `json:"reasonCodes"`
}
type Page[T any] struct {
	Items        []T       `json:"items"`
	NextCursor   string    `json:"nextCursor,omitempty"`
	HasMore      bool      `json:"hasMore"`
	EvidenceAsOf time.Time `json:"evidenceAsOf"`
	StateReadAt  time.Time `json:"stateReadAt"`
	Coverage     Coverage  `json:"coverage"`
}
type Stats struct {
	ObservedAttempts          int      `json:"observedAttempts"`
	ObservedCorrect           int      `json:"observedCorrect"`
	IndependentAttempts       int      `json:"independentAttempts"`
	IndependentCorrect        int      `json:"independentCorrect"`
	AssistedAttempts          int      `json:"assistedAttempts"`
	UnknownAssistanceAttempts int      `json:"unknownAssistanceAttempts"`
	Accuracy                  *float64 `json:"accuracy"`
	IndependentAccuracy       *float64 `json:"independentAccuracy"`
}

func (s *Stats) finish() {
	if s.ObservedAttempts > 0 {
		v := float64(s.ObservedCorrect) / float64(s.ObservedAttempts)
		s.Accuracy = &v
	}
	if s.IndependentAttempts > 0 {
		v := float64(s.IndependentCorrect) / float64(s.IndependentAttempts)
		s.IndependentAccuracy = &v
	}
}

type Skill struct {
	MasteredAt       *time.Time `json:"-"`
	StateRecorded    bool       `json:"-"`
	SkillCode        string     `json:"skillCode"`
	Label            string     `json:"label"`
	MasteryStatus    string     `json:"masteryStatus"`
	Practiced        bool       `json:"practiced"`
	DueAt            *time.Time `json:"dueAt"`
	Stats            Stats      `json:"stats"`
	EvidenceCoverage string     `json:"evidenceCoverage"`
}
type Point struct {
	MasteryCategory    string     `json:"masteryCategory"`
	ReviewDue          bool       `json:"reviewDue"`
	FirstMasteredAt    *string    `json:"firstMasteredAt"`
	KpID               int64      `json:"kpId"`
	Title              string     `json:"title"`
	SubjectCode        string     `json:"subjectCode"`
	SubjectName        string     `json:"subjectName"`
	ModuleCode         string     `json:"moduleCode"`
	ModuleName         string     `json:"moduleName"`
	OrderNo            int        `json:"-"`
	MasteryStatus      string     `json:"masteryStatus"`
	Stats              Stats      `json:"stats"`
	Skills             []Skill    `json:"skills" gorm:"-"`
	WrongCount         int        `json:"wrongCount"`
	LastPracticedAt    *time.Time `json:"lastPracticedAt"`
	HasMasteredAbility bool       `json:"hasMasteredAbility"`
	DueAt              *time.Time `json:"dueAt"`
	Coverage           Coverage   `json:"coverage" gorm:"-"`
}
type AbilityCounts struct {
	Mastered    int `json:"mastered"`
	Learning    int `json:"learning"`
	Shaky       int `json:"shaky"`
	Unpracticed int `json:"unpracticed"`
	Unknown     int `json:"unknown"`
}
type Subject struct {
	PointCounts          PointCounts   `json:"pointCounts"`
	AttemptsCount        int           `json:"attemptsCount"`
	WrongCount           int           `json:"wrongCount"`
	Abilities            AbilityCounts `json:"abilities"`
	UnmappedPointCount   int           `json:"unmappedPointCount"`
	Code                 string        `json:"code"`
	Name                 string        `json:"name"`
	Total                int           `json:"total"`
	PracticedCount       int           `json:"practicedCount"`
	MasteredAbilityCount int           `json:"masteredAbilityCount"`
	WrongPointCount      int           `json:"wrongPointCount"`
	Coverage             Coverage      `json:"coverage"`
}
type Summary struct {
	TotalCount  int         `json:"totalCount"`
	PointCounts PointCounts `json:"pointCounts"`
	Child       struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Grade string `json:"grade"`
	} `json:"child"`
	Subjects             []Subject `json:"subjects"`
	Stats                Stats     `json:"stats"`
	PracticedCount       int       `json:"practicedCount"`
	MasteredAbilityCount int       `json:"masteredAbilityCount"`
	WrongPointCount      int       `json:"wrongPointCount"`
	WrongCount           int       `json:"wrongCount"`
	Coverage             Coverage  `json:"coverage"`
}
type SourceRef struct {
	Kind              string `json:"kind"`
	ReceiptID         int64  `json:"receiptId,omitempty"`
	QuestionVersionID int64  `json:"questionVersionId,omitempty"`
	InstanceID        string `json:"instanceId,omitempty"`
	ClientID          string `json:"clientId,omitempty"`
	PlanID            int64  `json:"planId,omitempty"`
	ItemID            int64  `json:"itemId,omitempty"`
}
type Option struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	SemanticID   string `json:"semanticId,omitempty"`
	ImageMediaID string `json:"imageMediaId,omitempty"`
	AudioMediaID string `json:"audioMediaId,omitempty"`
}
type Stem struct {
	Text         string `json:"text"`
	ImageMediaID string `json:"imageMediaId,omitempty"`
	AudioMediaID string `json:"audioMediaId,omitempty"`
}
type QuestionView struct {
	Interaction    string          `json:"interaction"`
	Stem           Stem            `json:"stem"`
	Options        []Option        `json:"options"`
	AnswerOptionID string          `json:"answerOptionId,omitempty"`
	TargetText     string          `json:"targetText"`
	Visual         json.RawMessage `json:"visual,omitempty"`
}
type ResponseView struct {
	Kind             string          `json:"kind"`
	SelectedOptionID string          `json:"selectedOptionId,omitempty"`
	Value            string          `json:"value,omitempty"`
	Strokes          json.RawMessage `json:"strokes,omitempty"`
	HintsUsed        *int            `json:"hintsUsed,omitempty"`
	Evaluation       json.RawMessage `json:"evaluation,omitempty"`
	EvaluatorVersion string          `json:"evaluatorVersion,omitempty"`
}
type Evidence struct {
	LaterIndependentCorrect bool                           `json:"-"`
	AttemptID               int64                          `json:"attemptId"`
	ChildID                 int64                          `json:"childId"`
	KpID                    int64                          `json:"kpId"`
	Title                   string                         `json:"title"`
	SubjectCode             string                         `json:"subjectCode"`
	SubjectName             string                         `json:"subjectName"`
	ModuleCode              string                         `json:"moduleCode"`
	ModuleName              string                         `json:"moduleName"`
	SkillCode               string                         `json:"skillCode"`
	SkillLabel              string                         `json:"skillLabel"`
	QuestionType            string                         `json:"questionType"`
	OccurredAt              time.Time                      `json:"occurredAt"`
	IsCorrect               bool                           `json:"isCorrect"`
	CostMs                  int                            `json:"costMs"`
	Assistance              string                         `json:"assistance"`
	Source                  SourceRef                      `json:"source" gorm:"-"`
	PlanID                  int64                          `json:"planId,omitempty"`
	PlanItemID              int64                          `json:"planItemId,omitempty"`
	QuestionFidelity        string                         `json:"questionFidelity"`
	SelectionFidelity       string                         `json:"selectionFidelity"`
	MediaFidelity           string                         `json:"mediaFidelity"`
	ReviewEligible          bool                           `json:"reviewEligible"`
	ReviewBlockReasons      []string                       `json:"reviewBlockReasons" gorm:"-"`
	EvidenceReasonCodes     []string                       `json:"evidenceReasonCodes" gorm:"-"`
	Question                *QuestionView                  `json:"question" gorm:"-"`
	MathExample             *mathcontent.MathExample       `json:"mathExample,omitempty" gorm:"-"`
	EnglishExample          *englishcontent.EnglishExample `json:"englishExample,omitempty" gorm:"-"`
	PhraseExample           *phrasecontent.PhraseExample   `json:"phraseExample,omitempty" gorm:"-"`
	ChengyuExample          *chengyucontent.ChengyuExample `json:"chengyuExample,omitempty" gorm:"-"`
	ScienceExample          *sciencecontent.ScienceExample `json:"scienceExample,omitempty" gorm:"-"`
	PoemExample             *poemcontent.PoemExample      `json:"poemExample,omitempty" gorm:"-"`
	LogicExample            *logiccontent.LogicExample    `json:"logicExample,omitempty" gorm:"-"`
	AudioMutable            bool                           `json:"audioMutable,omitempty" gorm:"-"`
	Response                ResponseView                   `json:"response" gorm:"-"`
	FollowUpState           string                         `json:"followUpState"`
	LaterAttempts           int                            `json:"laterAttempts"`
	InstanceKey             string                         `json:"-"`
	SelectedSemanticID      string                         `json:"-"`
	Media                   map[string]string              `json:"-" gorm:"-"`
	ClientID                string                         `json:"-"`
	QuestionID              *int64                         `json:"-"`
}
type Candidate struct {
	EvidenceHasMore          bool                `json:"evidenceHasMore"`
	WrongInstanceCount       *int                `json:"wrongInstanceCount"`
	Key                      string              `json:"key"`
	KpID                     int64               `json:"kpId"`
	Title                    string              `json:"title"`
	SubjectCode              string              `json:"subjectCode"`
	ModuleCode               string              `json:"moduleCode"`
	SkillCode                string              `json:"skillCode"`
	QuestionType             string              `json:"questionType"`
	ReasonCode               string              `json:"reasonCode"`
	ReasonText               string              `json:"reasonText"`
	IndependentInstanceCount int                 `json:"independentInstanceCount"`
	WrongInitialCount        int                 `json:"wrongInitialCount"`
	WrongCount               int                 `json:"wrongCount"`
	LastWrongAt              time.Time           `json:"lastWrongAt"`
	Evidence                 []CandidateEvidence `json:"evidence"`
	Mode                     string              `json:"mode"`
	RequestedCount           int                 `json:"requestedCount"`
	PreferredDistractorKpIDs []int64             `json:"preferredDistractorKpIds"`
	ReviewEligible           bool                `json:"reviewEligible"`
	ReviewBlockReasons       []string            `json:"reviewBlockReasons" gorm:"-"`
}
type CandidateEvidence struct {
	AttemptID int64     `json:"attemptId"`
	Role      string    `json:"role"`
	Source    SourceRef `json:"source" gorm:"-"`
}
