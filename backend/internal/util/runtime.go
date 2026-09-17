package util

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string { return fmt.Sprintf("code=%d message=%s", e.Code, e.Message) }

func NewAppError(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(err error, format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

type Claims struct {
	UserID uint64 `json:"user_id"`
	Phone  string `json:"phone"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(secret string, expire time.Duration, userID uint64, phone, role string) (string, error) {
	claims := Claims{UserID: userID, Phone: phone, Role: role, RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)), IssuedAt: jwt.NewNumericDate(time.Now()),
		Issuer: "airport-ground-equipment-clearance",
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithIssuer("airport-ground-equipment-clearance"), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func DurationHours(hours int) time.Duration { return time.Duration(hours) * time.Hour }
