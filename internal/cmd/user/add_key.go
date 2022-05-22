package user

import (
	"authfish/internal/context"
	"authfish/internal/database"
	"authfish/internal/utils"
	"fmt"
	"os"
)

type AddKeyCmd struct {
	Username string `arg:""`
	Memo     string `arg:""`
}

func (r *AddKeyCmd) Run(ctx *context.AppContext) error {
	user, err := database.FindUserByUsername(ctx.Db, utils.NormalizeUsername(r.Username))

	if err != nil {
		fmt.Printf("Error adding api key: %v\n", err)
		os.Exit(1)
	}

	if user == nil {
		fmt.Printf("User does not exist: %s\n", r.Username)
		os.Exit(1)
	}

	apiKey, err := database.CreateApiKey(ctx.Db, *user, r.Memo)

	if err != nil {
		fmt.Printf("Error adding api key: %v\n", err)
		os.Exit(1)
	}

	if apiKey == nil {
		fmt.Printf("api key does not exist\n")
		os.Exit(1)
	}

	_, err = fmt.Printf("Key: %s", apiKey.Key)

	return err
}
