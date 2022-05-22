package check

import (
	"authfish/internal/web/current_user"
	"authfish/internal/web/session"
	"log"

	_ "embed"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

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

	if err != nil {
		log.Printf("Encountered error checking for current user: %v", err)
		session.DeleteSession(rw, r, s.store)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if currentUser == nil {
		log.Printf("Current user not found")
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	rw.WriteHeader(http.StatusOK)
}
