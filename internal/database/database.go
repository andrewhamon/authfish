package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"authfish/internal/user"
	"authfish/internal/utils"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

var migrations []string = []string{
	`
		create table if not exists users (
			id integer not null primary key,
			username text not null unique,
			hashed_password blob,
			registration_token text unique,
			created_at timestamp default current_timestamp not null,
			updated_at timestamp default current_timestamp not null
		);
	`,

	`
	  create trigger if not exists set_updated_at after update on users for each row begin
		  update users set updated_at = current_timestamp where id = old.id;
		end;
	`,
}

const (
	registrationTokenSize = 16
)

func OpenDB(file string) *sqlx.DB {
	return sqlx.MustConnect("sqlite3", file)
}

func RunMigrations(db *sqlx.DB) {
	for _, migration := range migrations {
		_ = db.MustExec(migration)
	}
}

func FindUserByUsername(db *sqlx.DB, username string) (*user.User, error) {
	username = utils.NormalizeUsername(username)

	users := make([]user.User, 0, 1)

	err := db.Select(&users, "select * from users where username = ? limit 1", username)

	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, nil
	}

	return &users[0], nil
}

func FindUserById(db *sqlx.DB, id int64) (*user.User, error) {
	users := []user.User{}

	err := db.Select(&users, "select * from users where id = ? limit 1", id)
	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, nil
	}

	return &users[0], nil
}

func FindUserByRegistrationToken(db *sqlx.DB, registrationToken string) (*user.User, error) {
	users := []user.User{}

	err := db.Select(&users, "select * from users where registration_token = ? limit 1", registrationToken)
	if err != nil {
		return nil, err
	}

	if len(users) != 1 {
		return nil, nil
	}

	return &users[0], nil
}

func RegisterNewUser(db *sqlx.DB, username string) (*user.User, error) {
	username = utils.NormalizeUsername(username)

	existingUser, err := FindUserByUsername(db, username)

	if err != nil {
		return nil, fmt.Errorf("error querying database: %w", err)
	}

	if existingUser != nil {
		return nil, fmt.Errorf("user %s already exists (id: %d)", existingUser.Username, existingUser.Id)
	}

	registrationTokenBytes := make([]byte, registrationTokenSize)
	numRead, err := rand.Read(registrationTokenBytes)

	if err != nil {
		return nil, fmt.Errorf("could not generate random registrationToken: %w", err)
	}

	if numRead != registrationTokenSize {
		return nil, fmt.Errorf(
			"tried to generate random registrationToken with %d bytes, but only %d random bytes were read",
			registrationTokenSize,
			numRead,
		)
	}

	registrationTokenHex := hex.EncodeToString(registrationTokenBytes)

	sqlResult, err := db.Exec(
		"insert into users (username, registration_token) values ($1, $2)",
		username,
		registrationTokenHex,
	)

	if err != nil {
		return nil, fmt.Errorf("error inserting new user into the database: %w", err)
	}

	id, err := sqlResult.LastInsertId()

	if err != nil {
		return nil, fmt.Errorf("error retrieving ID of newly inserted user: %w", err)
	}

	newUser := &user.User{
		Id:                id,
		Username:          username,
		HashedPassword:    []byte{},
		RegistrationToken: &registrationTokenHex,
	}

	return newUser, nil
}

func CompleteRegistration(db *sqlx.DB, userId int64, registrationToken string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	result, err := db.Exec("update users set hashed_password = ?, registration_token = null where id = ? and registration_token = ?", hashedPassword, userId, registrationToken)

	if err != nil {
		return err
	}

	updatedCount, err := result.RowsAffected()

	if err != nil {
		return err
	}

	if updatedCount != 1 {
		return fmt.Errorf("expected to update exactly 1 user, but instead updated %d users", updatedCount)
	}

	return nil
}

func DeleteUser(db *sqlx.DB, username string) (int64, error) {
	username = utils.NormalizeUsername(username)

	result, err := db.Exec("delete from users where username = ?", username)

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func ListUsers(db *sqlx.DB) ([]user.User, error) {
	users := []user.User{}

	err := db.Select(&users, "select * from users")

	if err != nil {
		return nil, err
	}

	return users, nil
}
