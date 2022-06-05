package register

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"authfish/internal/database"
	"authfish/internal/web/session"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed register.template.html
	templateString string
	parsedTemplate *template.Template = template.Must(template.New("register").Parse(templateString))
)

const (
	MinimumPasswordLength = 6
)

type templateVars struct {
	Username          string
	Password          string
	ConfirmPassword   string
	RegistrationToken string
	Errors            []error
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

func (s *Service) showRegistration(rw http.ResponseWriter, r *http.Request) {
	registrationToken := strings.TrimSpace(r.URL.Query().Get("registrationToken"))

	user, err := database.FindUserByRegistrationToken(s.db, registrationToken)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		renderTemplate(rw, http.StatusInternalServerError, templateVars{
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("error querying database: %w", err)},
		})

		return
	}

	if user == nil {
		renderTemplate(rw, http.StatusUnauthorized, templateVars{
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("registration token not valid")},
		})
		return
	}

	renderTemplate(rw, http.StatusOK, templateVars{
		Username:          user.Username,
		RegistrationToken: registrationToken,
	})
}

func (s *Service) handleRegistration(rw http.ResponseWriter, r *http.Request) {
	registrationToken := r.FormValue("registrationToken")
	usernameFromForm := r.FormValue("username")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirmPassword")

	user, err := database.FindUserByRegistrationToken(s.db, registrationToken)

	if err != nil {
		renderTemplate(rw, http.StatusBadRequest, templateVars{
			Username:          usernameFromForm,
			Password:          password,
			ConfirmPassword:   confirmPassword,
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("error querying database: %w", err)},
		})
	}

	if user == nil {
		renderTemplate(rw, http.StatusUnauthorized, templateVars{
			Username:          usernameFromForm,
			Password:          password,
			ConfirmPassword:   confirmPassword,
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("registration token not valid")},
		})
		return
	}

	passwordErrors := validatePasswords(password, confirmPassword)
	if len(passwordErrors) > 0 {
		renderTemplate(rw, http.StatusBadRequest, templateVars{
			Username:          user.Username,
			Password:          password,
			ConfirmPassword:   confirmPassword,
			RegistrationToken: registrationToken,
			Errors:            passwordErrors,
		})
		return
	}

	err = database.CompleteRegistration(s.db, user.Id, registrationToken, password)

	if err != nil {
		renderTemplate(rw, http.StatusInternalServerError, templateVars{
			Username:          user.Username,
			Password:          password,
			ConfirmPassword:   confirmPassword,
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("could not create user: %w", err)},
		})
		return
	}

	if err := session.SetUserSession(rw, r, s.store, s.domains, *user); err != nil {
		renderTemplate(rw, http.StatusInternalServerError, templateVars{
			Username:          user.Username,
			Password:          password,
			ConfirmPassword:   confirmPassword,
			RegistrationToken: registrationToken,
			Errors:            []error{fmt.Errorf("could not save user session: %w", err)},
		})
		return
	}

	http.Redirect(rw, r, "/me", http.StatusFound)
}

func (s *Service) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.showRegistration(rw, r)
		return
	}

	if r.Method != http.MethodPost {
		errMessage := fmt.Sprintf("Method %s is not allowed. Try GET or POST", r.Method)
		http.Error(rw, errMessage, http.StatusMethodNotAllowed)
		return
	}

	s.handleRegistration(rw, r)
}

func validatePasswords(password string, confirmPassword string) []error {
	errors := []error{}

	if err := checkPasswordsMatch(password, confirmPassword); err != nil {
		errors = append(errors, err)
	}

	if err := checkPasswordLength(password); err != nil {
		errors = append(errors, err)
	}

	return errors
}

func checkPasswordsMatch(password string, confirmPassword string) error {
	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func checkPasswordLength(password string) error {
	if len(password) < MinimumPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", MinimumPasswordLength)
	}

	return nil
}
