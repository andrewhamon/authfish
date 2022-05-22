package current_user

import (
	"authfish/internal/database"
	"authfish/internal/user"
	"authfish/internal/web/session"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

// Try to find the current user based on the session cookie. If any errors are
// encountered, delete the session, but otherwise do nothing. HTTP handlers
// are required to check for an authenticated user in the request context.
func AddCurrentUserToRequestContext(db *sqlx.DB, store sessions.Store, handler http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userId, err := session.GetUserIdFromSession(rw, r, store)

		if err != nil {
			session.DeleteSession(rw, r, store)
			handler.ServeHTTP(rw, r)
			return
		}

		currentUser, err := database.FindUserById(db, userId)

		if err != nil {
			log.Printf("Valid session found, but encountered database error fetching user. Deleting session. Original error: %v", err)
			session.DeleteSession(rw, r, store)
			handler.ServeHTTP(rw, r)
			return
		}

		if currentUser == nil {
			log.Printf("Valid session found, but user %d no longer exists. Deleting session.", userId)
			session.DeleteSession(rw, r, store)
			handler.ServeHTTP(rw, r)
			return
		}

		newContex := context.WithValue(r.Context(), user.CurrentUserContextKey, currentUser)

		handler.ServeHTTP(rw, r.WithContext(newContex))
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
