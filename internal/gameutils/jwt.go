package gameutils

import (
	"gofishing-game/internal/errcode"
	"quasar/config"

	"github.com/golang-jwt/jwt/v4"
)

var defaultJWT *JWT

var errMalformedToken = errcode.New("malformed_jwt_token", "malformed jwt token")
var errInvalidToken = errcode.New("invalid_jwt_token", "invalid jwt token")
var errExpiredToken = errcode.New("expired_jwt_token", "expired jwt token")
var errUnavailableToken = errcode.New("unavailable_jwt_token", "unavailable jwt token")
var errUnknowToken = errcode.New("unknow_jwt_token", "unknow jwt token")

type JWT struct {
	key []byte
}

func init() {
	defaultJWT = &JWT{key: []byte(config.Config().ServerKey)}
}

type CustomClaims struct {
	Uid int `json:"uid"`
	jwt.RegisteredClaims
}

func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.key)
}

func (j *JWT) ParserToken(tokenString string) (*CustomClaims, errcode.Error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return j.key, nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errMalformedToken
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errExpiredToken
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errInvalidToken
			} else {
				return nil, errUnavailableToken
			}
		}
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errUnknowToken
}

func Validate(uid int, token string) errcode.Error {
	if token == "" {
		return errcode.New("empty_token", "empty token")
	}
	claims, err := defaultJWT.ParserToken(token)
	if err != nil {
		return err
	}
	if claims.Uid != uid {
		return errcode.New("not_match_user", "not match user")
	}
	return nil
}
