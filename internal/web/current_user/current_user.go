package current_user

import (
	"authfish/internal/database"
	"authfish/internal/user"
	"authfish/internal/web/session"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

// Try to find the current user based on the session cookie. If any errors are
// encountered, delete the session, but otherwise do nothing. HTTP handlers
// are required to check for an authenticated user in the request context.
func AddCurrentUserToRequestContext(db *sqlx.DB, store sessions.Store, handler http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userFromSession, err := findUserFromSession(db, store, rw, r)
		if err == nil && userFromSession != nil {
			setUserContextAndServe(userFromSession, handler, rw, r)
			return
		}

		userFromBasicAuth, err := findUserFromBasicAuth(db, store, rw, r)
		if err == nil && userFromBasicAuth != nil {
			setUserContextAndServe(userFromBasicAuth, handler, rw, r)
			return
		}

		userFromBearerToken, err := findUserFromBearerToken(db, store, rw, r)
		if err == nil && userFromBearerToken != nil {
			setUserContextAndServe(userFromBearerToken, handler, rw, r)
			return
		}

		handler.ServeHTTP(rw, r)
	}
}

func CurrentUser(context context.Context) (*user.User, error) {
	rawCurrentUser := context.Value(user.CurrentUserContextKey)

	if rawCurrentUser == nil {
		return nil, nil
	}

	currentUser, ok := rawCurrentUser.(*user.User)

	if !ok {
		log.Printf("Error fetching current user. Could not cast %#v to *user.User", rawCurrentUser)
		return nil, fmt.Errorf("error casting user context to user")
	}

	return currentUser, nil
}

func findUserFromSession(db *sqlx.DB, store sessions.Store, rw http.ResponseWriter, r *http.Request) (*user.User, error) {
	userId, err := session.GetUserIdFromSession(rw, r, store)

	if err != nil {
		return nil, err
	}

	return database.FindUserById(db, userId)
}

func findUserFromBasicAuth(db *sqlx.DB, store sessions.Store, rw http.ResponseWriter, r *http.Request) (*user.User, error) {
	_, apiKey, ok := r.BasicAuth()

	if !ok {
		return nil, fmt.Errorf("basic auth credentials not found")
	}

	return database.FindUserByApiKey(db, apiKey)
}

func findUserFromBearerToken(db *sqlx.DB, store sessions.Store, rw http.ResponseWriter, r *http.Request) (*user.User, error) {
	reqToken := strings.TrimSpace(r.Header.Get("Authorization"))
	tokenParts := strings.SplitN(reqToken, "Bearer", 2)
	if len(tokenParts) != 2 {
		return nil, fmt.Errorf("bearer token not found")
	}
	apiKey := strings.TrimSpace(tokenParts[1])

	return database.FindUserByApiKey(db, apiKey)
}

func setUserContextAndServe(u *user.User, handler http.Handler, rw http.ResponseWriter, r *http.Request) {
	newContext := context.WithValue(r.Context(), user.CurrentUserContextKey, u)
	handler.ServeHTTP(rw, r.WithContext(newContext))
}
