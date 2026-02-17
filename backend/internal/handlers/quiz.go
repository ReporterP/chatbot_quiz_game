package handlers

import (
	"net/http"
	"strconv"

	"quiz-game-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type QuizHandler struct {
	quizService *services.QuizService
}

func NewQuizHandler(quizService *services.QuizService) *QuizHandler {
	return &QuizHandler{quizService: quizService}
}

type CreateQuizRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255" example:"My Quiz"`
}

type UpdateQuizRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255" example:"Updated Quiz"`
}

// ListQuizzes godoc
// @Summary      List all quizzes
// @Description  Get all quizzes for the authenticated host
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} Quiz
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/quizzes [get]
func (h *QuizHandler) ListQuizzes(c *gin.Context) {
	hostID := c.GetUint("host_id")

	quizzes, err := h.quizService.GetQuizzesByHost(hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, quizzes)
}

// CreateQuiz godoc
// @Summary      Create a quiz
// @Description  Create a new quiz for the authenticated host
// @Tags         quizzes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateQuizRequest true "Quiz data"
// @Success      201 {object} Quiz
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/quizzes [post]
func (h *QuizHandler) CreateQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")

	var req CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	quiz, err := h.quizService.CreateQuiz(hostID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, quiz)
}

// GetQuiz godoc
// @Summary      Get a quiz
// @Description  Get quiz with all questions and options
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Success      200 {object} Quiz
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id} [get]
func (h *QuizHandler) GetQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	quiz, err := h.quizService.GetQuizByID(uint(quizID), hostID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// UpdateQuiz godoc
// @Summary      Update a quiz
// @Description  Update quiz title
// @Tags         quizzes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Param        request body UpdateQuizRequest true "Quiz data"
// @Success      200 {object} Quiz
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id} [put]
func (h *QuizHandler) UpdateQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	var req UpdateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	quiz, err := h.quizService.UpdateQuiz(uint(quizID), hostID, req.Title)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// DeleteQuiz godoc
// @Summary      Delete a quiz
// @Description  Delete a quiz and all its questions
// @Tags         quizzes
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Success      200 {object} MessageResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id} [delete]
func (h *QuizHandler) DeleteQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	if err := h.quizService.DeleteQuiz(uint(quizID), hostID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "quiz deleted"})
}
