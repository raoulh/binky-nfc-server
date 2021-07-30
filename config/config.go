package config

import (
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

//Config is the main app config
var Config = koanf.New(".")

var (
	defaultConfig map[string]interface{} = map[string]interface{}{
		"general.http.port":    48488,
		"general.http.address": "",
		"log.default":          "info",
		"lms.address":          "127.0.0.1",
		"lms.port":             9000,
		"db.sqlite":            "binky.db",
	}

	//a dedicated logger must be used here to avoid conflict
	logging *logrus.Logger
)

//InitConfig from config file, must be called early
func InitConfig(conffile *string) {
	logging = logrus.New()
	logging.Formatter = &logrus.TextFormatter{
		DisableTimestamp: true,
		QuoteEmptyFields: true,
	}
	logging.SetLevel(logrus.TraceLevel)
	log.SetOutput(os.Stdout)

	// Load default values using the confmap provider.
	// We provide a flat map with the "." delimiter.
	// A nested map can be loaded by setting the delimiter to an empty string "".
	Config.Load(confmap.Provider(defaultConfig, "."), nil)

	// Load JSON config.
	logging.Println("Loading config from", *conffile)
	errJson := Config.Load(file.Provider(*conffile), json.Parser())

	// Load TOML config and merge into the previously loaded config (because we can).
	errToml := Config.Load(file.Provider(*conffile), toml.Parser())

	if errJson != nil && errToml != nil {
		logging.WithField("parser", "json").Errorf("error loading config (%s): %v", *conffile, errJson)
		logging.WithField("parser", "toml").Errorf("error loading config (%s): %v", *conffile, errToml)
	}

	// Load environment variables and merge into the loaded config.
	// "BINKY" is the prefix to filter the env vars by.
	// "." is the delimiter used to represent the key hierarchy in env vars.
	// The (optional, or can be nil) function can be used to transform
	// the env var names, for instance, to lowercase them.
	//
	// For example, env vars: BINKY_TYPE and BINKY_PARENT1_CHILD1_NAME
	// will be merged into the "type" and the nested "parent1.child1.name"
	// keys in the config file here as we lowercase the key,
	// replace `_` with `.` and strip the BINKY_ prefix so that
	// only "parent1.child1.name" remains.
	Config.Load(env.Provider("BINKY_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "BINKY_")), "_", ".", -1)
	}), nil)
}
