package util

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"groundclearance/internal/constants"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken("test-secret", time.Hour, 7, "13800000001", constants.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken("test-secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 7 || claims.Role != constants.RoleAdmin || claims.Phone != "13800000001" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestParseTokenRejectsNonHS256Algorithm(t *testing.T) {
	claims := Claims{UserID: 7, Phone: "13800000001", Role: constants.RoleAdmin, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "airport-ground-equipment-clearance", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken("test-secret", token); err == nil {
		t.Fatal("HS384 token must be rejected")
	}
}

func TestAppError(t *testing.T) {
	err := NewAppError(constants.CodeStateConflict, "transition blocked")
	if err.Code != constants.CodeStateConflict || err.Error() == "" {
		t.Fatal("app error mismatch")
	}
}
