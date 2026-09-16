package plan

import (
	"errors"
	"sync"

	"github.com/conchi/poem-server/internal/catalog"
)

var (
	ErrPavilionNotFound = errors.New("pavilion not found")
	ErrPlanNotFound     = errors.New("plan not found")
	ErrItemNotFound     = errors.New("item not found")
)

type Item struct {
	ID     int64  `json:"id"`
	ClueID string `json:"clueId"`
	Order  int    `json:"order"`
}

type Detail struct {
	ID           int64  `json:"id"`
	ChildID      int64  `json:"childId"`
	PavilionCode string `json:"pavilionCode"`
	Status       string `json:"status"`
	Items        []Item `json:"items"`
}

type AnswerResult struct {
	Correct bool `json:"correct"`
}

type Store struct {
	mu     sync.Mutex
	nextID int64
	plans  map[int64]Detail
}

func NewStore() *Store {
	return &Store{nextID: 1_000_000_000, plans: map[int64]Detail{}}
}

func (s *Store) Create(childID int64, pavilionCode string) (Detail, error) {
	clues := catalog.Clues(pavilionCode)
	if len(clues) == 0 {
		return Detail{}, ErrPavilionNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	items := make([]Item, len(clues))
	for i, clue := range clues {
		items[i] = Item{ID: int64(i + 1), ClueID: clue.ID, Order: i + 1}
	}
	detail := Detail{ID: id, ChildID: childID, PavilionCode: pavilionCode, Status: "ready", Items: items}
	s.plans[id] = detail
	return detail, nil
}

func (s *Store) Get(childID, planID int64) (Detail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detail, ok := s.plans[planID]
	if !ok || detail.ChildID != childID {
		return Detail{}, ErrPlanNotFound
	}
	return detail, nil
}

func (s *Store) Start(childID, planID int64) (Detail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detail, ok := s.plans[planID]
	if !ok || detail.ChildID != childID {
		return Detail{}, ErrPlanNotFound
	}
	detail.Status = "ongoing"
	s.plans[planID] = detail
	return detail, nil
}

func (s *Store) Answer(childID, planID, itemID int64, answerID string) (AnswerResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detail, ok := s.plans[planID]
	if !ok || detail.ChildID != childID {
		return AnswerResult{}, ErrPlanNotFound
	}
	var clueID string
	for _, item := range detail.Items {
		if item.ID == itemID {
			clueID = item.ClueID
			break
		}
	}
	if clueID == "" {
		return AnswerResult{}, ErrItemNotFound
	}
	for _, clue := range catalog.Clues(detail.PavilionCode) {
		if clue.ID == clueID {
			return AnswerResult{Correct: clue.AnswerID == answerID}, nil
		}
	}
	return AnswerResult{}, ErrItemNotFound
}

func (s *Store) Finish(childID, planID int64) (Detail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detail, ok := s.plans[planID]
	if !ok || detail.ChildID != childID {
		return Detail{}, ErrPlanNotFound
	}
	detail.Status = "done"
	s.plans[planID] = detail
	return detail, nil
}
