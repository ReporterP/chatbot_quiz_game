package services

import (
	"encoding/json"
	"fmt"
	"math"
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

func (s *ScoringService) CalculateScoresForType(answers []models.Answer, totalParticipants int, question *models.Question) []models.Answer {
	if len(answers) == 0 {
		return answers
	}

	qType := question.Type
	if qType == "" || qType == models.QuestionTypeSingleChoice {
		return s.CalculateScores(answers, totalParticipants)
	}

	sort.Slice(answers, func(a, b int) bool {
		return answers[a].AnsweredAt.Before(answers[b].AnsweredAt)
	})

	maxBonus := totalParticipants * 10
	if maxBonus < 10 {
		maxBonus = 10
	}
	step := 10

	for i := range answers {
		rank := i + 1
		speedBonus := maxBonus - (rank-1)*step
		if speedBonus < 10 {
			speedBonus = 10
		}

		partialScore := s.calculatePartialScore(qType, question, &answers[i])
		answers[i].Score = partialScore + speedBonus
	}

	return answers
}

func (s *ScoringService) calculatePartialScore(qType string, question *models.Question, answer *models.Answer) int {
	if answer.AnswerData == "" {
		if answer.IsCorrect {
			return 100
		}
		return 0
	}

	var data ComplexAnswerData
	if err := json.Unmarshal([]byte(answer.AnswerData), &data); err != nil {
		if answer.IsCorrect {
			return 100
		}
		return 0
	}

	switch qType {
	case models.QuestionTypeMultipleChoice:
		correctIDs := make(map[uint]bool)
		for _, o := range question.Options {
			if o.IsCorrect {
				correctIDs[o.ID] = true
			}
		}
		if len(correctIDs) == 0 {
			return 0
		}
		correctHits := 0
		wrongHits := 0
		for _, id := range data.OptionIDs {
			if correctIDs[id] {
				correctHits++
			} else {
				wrongHits++
			}
		}
		score := float64(correctHits) / float64(len(correctIDs)) * 100
		penalty := float64(wrongHits) / float64(len(question.Options)-len(correctIDs)) * 50
		result := int(math.Max(0, score-penalty))
		return result

	case models.QuestionTypeOrdering:
		posMap := make(map[uint]int)
		for _, o := range question.Options {
			if o.CorrectPosition != nil {
				posMap[o.ID] = *o.CorrectPosition
			}
		}
		total := len(question.Options)
		if total == 0 {
			return 0
		}
		correct := 0
		for i, optID := range data.Order {
			if posMap[optID] == i+1 {
				correct++
			}
		}
		return int(float64(correct) / float64(total) * 100)

	case models.QuestionTypeMatching:
		total := len(question.Options)
		if total == 0 {
			return 0
		}
		correctPairs := make(map[string]string)
		for _, o := range question.Options {
			correctPairs[formatUint(o.ID)] = o.MatchText
		}
		correct := 0
		for leftID, rightText := range data.Pairs {
			if correctPairs[leftID] == rightText {
				correct++
			}
		}
		return int(float64(correct) / float64(total) * 100)

	case models.QuestionTypeNumeric:
		if answer.IsCorrect {
			return 100
		}
		return 0

	default:
		if answer.IsCorrect {
			return 100
		}
		return 0
	}
}

func formatUint(v uint) string {
	return fmt.Sprintf("%d", v)
}
