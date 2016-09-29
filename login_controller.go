package main

import (
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// GET /register
func registerHandler(w http.ResponseWriter, r *http.Request) {
	_, user, err := getSession(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	templates.HTML(w, http.StatusOK, "register", struct {
		Config Config
		User User
	}{config, user})
}

// POST /register
func doRegisterHandler(w http.ResponseWriter, r *http.Request) {
	T, user, _ := getSession(r)

	if user.ID > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()

	// Check confirmations
	if r.Form["email"][0] != r.Form["confirmemail"][0] {
		// TODO
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_email_mismatch")))
		return
	}

	if r.Form["password"][0] != r.Form["confirmpassword"][0] {
		// TODO
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_password_mismatch")))
		return
	}

	newUser := User{Email: r.Form["email"][0], Password: r.Form["password"][0], Rank: USER}

	if err := db.Create(&newUser).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// GET /login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	templates.HTML(w, http.StatusOK, "login", struct {
		Config Config
		User User
	}{config, user})
}

// POST /login
func doLoginHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	id, _ := getSessionID(r)

	if id > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()

	// pull the user from the database
	if err := db.Where("email = ?", r.Form["email"][0]).First(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// see if passwords match
	if err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(r.Form["password"][0])); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setSessionID(w, r, user.ID)
	http.Redirect(w, r, "/", http.StatusFound)
}

// GET /logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSessionID(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
}
