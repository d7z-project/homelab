package common

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

var Opts = &Options{
	Bind:         ":8080",
	DB:           "memory://",
	Lock:         "memory://",
	VFS:          "memory://",
	TempDir:      "memory://",
	PubSub:       "memory://",
	RootPassword: "admin",
	TotpAuth:     "",
	JWTSecret:    "change-me-please",
	Workflow:     true,
	Intelligence: true,
}

type Options struct {
	Bind         string `yaml:"bind" env:"HOMELAB_BIND"`
	DB           string `yaml:"db" env:"HOMELAB_DB"`
	Lock         string `yaml:"lock" env:"HOMELAB_LOCK"`
	VFS          string `yaml:"vfs" env:"HOMELAB_VFS"`
	TempDir      string `yaml:"temp_dir" env:"HOMELAB_TEMP_DIR"`
	PubSub       string `yaml:"pub_sub" env:"HOMELAB_PUB_SUB"`
	RootPassword string `yaml:"password" env:"HOMELAB_PASSWORD"`
	TotpAuth     string `yaml:"totp_auth" env:"HOMELAB_TOTP_AUTH"`
	JWTSecret    string `yaml:"jwt_secret" env:"HOMELAB_JWT_SECRET"`
	SessionTTL   string `yaml:"session_ttl" env:"HOMELAB_SESSION_TTL"`
	Workflow     bool   `yaml:"workflow" env:"HOMELAB_WORKFLOW"`
	Intelligence bool   `yaml:"intelligence" env:"HOMELAB_INTELLIGENCE"`
}

func (o *Options) ParseEnv() {
	if o.SessionTTL == "" {
		o.SessionTTL = "30m"
	}
	val := reflect.ValueOf(o).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		envKey := field.Tag.Get("env")
		if envKey == "" {
			continue
		}

		if envVal := os.Getenv(envKey); envVal != "" {
			fieldVal := val.Field(i)
			if fieldVal.Kind() == reflect.String {
				fieldVal.SetString(envVal)
			}
			if fieldVal.Kind() == reflect.Bool {
				if parsed, err := strconv.ParseBool(envVal); err == nil {
					fieldVal.SetBool(parsed)
				}
			}
		}
	}
}

func (o *Options) ParseEnvWithPrefix(prefix string) {
	val := reflect.ValueOf(o).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		envKey := prefix + "_" + strings.ToUpper(strings.ReplaceAll(tag, "-", "_"))
		if envVal := os.Getenv(envKey); envVal != "" {
			fieldVal := val.Field(i)
			if fieldVal.Kind() == reflect.String {
				fieldVal.SetString(envVal)
			}
			if fieldVal.Kind() == reflect.Bool {
				if parsed, err := strconv.ParseBool(envVal); err == nil {
					fieldVal.SetBool(parsed)
				}
			}
		}
	}
}
