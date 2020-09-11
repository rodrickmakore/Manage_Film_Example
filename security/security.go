package security

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"skaffolder/Manage_Film_Example/config"
	"skaffolder/Manage_Film_Example/utils"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/globalsign/mgo/bson"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
)

type Config struct {
	*config.Config
}

func New(configuration *config.Config) *Config {
	return &Config{configuration}
}

func HasRole(roles ...string) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {

			// Get user from token
			header := req.Header.Get("Authorization")
			token := strings.Replace(header, "Bearer ", "", -1)
			user, err := getUserFromToken(token)

			// Not valid token
			if err != nil {
				http.Error(writer, http.StatusText(401), 401)
				return
			}

			// Check roles
			if checkRole(user, roles) != true {
				http.Error(writer, http.StatusText(401), 401)
				return
			}

			// Store user in context
			ctx := context.WithValue(req.Context(), "User", user)

			next.ServeHTTP(writer, req.WithContext(ctx))
		})
	}
}

func checkRole(user User, roles []string) bool {

	// Allow all to ADMIN
	if utils.Index(user.Roles, "ADMIN") != -1 {
		return true
	}

	// All to all users
	if len(roles) == 0 {
		return true
	}

	// Check roles
	for _, userRole := range user.Roles {
		for _, apiRole := range roles {
			if apiRole == userRole {
				return true
			}
		}
	}

	return false
}

// Routes
func (config *Config) Routes() *chi.Mux {
	router := chi.NewRouter()

	router.Post("/login", config.login)
	router.Post("/verifyToken", config.verifyToken)
	router.Group(func(router chi.Router) {
		router.Use(HasRole())
		router.Post("/changePassword", config.changePassword)
	})

	return router
}

/**
* Login API
 */
func (config *Config) login(writer http.ResponseWriter, req *http.Request) {

	var result User
	var body User
	json.NewDecoder(req.Body).Decode(&body)
	// Search User
	err := config.Database.C("User").Find(bson.M{"username": body.Username, "password": body.Password}).One(&result)

	if err != nil {
		// Unauthorized
		render.Status(req, 401)
		render.JSON(writer, req, nil)
	} else {
		// Create JWT token
		var tokenAuth *jwtauth.JWTAuth
		tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)
		result.Password = ""
		var userJson, _ = json.Marshal(result)
		var userJsonString = string(userJson)
		_, tokenString, _ := tokenAuth.Encode(jwt.MapClaims{
			"user": userJsonString,
		})

		result.Token = tokenString
		render.JSON(writer, req, result)
	}
}

func getUserFromToken(token string) (jwtUser User, err error) {

	var tokenAuth *jwtauth.JWTAuth
	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)

	if token != "" {
		var jwtObj *jwt.Token
		jwtObj, err = tokenAuth.Decode(token)
		if claims, ok := jwtObj.Claims.(jwt.MapClaims); ok && jwtObj.Valid {
			keysBody := []byte(claims["user"].(string))
			json.Unmarshal(keysBody, &jwtUser)
		} else {
			err = errors.New("Token not valid")
		}
	} else {
		err = errors.New("No Token provided")
	}

	return jwtUser, err
}

/**
* Verify Token
 */
func (config *Config) verifyToken(writer http.ResponseWriter, req *http.Request) {

	var body User
	json.NewDecoder(req.Body).Decode(&body)

	// Get user
	jwtUser, err := getUserFromToken(body.Token)

	if err != nil {
		render.Status(req, 403)
	}

	render.JSON(writer, req, jwtUser)
}

/**
 * UserService.changePassword
 *   @description Change password of user from admin
 *   @returns object
 *
 */
func (config *Config) changePassword(writer http.ResponseWriter, req *http.Request) {

	// Get body vars
	type PwdUser struct {
		PasswordNew string `json:"passwordNew"`
		PasswordOld string `json:"passwordOld"`
	}
	var body PwdUser
	json.NewDecoder(req.Body).Decode(&body)

	// Get user from contect
	ctx := req.Context()
	userLogged, _ := ctx.Value("User").(User)

	// Check old password
	var result User
	err := config.Database.C("User").Find(bson.M{"username": userLogged.Username, "password": body.PasswordOld}).One(&result)
	if err != nil {
		render.Status(req, 403)
		err = errors.New("Old password not valid")
		render.JSON(writer, req, nil)
		return
	}

	// Update password
	config.Database.C("User").UpdateId(result.ID, bson.M{"$set": bson.M{"password": body.PasswordNew}})

	render.JSON(writer, req, nil)
}
