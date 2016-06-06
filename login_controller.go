package main

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// GET /register
func registerHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := getSessionID(r)

	if id > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	templates.HTML(w, http.StatusOK, "register", struct {
		Config Config
	}{config})
}

// POST /register
func doRegisterHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := getSessionID(r)

	if id > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()

	// Check confirmations
	if r.Form["email"][0] != r.Form["confirmemail"][0] {
		// TODO
		http.Error(w, "Emails do not match", http.StatusInternalServerError)
		return
	}

	if r.Form["password"][0] != r.Form["confirmpassword"][0] {
		// TODO
		http.Error(w, "Passwords do not match", http.StatusInternalServerError)
		return
	}

	user := User{Email: r.Form["email"][0], Password: r.Form["password"][0], Rank: USER}

	if err := db.Create(&user).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// GET /login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := getSessionID(r)

	if id > 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	templates.HTML(w, http.StatusOK, "login", struct {
		Config Config
	}{config})
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
