package auth

import (
	"database/sql"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

// Auth handles authentication token extraction from Cursor's local database
type Auth struct {
	cursorDBPath string
	cachedToken  string
	tokenMutex  sync.RWMutex
}

// New creates a new Auth instance with the default Cursor database path
func New() (*Auth, error) {
	path, err := getCursorDBPath()
	if err != nil {
		return nil, fmt.Errorf("getting cursor database path: %w", err)
	}

	return &Auth{
		cursorDBPath: path,
	}, nil
}

// NewWithPath creates a new Auth instance with a custom database path
func NewWithPath(dbPath string) *Auth {
	return &Auth{
		cursorDBPath: dbPath,
	}
}

// GetToken retrieves the session token from Cursor's database
// Returns a session token in the format: {userId}%3A%3A{jwt_token}
func (a *Auth) GetToken() (string, error) {
	// Check cache first
	a.tokenMutex.RLock()
	if a.cachedToken != "" {
		token := a.cachedToken
		a.tokenMutex.RUnlock()
		return token, nil
	}
	a.tokenMutex.RUnlock()

	// Read from database
	jwtToken, err := a.readTokenFromDB()
	if err != nil {
		return "", fmt.Errorf("reading cursor database: %w", err)
	}

	// Extract user ID from JWT
	userID, err := a.extractUserID(jwtToken)
	if err != nil {
		return "", fmt.Errorf("extracting user ID from JWT: %w", err)
	}

	// Create session token format: userId%3A%3AjwtToken
	sessionToken := fmt.Sprintf("%s%%3A%%3A%s", userID, jwtToken)

	// Cache the token
	a.tokenMutex.Lock()
	a.cachedToken = sessionToken
	a.tokenMutex.Unlock()

	return sessionToken, nil
}

// RefreshToken clears the cached token, forcing a re-read from database
func (a *Auth) RefreshToken() {
	a.tokenMutex.Lock()
	a.cachedToken = ""
	a.tokenMutex.Unlock()
}

// readTokenFromDB reads the JWT token from Cursor's SQLite database
func (a *Auth) readTokenFromDB() (string, error) {
	if _, err := os.Stat(a.cursorDBPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cursor database not found at %s: %w", a.cursorDBPath, err)
	}

	// Open SQLite database using database/sql interface
	db, err := sql.Open("sqlite", a.cursorDBPath)
	if err != nil {
		return "", fmt.Errorf("opening cursor database: %w", err)
	}
	defer db.Close()

	// Query for the access token
	var token string
	err = db.QueryRow("SELECT value FROM ItemTable WHERE key = ?", "cursorAuth/accessToken").Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("token not found in database")
		}
		return "", fmt.Errorf("querying database: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// extractUserID decodes the JWT and extracts the user ID from the 'sub' field
func (a *Auth) extractUserID(jwtToken string) (string, error) {
	// Decode JWT without verification (we don't have the signing key)
	token, _, err := jwt.NewParser().ParseUnverified(jwtToken, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("parsing JWT: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid JWT claims format")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("missing or invalid 'sub' claim in JWT")
	}

	// Extract user ID from sub field
	// Format is typically: "user|{userId}" or just "{userId}"
	parts := strings.Split(sub, "|")
	if len(parts) > 1 {
		return parts[1], nil
	}
	return sub, nil
}

// getCursorDBPath returns the path to Cursor's state.vscdb file on macOS
func getCursorDBPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}

	path := filepath.Join(
		usr.HomeDir,
		"Library",
		"Application Support",
		"Cursor",
		"User",
		"globalStorage",
		"state.vscdb",
	)

	return path, nil
}
