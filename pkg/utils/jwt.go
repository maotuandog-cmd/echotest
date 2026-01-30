package utils

import (
	"echotest/config"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"time"
)

// JWT 返回基于 echo-jwt 的 JWT 认证中间件，兼容 Echo v5。
// secret 为签名密钥，与 pkg/jwt 生成 token 时使用的密钥一致（如 []byte(cfg.JWT.Secret)）。
// 校验通过后会在 context 中设置 "user"（*jwt.Token）、"email"、"userID"，便于与原有逻辑兼容。
func JWT(secret []byte) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:    secret,
		SigningMethod: "HS256",
		TokenLookup:   "header:Authorization:Bearer ",
		ContextKey:    "user",

		// 使用与 pkg/jwt 相同的 UserCliams，保证生成的 token 可被正确解析
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return &UserCliams{}
		},

		// 校验成功后设置 email、userID，兼容原有 c.Get("email") / c.Get("userID") 用法
		SuccessHandler: func(c *echo.Context) error {
			token, ok := c.Get("user").(*jwt.Token)
			if !ok {
				return nil
			}
			claims, ok := token.Claims.(*UserCliams)
			if !ok {
				return nil
			}
			c.Set("email", claims.Email)
			c.Set("userID", claims.UserID)
			return nil
		},
	})
}

type JWTS struct {
	secret   []byte
	duration time.Duration
}

func NewJWT(cfg config.JWTConfig) *JWTS {
	return &JWTS{
		secret:   []byte(cfg.Secret),
		duration: cfg.Duration,
	}
}

type UserCliams struct {
	Email  string `json:"email"`
	UserID int    `json:"user_id"`
	jwt.RegisteredClaims
}

func (j *JWTS) Generate(email string, useID int) (string, error) {
	claims := UserCliams{
		Email:  email,
		UserID: useID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWTS) ParseToken(tokenString string) (*UserCliams, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserCliams{}, func(t *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserCliams); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("failed to parseToken")
}
