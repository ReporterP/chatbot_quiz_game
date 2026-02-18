package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"quiz-game-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ExportOption struct {
	Text            string `json:"text"`
	IsCorrect       bool   `json:"is_correct"`
	Color           string `json:"color,omitempty"`
	CorrectPosition *int   `json:"correct_position,omitempty"`
	MatchText       string `json:"match_text,omitempty"`
}

type ExportQuestion struct {
	Text          string         `json:"text"`
	Type          string         `json:"type,omitempty"`
	CorrectNumber *float64       `json:"correct_number,omitempty"`
	Tolerance     *float64       `json:"tolerance,omitempty"`
	Options       []ExportOption `json:"options"`
}

type ExportCategory struct {
	Title     string           `json:"title"`
	Questions []ExportQuestion `json:"questions"`
}

type ExportData struct {
	Title      string           `json:"title"`
	Categories []ExportCategory `json:"categories,omitempty"`
	Questions  []ExportQuestion `json:"questions,omitempty"`
}

func (h *QuizHandler) ExportQuiz(c *gin.Context) {
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

	format := c.DefaultQuery("format", "json")

	data := ExportData{Title: quiz.Title}
	for _, cat := range quiz.Categories {
		ec := ExportCategory{Title: cat.Title}
		for _, q := range cat.Questions {
			eq := ExportQuestion{Text: q.Text, Type: q.Type, CorrectNumber: q.CorrectNumber, Tolerance: q.Tolerance}
			for _, o := range q.Options {
				eq.Options = append(eq.Options, ExportOption{
					Text: o.Text, IsCorrect: o.IsCorrect, Color: o.Color,
					CorrectPosition: o.CorrectPosition, MatchText: o.MatchText,
				})
			}
			ec.Questions = append(ec.Questions, eq)
		}
		data.Categories = append(data.Categories, ec)
	}
	for _, q := range quiz.Questions {
		eq := ExportQuestion{Text: q.Text, Type: q.Type, CorrectNumber: q.CorrectNumber, Tolerance: q.Tolerance}
		for _, o := range q.Options {
			eq.Options = append(eq.Options, ExportOption{
				Text: o.Text, IsCorrect: o.IsCorrect, Color: o.Color,
				CorrectPosition: o.CorrectPosition, MatchText: o.MatchText,
			})
		}
		data.Questions = append(data.Questions, eq)
	}

	filename := strings.ReplaceAll(quiz.Title, " ", "_")

	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", filename))

		w := csv.NewWriter(c.Writer)
		w.Write([]string{"category", "question", "type", "option1", "option2", "option3", "option4", "correct", "color1", "color2", "color3", "color4"})

		writeQuestions := func(catTitle string, questions []ExportQuestion) {
			for _, q := range questions {
				row := make([]string, 12)
				row[0] = catTitle
				row[1] = q.Text
				row[2] = q.Type
				correctIdx := ""
				for i, o := range q.Options {
					if i < 4 {
						row[3+i] = o.Text
						row[8+i] = o.Color
					}
					if o.IsCorrect {
						correctIdx = strconv.Itoa(i + 1)
					}
				}
				row[7] = correctIdx
				w.Write(row)
			}
		}

		for _, cat := range data.Categories {
			writeQuestions(cat.Title, cat.Questions)
		}
		writeQuestions("", data.Questions)
		w.Flush()
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.json\"", filename))
	c.JSON(http.StatusOK, data)
}

func (h *QuizHandler) ImportQuiz(c *gin.Context) {
	hostID := c.GetUint("host_id")
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid quiz id"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "file required"})
		return
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "cannot read file"})
		return
	}

	var importData ExportData
	fname := strings.ToLower(header.Filename)

	if strings.HasSuffix(fname, ".csv") {
		importData, err = parseCSV(body)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
	} else {
		if err := json.Unmarshal(body, &importData); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
			return
		}
	}

	si := importToServiceInput(importData)
	count, err := h.quizService.ImportQuestions(uint(quizID), hostID, si)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"imported_questions": count})
}

func parseCSV(data []byte) (ExportData, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return ExportData{}, fmt.Errorf("invalid CSV: %w", err)
	}

	if len(records) < 2 {
		return ExportData{}, fmt.Errorf("CSV must have header + at least 1 row")
	}

	catMap := make(map[string]*ExportCategory)
	var catOrder []string
	var orphans []ExportQuestion

	for _, row := range records[1:] {
		if len(row) < 8 {
			if len(row) >= 7 {
				// Legacy format without type column
				row = append([]string{row[0], row[1], ""}, row[2:]...)
			} else {
				continue
			}
		}

		catTitle := strings.TrimSpace(row[0])
		questionText := strings.TrimSpace(row[1])
		if questionText == "" {
			continue
		}
		qType := strings.TrimSpace(row[2])

		correctIdx, _ := strconv.Atoi(row[7])

		var opts []ExportOption
		for i := 0; i < 4; i++ {
			text := ""
			if i+3 < len(row) {
				text = strings.TrimSpace(row[3+i])
			}
			if text == "" {
				continue
			}
			color := ""
			if 8+i < len(row) {
				color = strings.TrimSpace(row[8+i])
			}
			opts = append(opts, ExportOption{
				Text:      text,
				IsCorrect: (i + 1) == correctIdx,
				Color:     color,
			})
		}

		eq := ExportQuestion{Text: questionText, Type: qType, Options: opts}

		if catTitle == "" {
			orphans = append(orphans, eq)
		} else {
			if _, ok := catMap[catTitle]; !ok {
				catMap[catTitle] = &ExportCategory{Title: catTitle}
				catOrder = append(catOrder, catTitle)
			}
			catMap[catTitle].Questions = append(catMap[catTitle].Questions, eq)
		}
	}

	result := ExportData{Questions: orphans}
	for _, title := range catOrder {
		result.Categories = append(result.Categories, *catMap[title])
	}
	return result, nil
}

func importToServiceInput(data ExportData) services.ImportInput {
	mapOptions := func(opts []ExportOption) []services.OptionInput {
		var result []services.OptionInput
		for _, o := range opts {
			result = append(result, services.OptionInput{
				Text: o.Text, IsCorrect: o.IsCorrect, Color: o.Color,
				CorrectPosition: o.CorrectPosition, MatchText: o.MatchText,
			})
		}
		return result
	}

	input := services.ImportInput{}
	for _, cat := range data.Categories {
		ic := services.ImportCategory{Title: cat.Title}
		for _, q := range cat.Questions {
			iq := services.ImportQuestion{
				Text: q.Text, Type: q.Type,
				CorrectNumber: q.CorrectNumber, Tolerance: q.Tolerance,
				Options: mapOptions(q.Options),
			}
			ic.Questions = append(ic.Questions, iq)
		}
		input.Categories = append(input.Categories, ic)
	}
	for _, q := range data.Questions {
		iq := services.ImportQuestion{
			Text: q.Text, Type: q.Type,
			CorrectNumber: q.CorrectNumber, Tolerance: q.Tolerance,
			Options: mapOptions(q.Options),
		}
		input.Questions = append(input.Questions, iq)
	}
	return input
}
