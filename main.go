package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/unrolled/render"
)

var (
	db *gorm.DB
	config Config
	fileRouter = NewFileRouter()
	router = mux.NewRouter()
	store *sessions.CookieStore
	templates *render.Render
)

func main() {
	var (
		err error
		email string
		password string
	)

	// allowing different config files allows for configurations in different environments
	configName := flag.String("config", "config.json", "specify a configuration file (default: config.json)")

	flag.Parse()

	if flag.NArg() != 1 {
		panic("no command specified")
	}

	templateFuncs := template.FuncMap{
		"title": strings.Title,
		"id2url": func(i uint) string { return strconv.FormatUint(uint64(i), 36) },
		"formatTime": func(t time.Time, layout string) string { return t.Format(layout) },

		// dict function (http://stackoverflow.com/a/18276968)
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values) % 2 != 0 {
				return nil, errors.New("invalid dict call")
			}

			dict := make(map[string]interface{}, len(values) / 2)

			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)

				if !ok {
					return nil, errors.New("dict keys must be strings")
				}

				dict[key] = values[i + 1]
			}

			return dict, nil
		},
	}
	config = ReadConfig(*configName)
	templates = render.New(render.Options{
		Funcs: []template.FuncMap{templateFuncs},
		Layout: "layout",
		IsDevelopment: config.Debug,
	})
	store = sessions.NewCookieStore([]byte(config.Secret))

	if db, err = gorm.Open("sqlite3", config.Dsn); err != nil {
		panic(err.Error())
	}

	defer db.Close()

	if err = db.DB().Ping(); err != nil {
		panic(err.Error())
	}

	db.LogMode(config.Debug)
	db.AutoMigrate(&Ban{}, &File{}, &Post{}, &Tag{}, &User{})

	cmd := flag.Arg(0)

	switch cmd {
	case "init":
		// create a default administrator
		fmt.Print("creating admin...\nemail: ")
		fmt.Scanln(&email)
		fmt.Print("password: ")
		fmt.Scanln(&password)

		user := User{Email: email, Password: password, Rank: ADMIN}

		if err := db.Create(&user).Error; err != nil {
			panic(err.Error())
		}

		fmt.Printf("created admin '%s' with password hash '%s'\n", user.Email, string(user.HashedPassword))
	case "run":
		// Add file types
		fileRouter.AddFile("image/jpeg", jpegFile)
		fileRouter.AddFile("image/png", pngFile)

		// post_controller.go
		router.HandleFunc("/", indexHandler).Methods("GET")

		router.HandleFunc("/list", doListHandler).Methods("GET", "POST")
		router.HandleFunc("/list/{page:[0-9]+}", listHandler).Methods("GET")
		router.HandleFunc("/list/{query:.+}/{page:[0-9]+}", listHandler).Methods("GET")

		router.HandleFunc("/post", doThreadHandler).Methods("POST")
		router.HandleFunc("/post/{parentID:[0-9a-z]+}", postHandler).Methods("GET")
		router.HandleFunc("/post/{parentID:[0-9a-z]+}", doReplyHandler).Methods("POST")
		router.HandleFunc("/post/{id:[0-9a-z]+}/edit", postEditHandler).Methods("GET")
		router.HandleFunc("/post/{id:[0-9a-z]+}/edit", doPostEditHandler).Methods("POST")
		router.HandleFunc("/post/{id:[0-9a-z]+}/ban", postBanHandler).Methods("GET")
		router.HandleFunc("/post/{id:[0-9a-z]+}/ban", doPostBanHandler).Methods("POST")
		router.HandleFunc("/post/{id:[0-9a-z]+}/delete", postDeleteHandler).Methods("GET")

		// login_controller.go
		router.HandleFunc("/register", registerHandler).Methods("GET")
		router.HandleFunc("/register", doRegisterHandler).Methods("POST")
		router.HandleFunc("/login", loginHandler).Methods("GET")
		router.HandleFunc("/login", doLoginHandler).Methods("POST")
		router.HandleFunc("/logout", logoutHandler).Methods("GET")

		router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public")))
		http.Handle("/", router)
		http.ListenAndServe(":8080", nil)
	}
}
