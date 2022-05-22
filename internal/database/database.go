package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"authfish/internal/api_key"
	"authfish/internal/user"
	"authfish/internal/utils"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

var migrations []string = []string{
	`
		create table if not exists users (
			id                 integer not null primary key,
			username           text not null unique,
			hashed_password    blob,
			registration_token text unique,
			created_at         timestamp default current_timestamp not null,
			updated_at         timestamp default current_timestamp not null
		);
	`,

	`
	  create trigger if not exists set_updated_at after update on users for each row begin
		  update users set updated_at = current_timestamp where id = old.id;
		end;
	`,
	` 

	  create table if not exists api_keys (
			id         integer   not null primary key,
			user_id    integer   not null,
			memo       text      not null,
			key        text      unique,
			created_at timestamp default current_timestamp not null,
			last_seen  timestamp,

			FOREIGN KEY(user_id) REFERENCES users(id)
		);
	`,

	`
	  create index if not exists api_keys_user_id_idx ON api_keys (user_id);
	`,
}

const (
	registrationTokenSize = 16
)

func OpenDB(file string) *sqlx.DB {
	fileWithOptions := fmt.Sprintf("%s?_foreign_keys=1", file)
	return sqlx.MustConnect("sqlite3", fileWithOptions)
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

func FindUserByApiKey(db *sqlx.DB, apiKey string) (*user.User, error) {
	users := []user.User{}

	err := db.Select(&users,
		"select users.* from users inner join api_keys on users.id = api_keys.user_id where api_keys.key = ? limit 1",
		apiKey,
	)
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

	registrationTokenHex, err := generateRandomHex(registrationTokenSize)
	if err != nil {
		return nil, fmt.Errorf("could not generate random registrationToken: %w", err)
	}

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

func DeleteUser(db *sqlx.DB, username string) error {
	username = utils.NormalizeUsername(username)

	user, err := FindUserByUsername(db, username)

	if err != nil {
		return err
	}

	if user == nil {
		return fmt.Errorf("user %s does not exist", username)
	}

	_, err = db.Exec("delete from api_keys where user_id = ?", user.Id)
	if err != nil {
		return err
	}

	_, err = db.Exec("delete from users where id = ?", user.Id)
	if err != nil {
		return err
	}

	return nil
}

func ListUsers(db *sqlx.DB) ([]user.User, error) {
	users := []user.User{}

	err := db.Select(&users, "select * from users")

	if err != nil {
		return nil, err
	}

	return users, nil
}

func CreateApiKey(db *sqlx.DB, user user.User, memo string) (*api_key.ApiKey, error) {
	randomKey, err := generateRandomHex(16)

	if err != nil {
		return nil, fmt.Errorf("could not generate api key: %w", err)
	}

	sqlResult, err := db.Exec(
		"insert into api_keys (user_id, memo, key) values ($1, $2, $3)",
		user.Id,
		memo,
		randomKey,
	)

	if err != nil {
		return nil, fmt.Errorf("error inserting new api key into database: %w", err)
	}

	id, err := sqlResult.LastInsertId()

	if err != nil {
		return nil, fmt.Errorf("error retrieving ID of newly inserted api key: %w", err)
	}

	apiKey := api_key.ApiKey{
		Id:     id,
		UserId: user.Id,
		Memo:   memo,
		Key:    randomKey,
	}

	return &apiKey, nil
}

func ListApiKeys(db *sqlx.DB, user user.User) ([]api_key.ApiKey, error) {
	apiKeys := []api_key.ApiKey{}
	err := db.Select(&apiKeys, "select * from api_keys where user_id = ?", user.Id)

	if err != nil {
		return nil, err
	}

	return apiKeys, nil
}

func DeleteApiKey(db *sqlx.DB, user user.User, id int64) error {
	_, err := db.Exec("delete from api_keys where id = ? and user_id = ?", id, user.Id)
	return err
}

func generateRandomHex(nBytes int) (string, error) {
	randomBytes := make([]byte, nBytes)
	numRead, err := rand.Read(randomBytes)

	if err != nil {
		return "", fmt.Errorf("could not generate random bytes: %w", err)
	}

	if numRead != nBytes {
		return "", fmt.Errorf(
			"tried to generate random %d random bytes, but only %d random bytes were read",
			nBytes,
			numRead,
		)
	}

	return hex.EncodeToString(randomBytes), nil
}
