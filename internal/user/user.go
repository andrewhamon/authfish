package user

import "time"

type CurrentUserContext struct{}

var CurrentUserContextKey CurrentUserContext = CurrentUserContext{}

type User struct {
	Id                int64     `db:"id"`
	Username          string    `db:"username"`
	HashedPassword    []byte    `db:"hashed_password"`
	RegistrationToken *string   `db:"registration_token"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
