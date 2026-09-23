# Chirpy
Chirpy is a RESTful API server built from scratch in go
# Motivation
the goal with `Chirpy` is meant to be a reference for understanding how RESTful API server works under the hood
# installation
inside a Go module

```
go get https://github.com/GitShinobi/Chirpy.git
```
# 🚀 Quick Start Consumer
Take a note of the .env file 
```
DB_URL="postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
PLATFORM="dev"
SECRET_KEY="zsDkefR7CjTMyHDUiuP/vc2BkLgrGwjdaGZF6lkVEH9IIdXD3jznfWErHRhj49YpbYUz+/xs1nNiCoWHRxKTDg=="
POLKA_KEY="f271c81ff7084ee5b99a5091b42d486e"
```
the Key `PLATFORM` is meant to authenticate the admin user in the handlerReset function to prevent any user from deleting all users in the database
```
func (cfg *apiConfig) handlerReset(writer http.ResponseWriter, req *http.Request) {
	if cfg.Platform != "dev" {
		writer.WriteHeader(403)
		return
	}
	err := cfg.Db.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(writer, 400, "Error deleting users")
		return
	}
	cfg.FileserverHits.Store(0)
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
}
```
the key `SECRET_KEY` is used in the MakeJWT function to generate the server token used for communicating with the client
```
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
```
the key `POLKA_KEY` is used to secure a request for a membership upgrade by generating a token
```

func (cfg *apiConfig) handlerWebhooks(writer http.ResponseWriter, req *http.Request) {
	polkaKey,err := auth.GetAPIKey(req.Header) 
	if err != nil{
		respondWithError(writer, 401, err.Error())
		return
	}
	if polkaKey != cfg.Polka{
		respondWithError(writer, 401, "incorect API key")
		return
	}
	decoder := json.NewDecoder(req.Body)
	params := webhooksParams{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, 400, err.Error())
		return
	}
	if params.Event != "user.upgraded"{
		writer.WriteHeader(204)
		return
	}
	userId := params.Data.UserID
	err = cfg.Db.UpdateChirpyRedMembership(context.Background(),userId)
	if err != nil {
		respondWithError(writer, 404, fmt.Sprintf("Error updating membership %s", err.Error()))
		return
	}
	writer.WriteHeader(204)

}
```
# Usage
chirpy support multiple commands for differents endpoints.
Example : `Get /app/ ` redirect to the root of the restfull api server
## /admin/
this root is only allowed for priviledged authanticated users 
### /metrics
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|GET | /admin/metrics | display the number of pages's visit | 
### /reset
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|GET | /admin/reset | delete all the users of the database | 
## /api/
this root is allowed for every authanticated users 
### /healthz
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|GET | /api/healthz | is used to test the http command to the restfull api server| 
### /users
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST | /api/users |create and save in the database a new user account through data sent in the body | 
|PUT| /api/users |update a user account through data sent in the body | 
### /login
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST| /api/login |authanticate a user account through data sent in the body |  
### /refresh
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST | /api/refresh |check the validity of the current token before creating a new one| 
### /revoke
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST | /api/revoke |invalidate a token sent in the header| 
### /chirps
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST | /api/chirps |create a new chirp through data sent in the header| 
|GET | /api/chirps  |get all chirps from the database| 
|GET | /api/chirps{chirpID} |get a chirps from the database by his id sent in the URL|
|DELETE | /api/chirps{chirpID}|delete a chirps from the database by his id sent in the URL|
### /polka
| Methods | Endpoints | Description |
| :--- | :--- | :--- | 
|POST |/api/polka/webhooks | upgrate an user account after cheking the webhook token from the header | 
# Stability
That RESTful API server is currently in v0. And don't plan to develop it further in the futur
# 💬 Contact
[My YouTube Channel](https://www.youtube.com/@ShaderMonk)