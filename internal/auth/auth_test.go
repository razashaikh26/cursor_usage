package auth

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name    string
		jwtSub  string
		want    string
		wantErr bool
	}{
		{
			name:    "user|userId format",
			jwtSub:  "user|user_12345",
			want:    "user_12345",
			wantErr: false,
		},
		{
			name:    "direct userId",
			jwtSub:  "user_12345",
			want:    "user_12345",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal JWT token for testing
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": tt.jwtSub,
			})
			
			// Get the token string (unsigned, for testing)
			tokenString, err := token.SigningString()
			if err != nil {
				t.Fatalf("Failed to create test token: %v", err)
			}

			// Parse it back to test extraction
			parsedToken, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
			if err != nil {
				// For this test, we'll directly test the extraction logic
				// by creating a token with the sub claim
				claims := jwt.MapClaims{"sub": tt.jwtSub}
				got, err := extractUserIDFromClaims(claims)
				if (err != nil) != tt.wantErr {
					t.Errorf("extractUserID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("extractUserID() = %v, want %v", got, tt.want)
				}
				return
			}

			claims, ok := parsedToken.Claims.(jwt.MapClaims)
			if !ok {
				t.Fatal("Failed to get claims")
			}
			claims["sub"] = tt.jwtSub

			got, err := extractUserIDFromClaims(claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to extract user ID from claims (extracted from auth.go logic)
func extractUserIDFromClaims(claims jwt.MapClaims) (string, error) {
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing or invalid 'sub' claim in JWT")
	}

	// Extract user ID from sub field
	parts := strings.Split(sub, "|")
	if len(parts) > 1 {
		return parts[1], nil
	}
	return sub, nil
}
