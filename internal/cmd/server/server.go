package server

import (
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"authfish/internal/context"
	"authfish/internal/web/check"
	"authfish/internal/web/current_user"
	"authfish/internal/web/login"
	"authfish/internal/web/me"
	"authfish/internal/web/register"
	"authfish/internal/web/session"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type ServerCmd struct {
	Host     string `help:"Hostname or IP address to listen on, or path to socket if --protocol=unix" default:"127.0.0.1"`
	Port     int    `help:"Port to listen on. Only applies when --protocol=tcp (the default)" default:"8080"`
	Protocol string `help:"One of tcp,unix" default:"tcp" enum:"tcp,unix"`
}

func (r *ServerCmd) Run(ctx *context.AppContext) error {
	listenAddress := buildListenAddress(r.Host, r.Port, r.Protocol)
	listener, err := net.Listen(r.Protocol, listenAddress)

	log.Printf("Authfish server listening on %s://%s", r.Protocol, listenAddress)

	if err != nil {
		return err
	}

	var sessionStore = sessions.NewCookieStore(generateSecretAuthToken(), generateSecretEncryptionToken())
	sessionStore.Options.SameSite = http.SameSiteStrictMode
	sessionStore.Options.Secure = true
	sessionStore.Options.HttpOnly = true

	handler := handlers.RecoveryHandler()(
		handlers.CombinedLoggingHandler(
			os.Stdout,
			current_user.AddCurrentUserToRequestContext(
				ctx.Db,
				sessionStore,
				buildRoutes(ctx.Db, sessionStore),
			),
		),
	)

	return http.Serve(listener, handler)
}

// As per the gorilla session docs, use 32 or 64 bytes. Going with 64.
func generateSecretAuthToken() []byte {
	secretAuthToken := make([]byte, 64)
	numRead, err := rand.Read(secretAuthToken)
	if err != nil {
		panic(err)
	}

	if numRead != 64 {
		panic("Did not read enough random bytes into secretAuthToken")
	}

	return secretAuthToken
}

// As per the gorilla session docs, 32 bytes selects AES-256
func generateSecretEncryptionToken() []byte {
	secretEncryptionToken := make([]byte, 32)
	numRead, err := rand.Read(secretEncryptionToken)
	if err != nil {
		panic(err)
	}

	if numRead != 32 {
		panic("Did not read enough random bytes into secretEncryptionToken")
	}

	return secretEncryptionToken
}

func buildRoutes(db *sqlx.DB, store sessions.Store) *mux.Router {
	r := mux.NewRouter()

	registrationHandler := register.New(store, db)
	r.Handle("/register", registrationHandler)

	loginHandler := login.New(store, db)
	r.Handle("/login", loginHandler)

	checkHandler := check.New(store, db)
	r.Handle("/check", checkHandler)

	meHandler := me.New(store, db)
	r.Handle("/me", meHandler)

	r.Handle("/", meHandler)

	r.HandleFunc("/logout", func(rw http.ResponseWriter, r *http.Request) {
		session.DeleteSessionAndRedirectToLogin(rw, r, store)
	})

	return r
}

func buildListenAddress(host string, port int, protocol string) string {
	switch protocol {
	case "tcp":
		return fmt.Sprintf("%s:%d", host, port)
	case "unix":
		return host
	default:
		panic(fmt.Errorf("Unknown protocol %s", protocol))
	}
}
