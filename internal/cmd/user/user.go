package user

import (
	"net/url"
	"path"
)

type UserCmd struct {
	List   ListCmd   `cmd:"" default:""`
	Add    AddCmd    `cmd:"" aliases:"create,register"`
	AddKey AddKeyCmd `cmd:""`
	Remove RemoveCmd `cmd:"" aliases:"rm,del,delete"`
}

func buildRegistrationURL(base *url.URL, token *string) string {
	if token == nil {
		return "<nil>"
	} else {
		newPath := path.Join(base.Path, "register")
		dupUrl := *base
		dupUrl.Path = newPath

		queryParams := dupUrl.Query()
		queryParams.Set("registrationToken", *token)

		dupUrl.RawQuery = queryParams.Encode()

		return dupUrl.String()
	}
}
