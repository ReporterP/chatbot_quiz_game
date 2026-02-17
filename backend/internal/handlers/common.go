package handlers

import "quiz-game-backend/internal/models"

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

type MessageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

// Type aliases so swag can resolve models in annotations.
type Quiz = models.Quiz
type Question = models.Question
type Participant = models.Participant
