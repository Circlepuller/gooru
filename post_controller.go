package main

import (
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type Pagination struct {
	CurrentPage uint64
	TotalPages uint64
	Pages []uint64
}

// GET /list/<page>
// GET /list/<tags>/<page>
func listHandler(w http.ResponseWriter, r *http.Request) {
	var (
		posts []Post
		tags []string
		count uint64
		pages []uint64
		err error
	)
	id, _ := getSessionID(r)
	params := mux.Vars(r)
	query, ok := params["query"]

	if ok {
		// We have a specific set of tags
		if query, err = url.QueryUnescape(query); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tags = strings.Fields(query)
		count, err = CountPostsByTagNames(tags)
	} else {
		// We want ALL the posts!
		count, err = CountPosts()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count == 0 {
		// 404 much
		http.Error(w, "No posts found", http.StatusInternalServerError)
		return
	}

	totalPages := uint64(math.Ceil(float64(count) / float64(config.PostsPerPage)))
	currentPage, err := strconv.ParseUint(params["page"], 10, 64)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if currentPage < 1 {
		currentPage = 1
	} else if currentPage > totalPages {
		currentPage = totalPages
	}

	for page := uint64(1); page <= totalPages; page++ {
		if page == currentPage || (page >= currentPage - 5 && page <= currentPage + 5) {
			pages = append(pages, page)
		}
	}

	if ok {
		posts, err = GetPostsByTagNames(currentPage, tags)
	} else {
		posts, err = GetPosts(currentPage)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		templates.HTML(w, http.StatusOK, "list", struct {
			Config Config
			ID uint
			Posts []Post
			Pagination Pagination
		}{config, id, posts, Pagination{currentPage, totalPages, pages}})
	}
}

// GET /list
// POST /list
func doListHandler(w http.ResponseWriter, r *http.Request) {
	if tags := r.FormValue("tags"); r.Method == "GET" || tags == "" {
		http.Redirect(w, r, "/list/1", http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/list/%s/1", url.QueryEscape(tags)), http.StatusFound)
	}
}


// GET /
func indexHandler(w http.ResponseWriter, r *http.Request) {
	// temporary redirect, we'll probably have a 4chan-style index
	http.Redirect(w, r, "/list/1", http.StatusFound)
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

	db.Preload("Replies").Preload("Replies.File").Preload("Replies.User").Preload("File").Preload("Tags").Preload("User").Where("parent_id = ?", 0).First(&post, parentID)

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
	post.Message = r.MultipartForm.Value["message"][0]

	/* Don't allow subjects/tags for replies,
	 * it's useless and takes up space on the database
	 */
	if post.ParentID == 0 {
		post.Subject = r.MultipartForm.Value["subject"][0]
		post.Tags = TagsFromString(r.MultipartForm.Value["tags"][0])

		if len(post.Tags) < 1 {
			http.Error(w, "Must have one tag", http.StatusInternalServerError)
			return
		}
	}

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

// POST /post
func doThreadHandler(w http.ResponseWriter, r *http.Request) {
	doPost(w, r, 0)
}

// POST /post/<parentID>
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
