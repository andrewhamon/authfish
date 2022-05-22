package user

import (
	"authfish/internal/context"
	"authfish/internal/database"
	"authfish/internal/utils"
	"fmt"
	"os"
)

type AddCmd struct {
	Username string `arg:""`
}

func (r *AddCmd) Run(ctx *context.AppContext) error {
	username := utils.NormalizeUsername(r.Username)
	user, err := database.RegisterNewUser(ctx.Db, username)

	if err != nil {
		fmt.Printf("Error adding new user: %v\n", err)
		os.Exit(1)
	}

	registrationUrl := buildRegistrationURL(ctx.BaseUrl, user.RegistrationToken)

	_, err = fmt.Println(registrationUrl)
	return err
}
