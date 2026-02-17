package services

import (
	"errors"
	"time"

	"quiz-game-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret []byte
}

func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{db: db, jwtSecret: []byte(jwtSecret)}
}

func (s *AuthService) Register(username, password string) (string, error) {
	var existing models.Host
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return "", errors.New("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	host := models.Host{
		Username:     username,
		PasswordHash: string(hash),
	}
	if err := s.db.Create(&host).Error; err != nil {
		return "", err
	}

	return s.GenerateToken(host.ID)
}

func (s *AuthService) Login(username, password string) (string, error) {
	var host models.Host
	if err := s.db.Where("username = ?", username).First(&host).Error; err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(host.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	return s.GenerateToken(host.ID)
}

func (s *AuthService) GenerateToken(hostID uint) (string, error) {
	claims := jwt.MapClaims{
		"host_id": hostID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}

	hostIDFloat, ok := claims["host_id"].(float64)
	if !ok {
		return 0, errors.New("invalid host_id in token")
	}

	return uint(hostIDFloat), nil
}
