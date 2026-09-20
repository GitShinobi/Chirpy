package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)
 
type BearerTokenParam struct{
	headers http.Header
	expectedToken string
	expectError   bool
}
type tokenTestParam struct{
	userId uuid.UUID
	expire time.Duration
	tokenString string
	tokenSecret string
}

func testValidHashPassword(password string)(error){
	hashedPassword, err := HashPassword(password)
	if err != nil{
		return fmt.Errorf("error hashing password: %v",err)
	}
	match,err := CheckPasswordHash(password, hashedPassword)

	if err != nil{
		return fmt.Errorf("error matching passwords: %v",err)
	}
	if !match{
		return fmt.Errorf("error incorrect passwords ")
	}
	return nil
}
func testInValidHashPassword(password string)(error){
	_, err := HashPassword(password)
	if err != nil{
		return fmt.Errorf("error hashing password: %v",err)
	}
	wrongHash, _ := HashPassword("correct-password")
	match,err := CheckPasswordHash(password, wrongHash)
	if err != nil{
		return fmt.Errorf("error matching passwords: %v",err)
	}
	if match{
		return fmt.Errorf("expected passwords not to match")
	}
	return nil
}

func TestPwdSuit(t *testing.T){
	pwds := []string{}
	for i := range(5){
		pwd := fmt.Sprintf("password num_%d",i)
		pwds = append(pwds, pwd)
	} 

	for i,pwd := range(pwds){
		t.Run(fmt.Sprintf("validPwd_%d",i),func(t *testing.T) {
			displayIfError(t, testValidHashPassword(pwd))

		})
		t.Run(fmt.Sprintf("inValidPwd_%d",i),func(t *testing.T) {
			displayIfError(t, testInValidHashPassword(pwd))

		})
	} 
}
func testValidJwt(param tokenTestParam)(error){
	id, err := ValidateJWT(param.tokenString,param.tokenSecret)
	if err != nil{
		return fmt.Errorf("error validating token: %v",err)
	}
	if id != param.userId{
		return fmt.Errorf("expected parsed ID %s to match original ID %s", id, param.userId)
	}
	return nil
}
func testExpiredToken(param tokenTestParam)(error){
	expiredString, err := MakeJWT(param.userId, param.tokenSecret, -time.Second)
	if err != nil {
		return fmt.Errorf("error making expired JWT: %v", err)
	}	
	_, err = ValidateJWT(expiredString,param.tokenSecret)
	if err == nil{
		return fmt.Errorf("error expired token: %v",err)
	}
	return nil
}

func testInvalidTokenSecret(param tokenTestParam)(error){
	falseTokenSecret := param.tokenSecret+"4"
	_, err := ValidateJWT(param.tokenString,falseTokenSecret)
	if err == nil{
		return fmt.Errorf("error validating invalid Token: %v",err)
	}
	return nil
}

func testMalformedToken(param tokenTestParam)(error){
	_, err := ValidateJWT("false token",param.tokenSecret)
	if err == nil{
		return fmt.Errorf("error validating Malformed token: %v",err)
	}
	return nil
}

func displayIfError(t *testing.T, err error){
	if err != nil{
		t.Error(err.Error())
	}
}

func TestJWTSuite(t *testing.T){
	params := [] tokenTestParam{}
	for range(5){
		tokenSecret := uuid.NewString()
		userId, err := uuid.NewRandom()
		if err != nil{
			t.Errorf("error creating userId: %v",err)
		}
		expire := time.Minute
		tokenString,err := MakeJWT(userId, tokenSecret, expire)
		if err != nil {
			t.Fatalf("error making JWT: %v", err)
		}
		param := tokenTestParam{
		userId: userId,
		expire: expire,
		tokenString: tokenString,
		tokenSecret: tokenSecret,
		}
		params = append(params, param)
	}

	for i,param := range params{
		t.Run(fmt.Sprintf("ValidJWT_%d", i), func(t *testing.T) {
			displayIfError(t, testValidJwt(param))
		})
		t.Run(fmt.Sprintf("ExpiredToken_%d", i), func(t *testing.T) {
			displayIfError(t, testExpiredToken(param))
		})
		t.Run(fmt.Sprintf("InvalidSecret_%d", i), func(t *testing.T) {
			displayIfError(t, testInvalidTokenSecret(param))
		})
		t.Run(fmt.Sprintf("MalformedToken_%d", i), func(t *testing.T) {
			displayIfError(t, testMalformedToken(param))
		})
	}
}

func testGetBearerToken(bearertoken BearerTokenParam )(error){
	token,err := GetBearerToken(bearertoken.headers)
	if bearertoken.expectError {
        if err == nil {
            return fmt.Errorf("expected an error, but got token: %s", token)
        }
        return nil
    }
	if err != nil {
        return fmt.Errorf("unexpected error: %v", err)
    }
	if  token != bearertoken.expectedToken{
			return fmt.Errorf("error expected token %s to match original token %s ",token,bearertoken.expectedToken)
	}
	return nil
}


func TestBearerTokenSuite(t *testing.T){
	header := http.Header{}
	tokens := []BearerTokenParam{}
	header.Set("Authorization","Bearer ")
	tokens = append(tokens, BearerTokenParam{
				headers: header.Clone(),
				expectedToken: "error token don't exist",
				expectError: true,
	})
	header.Set("Authorization","Bearer 1234567890")
	tokens = append(tokens, BearerTokenParam{
				headers: header.Clone(),
				expectedToken: "1234567890",
				expectError: false,
	})
	for range (7){
		tokenStr := uuid.NewString()
		token := fmt.Sprintf("Bearer %s",tokenStr)
		header.Set("Authorization",token)
		tokens = append(tokens, BearerTokenParam{
				headers: header.Clone(),
				expectedToken: tokenStr,
				expectError: false,
	})
	}
	header.Set("Authorization","Bearer azertyuiop")
	tokens = append(tokens, BearerTokenParam{
				headers: header.Clone(),
				expectedToken: "azertyuiop",
				expectError: false,
	})
	for i, token := range tokens{
		t.Run(fmt.Sprintf("GetBearerToken_%d",i),func(t *testing.T) {
			displayIfError(t, testGetBearerToken(token))
		})
	}
}