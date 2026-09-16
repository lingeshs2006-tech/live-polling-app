package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var jwtSecret []byte

func InitAuth(secret string) {
	jwtSecret = []byte(secret)
}

func GenerateToken(userID primitive.ObjectID, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID.Hex(),
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Claims is the parsed identity of an authenticated user.
type Claims struct {
	UserID   primitive.ObjectID
	Username string
}

// RequireAuth protects routes: it fails the request unless a valid Bearer
// token is present, and stores the identity on the gin context.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}

		oid, err := primitive.ObjectIDFromHex(sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}

		c.Set("userID", oid)
		c.Set("username", claims["username"])
		c.Next()
	}
}

func UserID(c *gin.Context) primitive.ObjectID {
	v, _ := c.Get("userID")
	if v == nil {
		return primitive.NilObjectID
	}
	return v.(primitive.ObjectID)
}

func Username(c *gin.Context) string {
	v, _ := c.Get("username")
	if v == nil {
		return ""
	}
	return v.(string)
}