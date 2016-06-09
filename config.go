package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	// debug mode
	Debug bool
	// database address
	Dsn string
	// secret salt (for sessions)
	Secret string
	// give the website a personal name
	SiteName string
	// posts to allocate per page
	PostsPerPage uint64
	// allow duplicate files
	AllowDuplicates bool
	// image settings
	ThumbX uint
	ThumbY uint
}

func ReadConfig(name string) Config {
	file, _ := os.Open(name)
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)

	if err != nil {
		panic(err.Error())
	}

	return config
}
