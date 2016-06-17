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

func setSessionUser(w http.ResponseWriter, r *http.Request, u User) error {
	return setSessionID(w, r, u.ID)
}

func getSessionUser(r *http.Request) (u User, err error) {
	if id, err := getSessionID(r); err == nil {
		err = db.Where("id = ?", id).First(&u).Error
	}

	return
}

func clearSessionUser(w http.ResponseWriter, r *http.Request) error {
	return clearSessionID(w, r)
}
