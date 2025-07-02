package jwt

import (
	"encoding/json"
	"errors"
	"html/template"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"github.com/golang-jwt/jwt/v4"
)

// Custom JWT Claims.
type Claims struct {
	// Custom claims.
	IsAdmin    bool   `json:"op,omitempty"`
	VIP        bool   `json:"vip,omitempty"`
	Avatar     string `json:"img,omitempty"`
	ProfileURL string `json:"url,omitempty"`
	Nick       string `json:"nick,omitempty"`
	Emoji      string `json:"emoji,omitempty"`
	Gender     string `json:"gender,omitempty"`
	Rules      Rules  `json:"rules,omitempty"`

	// Standard claims. Notes:
	// subject = username
	jwt.RegisteredClaims
}

// ToJSON serializes the claims to JavaScript.
func (c Claims) ToJSON() template.JS {
	data, _ := json.Marshal(c)
	return template.JS(data)
}

// ParseAndValidate returns the Claims, a boolean authOK, and any errors.
func ParseAndValidate(tokenStr string) (*Claims, bool, error) {
	// Handle a JWT authentication token.
	var (
		claims = &Claims{}
		authOK bool
	)
	if tokenStr != "" {
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Current.JWT.SecretKey), nil
		})
		if err != nil {
			return nil, false, err
		}

		if parsed, ok := token.Claims.(*Claims); ok && token.Valid {
			claims = parsed
			authOK = true
		} else {
			return nil, false, errors.New("claims did not parse OK")
		}
	}

	return claims, authOK, nil
}

// ReSign will sign a new JWT token for existing claims. The chat server does this to send refreshed tokens
// to the front-end so the server can reboot gracefully, clients reconnect and not be told their auth had
// expired. New token expires after 5 minutes.
func (c Claims) ReSign() (string, error) {
	// Refresh timestamps.
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(5 * time.Minute))
	c.IssuedAt = jwt.NewNumericDate(time.Now())
	c.NotBefore = jwt.NewNumericDate(time.Now())

	// Generate the signed token and return it.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	ss, err := token.SignedString([]byte(config.Current.JWT.SecretKey))
	return ss, err
}
