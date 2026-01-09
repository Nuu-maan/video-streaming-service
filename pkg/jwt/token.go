package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secretKey      []byte
	tokenDuration  time.Duration
	issuer         string
}

func NewTokenService(secretKey string, tokenDuration time.Duration, issuer string) *TokenService {
	return &TokenService{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

func (t *TokenService) GenerateToken(userID, username, role string) (string, error) {
	now := time.Now()
	expirationTime := now.Add(t.tokenDuration)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    t.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(t.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (t *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (t *TokenService) RefreshToken(oldTokenString string) (string, error) {
	claims, err := t.ValidateToken(oldTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return t.GenerateToken(claims.UserID, claims.Username, claims.Role)
}

func (t *TokenService) ExtractUserID(tokenString string) (string, error) {
	claims, err := t.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (t *TokenService) ExtractClaims(tokenString string) (*Claims, error) {
	return t.ValidateToken(tokenString)
}
