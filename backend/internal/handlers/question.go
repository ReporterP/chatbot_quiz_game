package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"quiz-game-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	quizService *services.QuizService
}

func NewQuestionHandler(quizService *services.QuizService) *QuestionHandler {
	return &QuestionHandler{quizService: quizService}
}

type CreateQuestionRequest struct {
	Text          string                 `json:"text" binding:"required"`
	OrderNum      int                    `json:"order_num"`
	CategoryID    *uint                  `json:"category_id"`
	Type          string                 `json:"type"`
	CorrectNumber *float64               `json:"correct_number"`
	Tolerance     *float64               `json:"tolerance"`
	Options       []services.OptionInput `json:"options"`
}

// CreateQuestion godoc
// @Summary      Add a question to a quiz
// @Tags         questions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Param        request body CreateQuestionRequest true "Question data"
// @Success      201 {object} Question
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id}/questions [post]
func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	var req CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := services.QuestionInput{
		Text:          req.Text,
		OrderNum:      req.OrderNum,
		CategoryID:    req.CategoryID,
		Type:          req.Type,
		CorrectNumber: req.CorrectNumber,
		Tolerance:     req.Tolerance,
		Options:       req.Options,
	}

	question, err := h.quizService.CreateQuestion(uint(quizID), hostID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, question)
}

// UpdateQuestion godoc
// @Summary      Update a question
// @Tags         questions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Question ID"
// @Param        request body CreateQuestionRequest true "Question data"
// @Success      200 {object} Question
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/questions/{id} [put]
func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
	hostID := c.GetUint("host_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid question id"})
		return
	}

	var req CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := services.QuestionInput{
		Text:          req.Text,
		OrderNum:      req.OrderNum,
		CategoryID:    req.CategoryID,
		Type:          req.Type,
		CorrectNumber: req.CorrectNumber,
		Tolerance:     req.Tolerance,
		Options:       req.Options,
	}

	question, err := h.quizService.UpdateQuestion(uint(questionID), hostID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, question)
}

// DeleteQuestion godoc
// @Summary      Delete a question
// @Tags         questions
// @Security     BearerAuth
// @Param        id path int true "Question ID"
// @Success      200 {object} MessageResponse
// @Failure      404 {object} ErrorResponse
// @Router       /api/v1/questions/{id} [delete]
func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
	hostID := c.GetUint("host_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid question id"})
		return
	}

	if err := h.quizService.DeleteQuestion(uint(questionID), hostID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "question deleted"})
}

// CreateCategory godoc
// @Summary      Create a category in a quiz
// @Tags         questions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id}/categories [post]
func (h *QuestionHandler) CreateCategory(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cat, err := h.quizService.CreateCategory(uint(quizID), hostID, req.Title)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cat)
}

// UpdateCategory godoc
// @Summary      Update a category
// @Tags         questions
// @Security     BearerAuth
// @Param        id path int true "Category ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/categories/{id} [put]
func (h *QuestionHandler) UpdateCategory(c *gin.Context) {
	hostID := c.GetUint("host_id")
	catID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category id"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	cat, err := h.quizService.UpdateCategory(uint(catID), hostID, req.Title)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, cat)
}

// DeleteCategory godoc
// @Summary      Delete a category and its questions
// @Tags         questions
// @Security     BearerAuth
// @Param        id path int true "Category ID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/categories/{id} [delete]
func (h *QuestionHandler) DeleteCategory(c *gin.Context) {
	hostID := c.GetUint("host_id")
	catID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid category id"})
		return
	}

	if err := h.quizService.DeleteCategory(uint(catID), hostID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "category deleted"})
}

// ReorderQuiz godoc
// @Summary      Reorder categories and questions
// @Tags         questions
// @Accept       json
// @Security     BearerAuth
// @Param        id path int true "Quiz ID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/quizzes/{id}/reorder [put]
func (h *QuestionHandler) ReorderQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	var req services.ReorderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.quizService.ReorderQuiz(uint(quizID), hostID, req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "reordered"})
}

// UploadImage godoc
// @Summary      Upload an image
// @Tags         questions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formance file true "Image file"
// @Success      200 {object} map[string]string
// @Router       /api/v1/upload [post]
func (h *QuestionHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no file provided"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	audioExts := map[string]bool{".mp3": true, ".ogg": true, ".wav": true, ".m4a": true, ".aac": true, ".flac": true}
	videoExts := map[string]bool{".mp4": true, ".webm": true, ".mov": true, ".avi": true, ".mkv": true}

	if file.Size > 100<<20 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "file too large (max 100MB)"})
		return
	}

	mediaType := ""
	if imageExts[ext] {
		mediaType = "image"
	} else if audioExts[ext] {
		mediaType = "audio"
	} else if videoExts[ext] {
		mediaType = "video"
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "unsupported file format"})
		return
	}

	filename := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), rand.Intn(100000), ext)
	dst := filepath.Join("/uploads", filename)

	os.MkdirAll("/uploads", 0755)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename, "type": mediaType})
}

// AddQuestionImage godoc
// @Summary      Add image to a question
// @Tags         questions
// @Security     BearerAuth
// @Param        id path int true "Question ID"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/questions/{id}/images [post]
func (h *QuestionHandler) AddQuestionImage(c *gin.Context) {
	hostID := c.GetUint("host_id")
	questionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid question id"})
		return
	}

	var req struct {
		URL  string `json:"url" binding:"required"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	img, err := h.quizService.AddQuestionImage(uint(questionID), hostID, req.URL, req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, img)
}

// DeleteQuestionImage godoc
// @Summary      Delete a question image
// @Tags         questions
// @Security     BearerAuth
// @Param        id path int true "Image ID"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/images/{id} [delete]
func (h *QuestionHandler) DeleteQuestionImage(c *gin.Context) {
	hostID := c.GetUint("host_id")
	imageID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid image id"})
		return
	}

	if err := h.quizService.DeleteQuestionImage(uint(imageID), hostID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "image deleted"})
}
