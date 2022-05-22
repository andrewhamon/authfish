package main

import (
	"authfish/internal/cmd/server"
	"authfish/internal/cmd/user"
	"authfish/internal/context"
	"authfish/internal/database"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

type CLI struct {
	User         user.UserCmd     `cmd:""`
	Server       server.ServerCmd `cmd:""`
	BaseURL      string
	DatabasePath string `help:"Path to the authfish sqlite database. Default: ~/.authfish/authfish.sqlite"`
}

func main() {
	cliStruct := CLI{}
	cli := kong.Parse(&cliStruct)

	if len(cliStruct.DatabasePath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}

		cliStruct.DatabasePath = filepath.Join(homeDir, ".authfish", "authfish.sqlite")
	}

	databaseDir := filepath.Dir(cliStruct.DatabasePath)

	err := os.MkdirAll(databaseDir, os.ModePerm)

	if err != nil {
		fmt.Printf("Error creating database: %v\n", err)
		os.Exit(1)
	}

	db := database.OpenDB(cliStruct.DatabasePath)
	database.RunMigrations(db)

	appContext := context.AppContext{
		Db: db,
	}

	if len(cliStruct.BaseURL) == 0 {
		cliStruct.BaseURL = "/"
	}

	if baseUrl, err := url.ParseRequestURI(cliStruct.BaseURL); err == nil {
		appContext.BaseUrl = baseUrl
	} else {
		// fmt.Printf("Invalid option for --base-url: %v", err)
		os.Exit(1)
	}

	err = cli.Run(&appContext)

	if err != nil {
		panic(err)
	}
}
