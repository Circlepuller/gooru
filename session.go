package main

import (
	"net/http"
)

func setSessionID(w http.ResponseWriter, r *http.Request, id uint) (err error) {
	if session, err := store.Get(r, "session"); err == nil {
		session.Values["id"] = id
		session.Save(r, w)
	}

	return err
}

func getSessionID(r *http.Request) (id uint, err error) {
	if session, err := store.Get(r, "session"); err == nil {
		switch session.Values["id"].(type) {
		case uint:
			id = session.Values["id"].(uint)
		default:
			session.Values["id"] = uint(0)
			id = session.Values["id"].(uint)
		}
	}

	return id, err
}

func clearSessionID(w http.ResponseWriter, r *http.Request) (err error) {
	if session, err := store.Get(r, "session"); err == nil {
		session.Values["id"] = uint(0)
		session.Save(r, w)
	}

	return err
}
