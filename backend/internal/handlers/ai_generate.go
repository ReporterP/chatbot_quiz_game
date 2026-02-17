package handlers

import (
	"net/http"

	"quiz-game-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AIGenerateHandler struct {
	quizService *services.QuizService
	aiService   *services.AIGenerateService
}

func NewAIGenerateHandler(quizService *services.QuizService, aiService *services.AIGenerateService) *AIGenerateHandler {
	return &AIGenerateHandler{
		quizService: quizService,
		aiService:   aiService,
	}
}

type GenerateRequest struct {
	Prompt string `json:"prompt" binding:"required,min=3"`
}

// CheckAI godoc
// @Summary      Check if AI generation is available
// @Tags         ai
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/quizzes/ai-status [get]
func (h *AIGenerateHandler) CheckAI(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"available": h.aiService.IsAvailable()})
}

// Generate godoc
// @Summary      Generate quiz with AI
// @Description  Generate a full quiz from a text prompt using Qwen LLM
// @Tags         ai
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body GenerateRequest true "Generation prompt"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} ErrorResponse
// @Failure      503 {object} ErrorResponse
// @Router       /api/v1/quizzes/generate [post]
func (h *AIGenerateHandler) Generate(c *gin.Context) {
	if !h.aiService.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "AI generation is not configured. Set QWEN_API_KEY."})
		return
	}

	hostID := c.GetUint("host_id")

	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	importInput, title, err := h.aiService.GenerateQuiz(req.Prompt)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "AI generation failed: " + err.Error()})
		return
	}

	if title == "" {
		title = "AI Quiz"
	}

	quiz, err := h.quizService.CreateQuiz(hostID, title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create quiz: " + err.Error()})
		return
	}

	count, err := h.quizService.ImportQuestions(quiz.ID, hostID, *importInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to import generated questions: " + err.Error()})
		return
	}

	fullQuiz, _ := h.quizService.GetQuizByID(quiz.ID, hostID)

	c.JSON(http.StatusOK, gin.H{
		"quiz":               fullQuiz,
		"generated_questions": count,
	})
}
