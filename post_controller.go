package main

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// GET /
func indexHandler(w http.ResponseWriter, r *http.Request) {
	var posts []Post
	id, _ := getSessionID(r)

	if err := db.Preload("Replies").Preload("Replies.File").Preload("Replies.User").Preload("File").Preload("User").Where("parent_id = 0").Find(&posts).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templates.HTML(w, http.StatusOK, "index", struct {
		Config Config
		ID uint
		Posts []Post
	}{config, id, posts})

	/*if id > 0 {
		fmt.Fprintf(w, "<!DOCTYPE html><form action=\"/\" enctype=\"multipart/form-data\" method=\"POST\"><input type=\"text\" name=\"name\" placeholder=\"Name\"><textarea name=\"message\"></textarea><input type=\"file\" name=\"file\"><input type=\"file\" name=\"file\"><input type=\"submit\" name=\"submit\"></form><br>%s", templates.DefinedTemplates())
	} else {
		fmt.Fprintf(w, "<!DOCTYPE html>You should probably <a href=\"/login\">log in</a> or <a href=\"/register\">register</a>.")
	}*/
}

// GET /thread/<parentID>
func threadHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	id, _ := getSessionID(r)
	parentID, err := strconv.ParseUint(params["parentID"], 36, 64)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.Preload("Replies").Preload("Replies.File").Preload("Replies.User").Preload("File").Preload("User").Where("parent_id = ?", 0).First(&post, parentID)

	if post.ID == 0 {
		http.Error(w, "Post not found", http.StatusInternalServerError)
	} else {
		templates.HTML(w, http.StatusOK, "thread", struct {
			Config Config
			ID uint
			Post Post
		}{config, id, post})
	}
}

func doTagsHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("/tagged/%s", url.QueryEscape(r.FormValue("tags"))), http.StatusFound)
}

func tagsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, _ := getSessionID(r)
	query, err := url.QueryUnescape(params["query"])

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts, err := PostsByTagNames(strings.Fields(query))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		templates.HTML(w, http.StatusOK, "index", struct {
			Config Config
			ID uint
			Posts []Post
		}{config, id, posts})
	}
}

// POST /post
func doPost(w http.ResponseWriter, r *http.Request, parentID uint) {
	var post Post
	id, _ := getSessionID(r)

	if id == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := r.ParseMultipartForm(512); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	post.ParentID = parentID
	post.UserID = id
	post.ParseName(r.MultipartForm.Value["name"][0])
	post.Subject = r.MultipartForm.Value["subject"][0]
	post.Message = r.MultipartForm.Value["message"][0]
	post.Tags = TagsFromString(r.MultipartForm.Value["tags"][0])

	if len(r.MultipartForm.File["file"]) > 0 {
		file, err := func(h *multipart.FileHeader) (file File, err error) {
			f, err := h.Open()

			if err != nil {
				return
			}

			defer f.Close()

			fileHandler, err := fileRouter.File(w, f)

			if err != nil {
				return
			}

			file, err = fileHandler(w, r, f, h)

			return
		}(r.MultipartForm.File["file"][0])

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		post.File = file
	}

	if post.ParentID == 0 && post.File.Name == "" {
		http.Error(w, "You forgot a file", http.StatusInternalServerError)
		return
	}

	if err := db.Create(&post).Error; err != nil {
		if post.File.File != "" {
			post.File.DeleteUpload()
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if post.ParentID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/thread/%s", strconv.FormatUint(uint64(post.ID), 36)), http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/thread/%s", strconv.FormatUint(uint64(post.ParentID), 36)), http.StatusFound)
	}
}

func doThreadHandler(w http.ResponseWriter, r *http.Request) {
	doPost(w, r, 0)
}

func doReplyHandler(w http.ResponseWriter, r *http.Request) {
	var count uint
	params := mux.Vars(r)
	parentID, err := strconv.ParseUint(params["parentID"], 36, 64)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.Model(&Post{}).Where("id = ?", parentID).Where("parent_id = ?", 0).Count(&count)

	if count != 0 {
		doPost(w, r, uint(parentID))
	} else {
		// 404
		http.Error(w, "Post not found", http.StatusNotFound)
	}
}
