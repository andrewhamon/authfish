package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

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
	Host     string   `help:"Hello world !!! Hostname or IP address to listen on, or path to socket if --protocol=unix" default:"127.0.0.1"`
	Port     int      `help:"Port to listen on. Only applies when --protocol=tcp (the default)" default:"8080"`
	Protocol string   `help:"One of tcp,unix" default:"tcp" enum:"tcp,unix"`
	Domain   []string `help:"One or more domains to set cookies for. Must set X-Original-URL header when proxying. First domain which is a substring of the request host will be chosen."`
	Secure   bool     `help:"Set cookie to be secure (HTTPS) only. Defaults to secure." default:"true" negatable:""`
}

func (r *ServerCmd) Run(ctx *context.AppContext) error {
	listenAddress := buildListenAddress(r.Host, r.Port, r.Protocol)
	listener, err := net.Listen(r.Protocol, listenAddress)

	log.Printf("Authfish server listening on %s://%s", r.Protocol, listenAddress)

	if err != nil {
		return err
	}

	sk, err := generateSecretKey(ctx.DataDir)
	if err != nil {
		panic(err)
	}
	var sessionStore = sessions.NewCookieStore(sk.authToken[:], sk.encryptionToken[:])
	sessionStore.Options.SameSite = http.SameSiteStrictMode
	sessionStore.Options.Secure = r.Secure
	sessionStore.Options.HttpOnly = true

	handler := handlers.RecoveryHandler()(
		handlers.CombinedLoggingHandler(
			os.Stdout,
			current_user.AddCurrentUserToRequestContext(
				ctx.Db,
				sessionStore,
				buildRoutes(ctx.Db, sessionStore, r.Domain),
			),
		),
	)

	return http.Serve(listener, handler)
}

type secretKey struct {
	authToken       [64]byte // As per the gorilla session docs, use 32 or 64 bytes. Going with 64.
	encryptionToken [32]byte // As per the gorilla session docs, 32 bytes selects AES-256
}

func generateSecretKey(dataDir string) (secretKey, error) {
	sk := secretKey{}

	skPath := filepath.Join(dataDir, "secret_key")
	existingSk, err := readHexBytesFromFile(skPath, 96)

	if err == nil {
		copy(sk.authToken[:], existingSk[0:64])
		copy(sk.encryptionToken[:], existingSk[64:96])
		return sk, nil
	}

	newSkBytes := make([]byte, 96)
	numRead, err := rand.Read(newSkBytes)
	if err != nil {
		return sk, err
	}

	if numRead != 96 {
		return sk, fmt.Errorf("only generated %d random bytes, but wanted %d", numRead, 96)
	}

	err = writeBytesAsHexToFile(skPath, newSkBytes)
	if err != nil {
		return sk, err
	}

	copy(sk.authToken[:], newSkBytes[0:64])
	copy(sk.encryptionToken[:], newSkBytes[64:96])

	return sk, nil
}

func buildRoutes(db *sqlx.DB, store sessions.Store, domains []string) *mux.Router {
	r := mux.NewRouter()

	registrationHandler := register.New(store, db, domains)
	r.Handle("/register", registrationHandler)

	loginHandler := login.New(store, db, domains)
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
		panic(fmt.Errorf("unknown protocol %s", protocol))
	}
}

func readHexBytesFromFile(path string, numBytes int) ([]byte, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	// Because of the hex encoding, we need to read double the raw bytes
	b := make([]byte, numBytes*2)

	n, err := f.Read(b)

	if err != nil {
		return nil, err
	}

	if n != (numBytes * 2) {
		return nil, fmt.Errorf("expected to read %d*2 bytes from %s, but actually read %d", numBytes, path, n)
	}

	output := make([]byte, numBytes)

	n, err = hex.Decode(output, b)

	if err != nil {
		return nil, err
	}

	if n != numBytes {
		return nil, fmt.Errorf("expected to decode %d bytes from hex at path %s, but actually read %d", numBytes, path, n)
	}

	return output, nil
}

func writeBytesAsHexToFile(path string, b []byte) error {
	// We need double the lengh, because of hex encoding, + 1 more byte for a
	// newline.
	dataLength := len(b)*2 + 1
	data := make([]byte, dataLength)

	hex.Encode(data, b)

	// Add a newline to the end. Misconfigured terminal prompts can sometimes
	// eat lines without newlines. i.e. `cat file_with_no_newline` appears to be
	// empty when its not.
	data[len(b)*2] = '\n'

	err := os.WriteFile(path, data, 0600)
	if err != nil {
		return err

	}
	return nil
}
