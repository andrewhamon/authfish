package login

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"authfish/internal/database"
	"authfish/internal/user"
	"authfish/internal/utils"
	"authfish/internal/web/current_user"
	"authfish/internal/web/session"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var (
	//go:embed loginForm.template.html
	templateString string
	parsedTemplate *template.Template = template.Must(template.New("login").Parse(templateString))
)

type templateVars struct {
	Username  string
	Password  string
	Redirect  string
	Loginpath string
	Errors    []error
}

func renderTemplate(rw http.ResponseWriter, status int, vars templateVars) {
	rw.WriteHeader(status)
	parsedTemplate.Execute(rw, vars)
}

type Service struct {
	store   sessions.Store
	db      *sqlx.DB
	domains []string
}

func New(store sessions.Store, db *sqlx.DB, domains []string) *Service {
	return &Service{
		store:   store,
		db:      db,
		domains: domains,
	}
}

func (s *Service) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	currentUser, _ := current_user.CurrentUser(r.Context())

	originalUrl := r.Header.Get("X-Original-URL")

	parsedURL, err := url.ParseRequestURI(originalUrl)

	if err == nil && parsedURL.RequestURI() == r.URL.RequestURI() {
		originalUrl = "/"
	}

	if len(originalUrl) == 0 {
		originalUrl = "/"
	}

	loginpath := r.Header.Get("X-Authfish-Login-Path")
	if len(loginpath) == 0 {
		loginpath = "/login"
	}

	if currentUser != nil {
		http.Redirect(rw, r, originalUrl, http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(rw, http.StatusOK, templateVars{
			Redirect:  originalUrl,
			Loginpath: loginpath,
		})
		return
	}

	if r.Method != http.MethodPost {
		errMessage := fmt.Sprintf("Method %s is not allowed. Try GET or POST", r.Method)
		http.Error(rw, errMessage, http.StatusMethodNotAllowed)
		return
	}

	username := utils.NormalizeUsername(r.FormValue("username"))
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")
	loginpath = r.FormValue("loginpath")

	currentUser, err = checkLogin(s.db, username, password)

	if err != nil {
		renderTemplate(rw, http.StatusUnauthorized, templateVars{
			Username:  username,
			Password:  password,
			Redirect:  redirect,
			Loginpath: loginpath,
			Errors:    []error{err},
		})
		return
	}

	if err := session.SetUserSession(rw, r, s.store, s.domains, *currentUser); err != nil {
		renderTemplate(rw, http.StatusInternalServerError, templateVars{
			Username:  username,
			Password:  password,
			Redirect:  redirect,
			Loginpath: loginpath,
			Errors:    []error{fmt.Errorf("could not save user session: %w", err)},
		})
		return
	}

	http.Redirect(rw, r, redirect, http.StatusFound)
}

func checkLogin(db *sqlx.DB, username string, password string) (*user.User, error) {
	u, err := database.FindUserByUsername(db, username)

	if err != nil {
		return nil, fmt.Errorf("error running database query: %w", err)
	}

	if u == nil {
		return nil, fmt.Errorf("username not registered: %s", username)
	}

	err = bcrypt.CompareHashAndPassword(u.HashedPassword, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password for %s", username)
	}

	return u, nil
}
