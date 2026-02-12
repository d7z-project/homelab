package common

var Opts = &Options{
	Bind:         ":8080",
	DB:           "memory://",
	RootPassword: "admin",
}

type Options struct {
	Bind         string `yaml:"bind"`
	DB           string `yaml:"db"`
	RootPassword string `yaml:"password"`
}
