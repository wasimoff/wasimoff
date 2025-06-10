package config

import (
	"errors"
	"log"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Keep a global lazy-loaded instance of the configuration
var configuration *Configuration

// GetConfiguration returns a lazily-loaded configuration parsed from environment.
func GetConfiguration() (conf Configuration) {
	if configuration == nil {
		configuration = loadConfiguration()
	}
	return *configuration
}

// loadConfiguration checks if user requested help (-h/--help) and prints usage information
// or returns the configuration parsed from environment variables.
// TODO: replace with spf13/viper if file handling is needed
func loadConfiguration() (conf *Configuration) {
	conf = &Configuration{}

	// print help if requested on commandline
	if len(os.Args) >= 2 && slices.ContainsFunc(os.Args[1:], func(arg string) bool {
		return arg == "-h" || arg == "--help"
	}) {
		tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
		envconfig.Usagef(envprefix, conf, tabs, usageHelpFormat)
		tabs.Flush()
		os.Exit(1)
	}

	// load .env file into environment
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// ignore simple "not found" errors
		log.Fatalf("failed to load dotenv: %s", err)
	}

	// parse configuration from environment variables
	if err := envconfig.Process(envprefix, conf); err != nil {
		log.Fatalf("failed parsing config: %s", err)
	}
	return
}

// see https://github.com/kelseyhightower/envconfig/blob/v1.4.0/usage.go#L31
const usageHelpFormat = `This application is configured with the following environment variables:
KEY	DESCRIPTION	DEFAULT
{{range .}}{{usage_key .}}	{{usage_description .}}	{{usage_default .}}
{{end}}`
