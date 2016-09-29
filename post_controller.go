package main

import (
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	params := mux.Vars(r)
	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	// If there were posts found, get them
	if ok && count > 0 {
		posts, err = GetPostsByTagNames(currentPage, tags)
	} else if count > 0 {
		posts, err = GetPosts(currentPage)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		templates.HTML(w, http.StatusOK, "list", struct {
			Config Config
			User User
			Posts []Post
			Pagination Pagination
		}{config, user, posts, Pagination{currentPage, totalPages, pages}})
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

// GET /post/<parentID>
func postHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	parentID, err := strconv.ParseUint(params["parentID"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	GetPostPreloads().Where("parent_id = ?", 0).First(&post, parentID)

	if post.ID == 0 {
		http.Error(w, "Post not found", http.StatusInternalServerError)
	} else {
		templates.HTML(w, http.StatusOK, "post", struct {
			Config Config
			User User
			Post Post
		}{config, user, post})
	}
}

// GET /post/<id>/edit
func postEditHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	id, err := strconv.ParseUint(params["id"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "You need to be logged in", http.StatusInternalServerError)
		return
	}

	if err = db.Preload("Tags").Preload("User").Where("id = ?", id).First(&post).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.Rank < MOD && user.ID != post.User.ID {
		http.Error(w, "You aren't allowed to edit this", http.StatusInternalServerError)
		return
	}

	templates.HTML(w, http.StatusOK, "post-edit", struct {
		Config Config
		User User
		Post Post
	}{config, user, post})
}

// POST /post/<id>/edit
func doPostEditHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	var tags []Tag
	params := mux.Vars(r)
	id, err := strconv.ParseUint(params["id"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "You need to be logged in", http.StatusInternalServerError)
		return
	}

	if err = db.Preload("Tags").Preload("User").Where("id = ?", id).First(&post).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.Rank < MOD && user.ID != post.User.ID {
		http.Error(w, "You aren't allow to edit this", http.StatusInternalServerError)
		return
	}

	if post.ParentID == 0 {
		for _, n := range strings.Fields(r.FormValue("tags")) {
			var tag Tag

			if err = db.FirstOrInit(&tag, Tag{Name: n}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			tags = append(tags, tag)
		}

		if len(tags) < 1 {
			http.Error(w, "Must have at least one tag", http.StatusInternalServerError)
			return
		}

		if err = db.Model(&post).Association("Tags").Replace(tags).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	post.Message = r.FormValue("message")
	post.Edited = time.Now()

	if err = db.Save(&post).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if post.ParentID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ID), 36)), http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ParentID), 36)), http.StatusFound)
	}
}

func postBanHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	id, err := strconv.ParseUint(params["id"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	T, user, err := getSession(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_login_required")))
		return
	}

	if err = db.Where("id = ?", id).First(&post).Error; err != nil {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_database_error", map[string]interface {}{
			"Error": err.Error()}
		)))
		return
	} else if user.Rank < MOD {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_low_ranking")))
		return
	}

	templates.HTML(w, http.StatusOK, "post-ban", struct {
		Config Config
		Post Post
		User User
	}{config, post, user})
}

func doPostBanHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	id, err := strconv.ParseUint(params["id"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	T, user, err := getSession(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_login_required")))
		return
	}

	if err = db.Preload("User").Where("id = ?", id).First(&post).Error; err != nil {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_database_error", map[string]interface {}{
			"Error": err.Error()
		})))
		return
	} else if user.Rank < MOD {
		errorHandler(w, http.StatusInternalServerError, errors.New(T("err_low_ranking")))
		return
	}

	expires, err := time.ParseInLocation("2006-01-02T15:04", r.FormValue("expires"), time.Local)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ban := Ban{
		Creator: user,
		User: post.User,
		Post: post,
		Expires: expires,
		Reason: r.FormValue("reason"),
	}

	if err = db.Create(&ban).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if post.ParentID == 0 {
			http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ID), 36)), http.StatusFound)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ParentID), 36)), http.StatusFound)
		}
	}
}

func postDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var post Post
	params := mux.Vars(r)
	id, err := strconv.ParseUint(params["id"], 36, 64)

	if err != nil {
		// invalid id
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "You need to be logged in", http.StatusInternalServerError)
		return
	}

	if err = db.Preload("Tags").Preload("User").Where("id = ?", id).First(&post).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.Rank < JANITOR && user.ID != post.User.ID {
		http.Error(w, "You aren't allowed to delete this post", http.StatusInternalServerError)
		return
	} else if err = db.Delete(&post).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if post.ParentID != 0 {
		http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ParentID), 36)), http.StatusFound)
	} else {
		http.Redirect(w, r, "/list", http.StatusFound)
	}
}

// Posting "middleware"
func doPost(w http.ResponseWriter, r *http.Request, parentID uint) {
	var post Post
	user, err := getSessionUser(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "You need to be logged in", http.StatusInternalServerError)
		return
	}

	ban, err := user.CheckBans()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if ban != nil {
		templates.HTML(w, http.StatusForbidden, "banned", struct {
			Config Config
			User User
			Ban Ban
		}{config, user, *ban})
		return
	}

	if err := r.ParseMultipartForm(512); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	post.ParentID = parentID
	post.UserID = user.ID
	post.ParseName(r.FormValue("name"))
	post.Message = r.FormValue("message")

	/* Don't allow subjects/tags for replies,
	 * it's useless and takes up space on the database
	 */
	if post.ParentID == 0 {
		post.Subject = r.FormValue("subject")

		// you guys have no idea how frustrated i was making this - circle
		for _, n := range strings.Fields(r.FormValue("tags")) {
			var tag Tag

			if err := db.FirstOrInit(&tag, Tag{Name: n}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			post.Tags = append(post.Tags, tag)
		}

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
		// Note: this needs improvement..
		if post.File.File != "" {
			if err = post.File.BeforeDelete(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if post.ParentID == 0 {
		http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ID), 36)), http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/post/%s", strconv.FormatUint(uint64(post.ParentID), 36)), http.StatusFound)
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
