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
		// honestly the next 5 lines are the worst code ever
		if session.Values["id"] == nil {
			session.Values["id"] = uint(0)
		}

		id = session.Values["id"].(uint)
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
