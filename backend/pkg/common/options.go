package common

import (
	"os"
	"reflect"
	"strings"
)

var Opts = &Options{
	Bind:         ":8080",
	DB:           "memory://",
	RootPassword: "admin",
	TotpAuth:     "",
}

type Options struct {
	Bind         string `yaml:"bind" env:"HOMELAB_BIND"`
	DB           string `yaml:"db" env:"HOMELAB_DB"`
	RootPassword string `yaml:"password" env:"HOMELAB_PASSWORD"`
	TotpAuth     string `yaml:"totp_auth" env:"HOMELAB_TOTP_AUTH"`
}

func (o *Options) ParseEnv() {
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
		}
	}
}
