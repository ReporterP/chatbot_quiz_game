package services

import (
	"sort"

	"quiz-game-backend/internal/models"
)

type ScoringService struct{}

func NewScoringService() *ScoringService {
	return &ScoringService{}
}

func (s *ScoringService) CalculateScores(answers []models.Answer) []models.Answer {
	var correct []int
	for i, a := range answers {
		if a.IsCorrect {
			correct = append(correct, i)
		}
	}

	sort.Slice(correct, func(a, b int) bool {
		return answers[correct[a]].AnsweredAt.Before(answers[correct[b]].AnsweredAt)
	})

	for rank, idx := range correct {
		bonus := 50 - rank
		if bonus < 1 {
			bonus = 1
		}
		answers[idx].Score = 100 + bonus
	}

	return answers
}
