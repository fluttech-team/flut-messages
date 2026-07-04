package service

import (
	"fmt"

	"github.com/flutapp/chat-service/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Scope        string `json:"scope"`
	SessionID    string `json:"session_id"`
	TokenVersion int    `json:"token_version"`
	Role         string `json:"role"`
	Email        string `json:"email"`
	jwt.RegisteredClaims
}

type AuthService interface {
	VerifyToken(tokenStr string) (userID string, err error)
}

type authService struct {
	secret []byte
}

func NewAuthService(secret string) AuthService {
	return &authService{secret: []byte(secret)}
}

func (s *authService) VerifyToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return "", utils.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", utils.ErrInvalidToken
	}

	// claims.Subject contains userID
	return claims.Subject, nil
}
