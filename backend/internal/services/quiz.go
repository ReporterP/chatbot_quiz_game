package services

import (
	"errors"

	"quiz-game-backend/internal/models"

	"gorm.io/gorm"
)

type QuizService struct {
	db *gorm.DB
}

func NewQuizService(db *gorm.DB) *QuizService {
	return &QuizService{db: db}
}

func (s *QuizService) GetQuizzesByHost(hostID uint) ([]models.Quiz, error) {
	var quizzes []models.Quiz
	err := s.db.Where("host_id = ?", hostID).
		Preload("Categories", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Preload("Categories.Questions", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Preload("Categories.Questions.Options").
		Preload("Categories.Questions.Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Order("created_at DESC").
		Find(&quizzes).Error
	if err != nil {
		return nil, err
	}

	for i := range quizzes {
		var orphans []models.Question
		s.db.Where("quiz_id = ? AND category_id IS NULL", quizzes[i].ID).
			Order("order_num ASC").
			Preload("Options").
			Preload("Images", func(db *gorm.DB) *gorm.DB {
				return db.Order("order_num ASC")
			}).
			Find(&orphans)
		quizzes[i].Questions = orphans
	}

	return quizzes, nil
}

func (s *QuizService) CreateQuiz(hostID uint, title string) (*models.Quiz, error) {
	quiz := models.Quiz{
		HostID: hostID,
		Title:  title,
	}
	if err := s.db.Create(&quiz).Error; err != nil {
		return nil, err
	}
	return &quiz, nil
}

func (s *QuizService) GetQuizByID(quizID, hostID uint) (*models.Quiz, error) {
	var quiz models.Quiz
	err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).
		Preload("Categories", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Preload("Categories.Questions", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Preload("Categories.Questions.Options").
		Preload("Categories.Questions.Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		First(&quiz).Error
	if err != nil {
		return nil, errors.New("quiz not found")
	}

	var orphans []models.Question
	s.db.Where("quiz_id = ? AND category_id IS NULL", quizID).
		Order("order_num ASC").
		Preload("Options").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_num ASC")
		}).
		Find(&orphans)
	quiz.Questions = orphans

	return &quiz, nil
}

func (s *QuizService) UpdateQuiz(quizID, hostID uint, title, mode string) (*models.Quiz, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	quiz.Title = title
	if mode == "web" || mode == "bot" {
		quiz.Mode = mode
	}
	if err := s.db.Save(&quiz).Error; err != nil {
		return nil, err
	}
	return &quiz, nil
}

func (s *QuizService) DeleteQuiz(quizID, hostID uint) error {
	result := s.db.Where("id = ? AND host_id = ?", quizID, hostID).Delete(&models.Quiz{})
	if result.RowsAffected == 0 {
		return errors.New("quiz not found")
	}
	return result.Error
}

func (s *QuizService) CreateCategory(quizID, hostID uint, title string) (*models.Category, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	var maxOrder int
	s.db.Model(&models.Category{}).Where("quiz_id = ?", quizID).Select("COALESCE(MAX(order_num), -1)").Scan(&maxOrder)

	cat := models.Category{
		QuizID:   quizID,
		Title:    title,
		OrderNum: maxOrder + 1,
	}
	if err := s.db.Create(&cat).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

func (s *QuizService) UpdateCategory(categoryID, hostID uint, title string) (*models.Category, error) {
	var cat models.Category
	if err := s.db.First(&cat, categoryID).Error; err != nil {
		return nil, errors.New("category not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", cat.QuizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("access denied")
	}

	cat.Title = title
	s.db.Save(&cat)
	return &cat, nil
}

func (s *QuizService) DeleteCategory(categoryID, hostID uint) error {
	var cat models.Category
	if err := s.db.First(&cat, categoryID).Error; err != nil {
		return errors.New("category not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", cat.QuizID, hostID).First(&quiz).Error; err != nil {
		return errors.New("access denied")
	}

	s.db.Where("question_id IN (SELECT id FROM questions WHERE category_id = ?)", categoryID).Delete(&models.Option{})
	s.db.Where("question_id IN (SELECT id FROM questions WHERE category_id = ?)", categoryID).Delete(&models.QuestionImage{})
	s.db.Where("category_id = ?", categoryID).Delete(&models.Question{})
	return s.db.Delete(&cat).Error
}

func (s *QuizService) ReorderQuiz(quizID, hostID uint, order ReorderInput) error {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return errors.New("quiz not found")
	}

	tx := s.db.Begin()
	for _, c := range order.Categories {
		tx.Model(&models.Category{}).Where("id = ? AND quiz_id = ?", c.ID, quizID).Update("order_num", c.OrderNum)
		for _, q := range c.Questions {
			catID := c.ID
			tx.Model(&models.Question{}).Where("id = ? AND quiz_id = ?", q.ID, quizID).Updates(map[string]interface{}{
				"order_num":   q.OrderNum,
				"category_id": catID,
			})
		}
	}
	for _, q := range order.OrphanQuestions {
		tx.Model(&models.Question{}).Where("id = ? AND quiz_id = ?", q.ID, quizID).Select("order_num", "category_id").Updates(map[string]interface{}{
			"order_num":   q.OrderNum,
			"category_id": nil,
		})
	}
	tx.Commit()
	return nil
}

func (s *QuizService) CreateQuestion(quizID, hostID uint, text string, orderNum int, categoryID *uint, options []OptionInput) (*models.Question, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	if err := validateOptions(options); err != nil {
		return nil, err
	}

	question := models.Question{
		QuizID:     quizID,
		CategoryID: categoryID,
		Text:       text,
		OrderNum:   orderNum,
	}

	tx := s.db.Begin()
	if err := tx.Create(&question).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, o := range options {
		opt := models.Option{
			QuestionID: question.ID,
			Text:       o.Text,
			IsCorrect:  o.IsCorrect,
			Color:      o.Color,
		}
		if err := tx.Create(&opt).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()

	s.db.Preload("Options").Preload("Images").First(&question, question.ID)
	return &question, nil
}

func (s *QuizService) UpdateQuestion(questionID, hostID uint, text string, orderNum int, categoryID *uint, options []OptionInput) (*models.Question, error) {
	var question models.Question
	if err := s.db.Preload("Options").First(&question, questionID).Error; err != nil {
		return nil, errors.New("question not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", question.QuizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("quiz not found or access denied")
	}

	if err := validateOptions(options); err != nil {
		return nil, err
	}

	tx := s.db.Begin()

	question.Text = text
	question.OrderNum = orderNum
	question.CategoryID = categoryID
	if err := tx.Save(&question).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Where("question_id = ?", questionID).Delete(&models.Option{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, o := range options {
		opt := models.Option{
			QuestionID: questionID,
			Text:       o.Text,
			IsCorrect:  o.IsCorrect,
			Color:      o.Color,
		}
		if err := tx.Create(&opt).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()

	s.db.Preload("Options").Preload("Images").First(&question, questionID)
	return &question, nil
}

func (s *QuizService) DeleteQuestion(questionID, hostID uint) error {
	var question models.Question
	if err := s.db.First(&question, questionID).Error; err != nil {
		return errors.New("question not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", question.QuizID, hostID).First(&quiz).Error; err != nil {
		return errors.New("quiz not found or access denied")
	}

	s.db.Where("question_id = ?", questionID).Delete(&models.QuestionImage{})
	return s.db.Select("Options").Delete(&question).Error
}

func (s *QuizService) AddQuestionImage(questionID, hostID uint, url string) (*models.QuestionImage, error) {
	var question models.Question
	if err := s.db.First(&question, questionID).Error; err != nil {
		return nil, errors.New("question not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", question.QuizID, hostID).First(&quiz).Error; err != nil {
		return nil, errors.New("access denied")
	}

	var maxOrder int
	s.db.Model(&models.QuestionImage{}).Where("question_id = ?", questionID).Select("COALESCE(MAX(order_num), -1)").Scan(&maxOrder)

	img := models.QuestionImage{
		QuestionID: questionID,
		URL:        url,
		OrderNum:   maxOrder + 1,
	}
	if err := s.db.Create(&img).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

func (s *QuizService) DeleteQuestionImage(imageID, hostID uint) error {
	var img models.QuestionImage
	if err := s.db.First(&img, imageID).Error; err != nil {
		return errors.New("image not found")
	}

	var question models.Question
	if err := s.db.First(&question, img.QuestionID).Error; err != nil {
		return errors.New("question not found")
	}

	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", question.QuizID, hostID).First(&quiz).Error; err != nil {
		return errors.New("access denied")
	}

	return s.db.Delete(&img).Error
}

type ImportInput struct {
	Categories []ImportCategory
	Questions  []ImportQuestion
}

type ImportCategory struct {
	Title     string
	Questions []ImportQuestion
}

type ImportQuestion struct {
	Text    string
	Options []OptionInput
}

func (s *QuizService) ImportQuestions(quizID, hostID uint, input ImportInput) (int, error) {
	var quiz models.Quiz
	if err := s.db.Where("id = ? AND host_id = ?", quizID, hostID).First(&quiz).Error; err != nil {
		return 0, errors.New("quiz not found")
	}

	var maxCatOrder int
	s.db.Model(&models.Category{}).Where("quiz_id = ?", quizID).Select("COALESCE(MAX(order_num), -1)").Scan(&maxCatOrder)

	var maxQOrder int
	s.db.Model(&models.Question{}).Where("quiz_id = ? AND category_id IS NULL", quizID).Select("COALESCE(MAX(order_num), -1)").Scan(&maxQOrder)

	tx := s.db.Begin()
	count := 0

	for _, cat := range input.Categories {
		maxCatOrder++
		dbCat := models.Category{QuizID: quizID, Title: cat.Title, OrderNum: maxCatOrder}
		if err := tx.Create(&dbCat).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

		for qIdx, q := range cat.Questions {
			if len(q.Options) < 2 {
				continue
			}
			dbQ := models.Question{QuizID: quizID, CategoryID: &dbCat.ID, Text: q.Text, OrderNum: qIdx}
			if err := tx.Create(&dbQ).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
			for _, o := range q.Options {
				opt := models.Option{QuestionID: dbQ.ID, Text: o.Text, IsCorrect: o.IsCorrect, Color: o.Color}
				if err := tx.Create(&opt).Error; err != nil {
					tx.Rollback()
					return 0, err
				}
			}
			count++
		}
	}

	for _, q := range input.Questions {
		if len(q.Options) < 2 {
			continue
		}
		maxQOrder++
		dbQ := models.Question{QuizID: quizID, Text: q.Text, OrderNum: maxQOrder}
		if err := tx.Create(&dbQ).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		for _, o := range q.Options {
			opt := models.Option{QuestionID: dbQ.ID, Text: o.Text, IsCorrect: o.IsCorrect, Color: o.Color}
			if err := tx.Create(&opt).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
		}
		count++
	}

	tx.Commit()
	return count, nil
}

type OptionInput struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
	Color     string `json:"color"`
}

type ReorderInput struct {
	Categories      []CategoryOrder `json:"categories"`
	OrphanQuestions []QuestionOrder `json:"orphan_questions"`
}

type CategoryOrder struct {
	ID        uint            `json:"id"`
	OrderNum  int             `json:"order_num"`
	Questions []QuestionOrder `json:"questions"`
}

type QuestionOrder struct {
	ID       uint `json:"id"`
	OrderNum int  `json:"order_num"`
}

func validateOptions(options []OptionInput) error {
	if len(options) < 2 || len(options) > 4 {
		return errors.New("question must have 2 to 4 options")
	}

	correctCount := 0
	for _, o := range options {
		if o.IsCorrect {
			correctCount++
		}
	}
	if correctCount != 1 {
		return errors.New("exactly one option must be marked as correct")
	}

	return nil
}
