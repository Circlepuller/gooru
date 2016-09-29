package main

import (
	"net/http"
)

func errorHandler(w http.ResponseWriter, status int, err error) {
	switch err.(type) {
	default:
		templates.HTML(w, status, "error", struct {
			Config Config
			StatusCode int
			Error error
			User interface {}
		}{config, status, err, nil})
	}
}
