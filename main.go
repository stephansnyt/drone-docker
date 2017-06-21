package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var rev string = "[unknown]"

func main() {
	// Load env-file if it exists first
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	app := cli.NewApp()
	app.Name = "gdm plugin"
	app.Usage = "gdm plugin"
	app.Action = run
	app.Version = fmt.Sprintf("1.0.0-%s", rev)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "action",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_ACTION",
			Value:  "update",
		},
		cli.BoolFlag{
			Name:   "async",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_ASYNC",
		},
		cli.StringFlag{
			Name:   "config-template",
			Usage:  "template for deployment configuration",
			EnvVar: "PLUGIN_CONFIG_TEMPLATE",
			Value:  ".gdm.yml",
		},
		cli.StringFlag{
			Name:   "create-policy",
			Usage:  "gcloud pass-through ",
			EnvVar: "PLUGIN_CREATE_POLICY",
		},
		cli.StringFlag{
			Name:   "delete-policy",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_DELETE_POLICY",
		},
		cli.StringFlag{
			Name:   "deployment",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_DEPLOYMENT",
		},
		cli.StringFlag{
			Name:   "description",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_DESCRIPTION",
		},
		cli.BoolFlag{
			Name:   "dry-run",
			Usage:  "If set, skips the final gcloud deployment manager command",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.StringFlag{
			Name:   "gcloud-cmd",
			Usage:  "alternative gcloud cmd path, useful for local testing",
			EnvVar: "PLUGIN_GCLOUD_CMD",
		},
		cli.StringFlag{
			Name:   "output-file",
			Usage:  "interpolated template output file path",
			EnvVar: "PLUGIN_OUTPUT_FILE",
			Value:  ".drone-gdm.yml",
		},
		cli.BoolFlag{
			Name:   "preview",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_PREVIEW",
		},
		cli.StringFlag{
			Name:   "project",
			Usage:  "gcloud pass-through",
			EnvVar: "PLUGIN_PROJECT",
		},
		cli.StringFlag{
			Name:   "vars",
			Usage:  "variables to use in the config template",
			EnvVar: "PLUGIN_VARS",
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "verbose output including the interpolated template",
			EnvVar: "PLUGIN_VERBOSE",
		},
		cli.StringFlag{
			Name:   "token",
			Usage:  "service account JSON",
			EnvVar: "TOKEN",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// this is the one bit of pre-processing done before calling plugin.Exec
	// just to get Vars to be the right type
	varsJson := c.String("vars")
	vars := make(map[string]interface{})
	if varsJson != "" {
		err := json.Unmarshal([]byte(varsJson), &vars)
		if err != nil {
			return fmt.Errorf("Failure to unmarshal vars %s", err)
		}
	}

	plugin := Plugin{
		Action:         c.String("action"),
		Async:          c.Bool("async"),
		ConfigTemplate: c.String("config-template"),
		CreatePolicy:   c.String("create-policy"),
		DeletePolicy:   c.String("delete-policy"),
		Dryrun:         c.Bool("dry-run"),
		Deployment:     c.String("deployment"),
		Description:    c.String("description"),
		GcloudCmd:      c.String("gcloud-cmd"),
		OutputFile:     c.String("output-file"),
		Preview:        c.Bool("preview"),
		Project:        c.String("project"),
		Token:          c.String("token"),
		Vars:           vars,
		Verbose:        c.Bool("verbose"),
	}

	return plugin.Exec()
}
