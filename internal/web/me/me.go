package me

import (
	"authfish/internal/user"
	"authfish/internal/web/current_user"
	"authfish/internal/web/session"

	_ "embed"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed me.template.html
	templateString string
	parsedTemplate *template.Template = template.Must(template.New("me").Parse(templateString))
)

type templateVars struct {
	User *user.User
}

func renderTemplate(rw http.ResponseWriter, status int, vars templateVars) {
	rw.WriteHeader(status)
	parsedTemplate.Execute(rw, vars)
}

type Service struct {
	store sessions.Store
	db    *sqlx.DB
}

func New(store sessions.Store, db *sqlx.DB) *Service {
	return &Service{
		store: store,
		db:    db,
	}
}

func (s *Service) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	currentUser, err := current_user.CurrentUser(r.Context())

	if err != nil || currentUser == nil {
		session.DeleteSessionAndRedirectToLogin(rw, r, s.store)
		return
	}

	renderTemplate(rw, http.StatusOK, templateVars{
		User: currentUser,
	})
}
