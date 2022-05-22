package session

import (
	"fmt"
	"net/http"

	"authfish/internal/user"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "authfishSession"
	UserIdKey   = "userId"
)

func DeleteSession(rw http.ResponseWriter, r *http.Request, store sessions.Store) {
	// Ignoring error on purpose, existing session might be invalid
	session, _ := store.Get(r, SessionName)

	// Clear all data from session
	session.Values = make(map[interface{}]interface{})

	// Delete the session by setting max age to -1
	session.Options.MaxAge = -1

	session.Save(r, rw)
}

func DeleteSessionAndRedirectToLogin(rw http.ResponseWriter, r *http.Request, store sessions.Store) {
	DeleteSession(rw, r, store)
	http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
}

func SetUserSession(rw http.ResponseWriter, r *http.Request, store sessions.Store, user user.User) error {
	// Ignoring error on purpose, existing session might be invalid
	session, _ := store.Get(r, SessionName)

	// Clear all data from session
	session.Values = make(map[interface{}]interface{})

	session.Values[UserIdKey] = user.Id

	// 10 year expiration
	session.Options.MaxAge = 10 * 365 * 24 * 3600

	return session.Save(r, rw)
}

func GetUserIdFromSession(rw http.ResponseWriter, r *http.Request, store sessions.Store) (int64, error) {
	session, err := store.Get(r, SessionName)

	if err != nil {
		return 0, err
	}

	rawUserId, ok := session.Values[UserIdKey]

	if !ok {
		return 0, fmt.Errorf("could not access user ID in session using key '%s'", UserIdKey)
	}

	userId, ok := rawUserId.(int64)

	if !ok {
		return 0, fmt.Errorf("could not cast raw user id (%#v) to int64", rawUserId)
	}

	return userId, nil
}
