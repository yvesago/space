package main

import (
    "os"
    "log"

    "github.com/joho/godotenv"
    "github.com/urfave/cli"

    "github.com/earaujoassis/space/tasks"
)

func init() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("The configuration file (.env) doesn't exist; exiting\n")
    }
}

func main() {
    app := cli.NewApp()
    app.Name = "space"
    app.Usage = "A user management microservice; OAuth 2 provider"
    app.EnableBashCompletion = true
    app.Commands = []cli.Command{
        {
            Name:    "serve",
            Aliases: []string{"s"},
            Usage:   "Serve the application server",
            Action:  func(c *cli.Context) error {
                tasks.Server()
                return nil
            },
        },
        {
            Name:    "client",
            Aliases: []string{"c"},
            Usage:   "Manage client application",
            Subcommands: []cli.Command{
                {
                    Name:  "create",
                    Usage: "Create a new client application",
                    Action: func(c *cli.Context) error {
                        tasks.CreateClient()
                        return nil
                    },
                },
            },
        },
    }

    app.Run(os.Args)
}
