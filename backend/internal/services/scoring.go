package services

import (
	"sort"

	"quiz-game-backend/internal/models"
)

type ScoringService struct{}

func NewScoringService() *ScoringService {
	return &ScoringService{}
}

func (s *ScoringService) CalculateScores(answers []models.Answer, totalParticipants int) []models.Answer {
	if len(answers) == 0 {
		return answers
	}

	sort.Slice(answers, func(a, b int) bool {
		return answers[a].AnsweredAt.Before(answers[b].AnsweredAt)
	})

	maxBonus := totalParticipants * 10
	if maxBonus < 10 {
		maxBonus = 10
	}
	step := 10

	for rank, i := 0, 0; i < len(answers); i++ {
		rank = i + 1
		speedBonus := maxBonus - (rank-1)*step
		if speedBonus < 10 {
			speedBonus = 10
		}

		correctBonus := 0
		if answers[i].IsCorrect {
			correctBonus = 100
		}

		answers[i].Score = correctBonus + speedBonus
	}

	return answers
}
