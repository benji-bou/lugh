package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "lugh",
		Usage: "lugh can be use to construct cyber security pipeline based on modules",
		Action: func(_ *cli.Context) error {
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
