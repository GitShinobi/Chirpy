package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error){
	return argon2id.CreateHash(password, argon2id.DefaultParams) 
}

func CheckPasswordHash(password, hash string) (bool, error){
	return argon2id.ComparePasswordAndHash(password,hash)}

	
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
			 jwt.RegisteredClaims{
			 Issuer: "chirpy-access",
			 IssuedAt : jwt.NewNumericDate(time.Now().UTC()),
			 ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			 Subject: userID.String(),
	})
	return  token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	keyfunc := func(token *jwt.Token)(any, error){
		return []byte(tokenSecret), nil
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},keyfunc)
	if err != nil{
		return uuid.Nil,err
	}
	subject,err := token.Claims.GetSubject()
	if err != nil{
		return uuid.Nil,err
	}
	id,err := uuid.Parse(subject) 
	if err != nil{
		return uuid.Nil,err
	}
	return id,nil
}
func GetBearerToken(headers http.Header) (string, error){
	Authorization  := headers.Get("Authorization")
	tokenTab := strings.Split(Authorization,"Bearer")
	if len(tokenTab) < 2{
		return "",fmt.Errorf("error token don't exist")
	}
	tokenStr := strings.TrimSpace(tokenTab[1])
	if len(tokenStr)==0{
		return "",fmt.Errorf("error token don't exist")
	}
	return tokenStr, nil
}

func MakeRefreshToken() string{
	key := make([]byte, 32)
	rand.Read(key)
	return  hex.EncodeToString(key)
}
func GetAPIKey(headers http.Header) (string, error){
	Authorization  := headers.Get("Authorization")
	tokenTab := strings.Split(Authorization,"ApiKey")
	if len(tokenTab) < 2{
		return "",fmt.Errorf("error token don't exist")
	}
	tokenStr := strings.TrimSpace(tokenTab[1])
	if len(tokenStr)==0{
		return "",fmt.Errorf("error token don't exist")
	}
	return tokenStr, nil

}