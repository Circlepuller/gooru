package main

import (
	"net/http"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/i18n"
)

type SessionError error
type SessionLangError SessionError
type SessionUserError SessionError

func loadLangs(path string) error {
	files, err := filepath.Glob(filepath.Join(path, "*.all.json"))

	if err != nil {
		return err
	}

	for _, f := range files {
		err = i18n.LoadTranslationFile(f)

		if err != nil {
			return err
		}
	}

	return nil
}

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

func setSessionUser(w http.ResponseWriter, r *http.Request, u User) SessionUserError {
	return setSessionID(w, r, u.ID)
}

func getSessionUser(r *http.Request) (u User, err SessionUserError) {
	if id, err := getSessionID(r); err == nil {
		err = db.Where("id = ?", id).First(&u).Error
	}

	return
}

func clearSessionUser(w http.ResponseWriter, r *http.Request) SessionUserError {
	return clearSessionID(w, r)
}

func getSession(r *http.Request) (i18n.TranslateFunc, User, SessionError) {
	T, _ := i18n.Tfunc(r.Header.Get("Accept-Language"), "en-US")
	u, err := getSessionUser(r)

	return T, u, err
}
