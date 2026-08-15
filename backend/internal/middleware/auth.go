package middleware

import (
	"net/http"
	"strings"
	"time"

	"smart-live-chats/internal/config"
	"smart-live-chats/internal/database"
	"smart-live-chats/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user
func GenerateToken(user *models.User, cfg *config.Config) (string, error) {
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWTExpiryDays) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// JWTMiddleware validates the JWT token and sets the user in context
func JWTMiddleware(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			tokenString := parts[1]

			// Parse and validate token
			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			// Get user from database
			var user models.User
			if err := database.DB.First(&user, claims.UserID).Error; err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
			}

			// Block suspended accounts. Returning 401 so the frontend
			// redirects to /login just like any other invalid-token case.
			if user.Suspended {
				return echo.NewHTTPError(http.StatusUnauthorized, "account suspended")
			}

			// Set user in context
			c.Set("user", &user)
			c.Set("user_id", user.ID)

			return next(c)
		}
	}
}

// GetUserFromContext retrieves the user from the Echo context
func GetUserFromContext(c echo.Context) *models.User {
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

