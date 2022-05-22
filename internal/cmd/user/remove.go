package user

import (
	"authfish/internal/context"
	"authfish/internal/database"
	"authfish/internal/utils"
	"fmt"
	"os"
)

type RemoveCmd struct {
	Username string `arg:""`
}

func (r *RemoveCmd) Run(ctx *context.AppContext) error {
	username := utils.NormalizeUsername(r.Username)
	err := database.DeleteUser(ctx.Db, username)

	if err != nil {
		fmt.Printf("Error deleting user %s: %v\n", username, err)
		os.Exit(1)
	}

	_, err = fmt.Printf("Deleted user %s\n", username)
	return err
}
