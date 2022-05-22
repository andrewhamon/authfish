package user

import (
	"authfish/internal/context"
	"authfish/internal/database"
	"fmt"

	"github.com/gosuri/uitable"
)

type ListCmd struct {
}

func (r *ListCmd) Run(ctx *context.AppContext) error {
	users, err := database.ListUsers(ctx.Db)
	if err != nil {
		return err
	}

	table := uitable.New()

	table.AddRow("Id", "Username", "Registration URL", "Created At", "Updated At")

	for _, user := range users {
		table.AddRow(user.Id, user.Username, buildRegistrationURL(ctx.BaseUrl, user.RegistrationToken), user.CreatedAt, user.UpdatedAt)
	}

	_, err = fmt.Println(table)

	return err
}
