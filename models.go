package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	//"regexp"
	"strings"
	"time"

	"github.com/aquilax/tripcode"
	"github.com/asaskevich/govalidator"
	"github.com/jinzhu/gorm"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/bcrypt"
)

const (
	USER = iota
	JANITOR
	MOD
	ADMIN
)

type Tag struct {
	gorm.Model
	Name string `gorm:"not_null;unique"`
}

/*func (t *Tag) BeforeSave() error {
	return t.Sanitize()
}

func (t *Tag) Sanitize() error {
	r, err := regexp.Compile(`[^A-Za-z0-9_\'\/\-\(\)`)

	if err != nil {
		return err
	}

	t.Name = string(r.ReplaceAll([]byte(t.Name), []byte("")))
	return nil
}*/

type File struct {
	gorm.Model
	PostID uint
	File string `gorm:"unique"`
	Hash string
	Width int
	Height int
	Size int
	Thumb string `gorm:"unique"`
	ThumbWidth int
	ThumbHeight int
	Name string
	Type string `gorm:"not_null"`
}

func (f *File) CreateHash(file multipart.File) {
	hash := sha256.New()
	file.Seek(0, 0)
	io.Copy(hash, file)
	file.Seek(0, 0)
	f.Hash = hex.EncodeToString(hash.Sum(nil))
}

func (f *File) UploadFile(file multipart.File) error {
	file.Seek(0, 0)

	if upload, err := os.Create(filepath.Join("public", "src", f.File)); err == nil {
		defer upload.Close()
		io.Copy(upload, file)
	} else {
		return err
	}

	file.Seek(0, 0)

	return nil
}

func (f *File) UploadThumbnail(i image.Image) (o image.Config, err error) {
	thumb := resize.Thumbnail(config.ThumbX, config.ThumbY, i, resize.Lanczos3)
	upload, err := os.Create(filepath.Join("public", "src", f.Thumb))

	if err != nil {
		return
	}

	defer upload.Close()

	if err = jpeg.Encode(upload, thumb, nil); err != nil {
		return
	}

	upload.Seek(0, 0)

	o, err = jpeg.DecodeConfig(upload)

	return
}

func (f *File) BeforeDelete() (err error) {
	if err = os.Remove(filepath.Join("public", "src", f.File)); err != nil {
		return
	}

	if f.Thumb != "" {
		err = os.Remove(filepath.Join("public", "src", f.Thumb))
	}

	return
}

type Post struct {
	gorm.Model
	User User `gorm:"ForeignKey:UserID"`
	UserID uint
	Edited time.Time
	Name string `gorm:"not_null;default:'Anonymous'"`
	Tripcode string
	Subject string
	Message string
	Tags []Tag `gorm:"many2many:post_tags"`
	Replies []Post `gorm:"ForeignKey:ParentID"`
	ParentID uint
	File File `gorm:"ForeignKey:PostID"`
	Stickied bool `gorm:"not_null;default:false"`
	NSFW bool `gorm:"not_null;default:false"`
}

func (p *Post) BeforeDelete() (err error) {
	var posts []Post
	var files []File

	// Delete replies
	if err = db.Where("parent_id = ?", p.ID).Find(&posts).Error; err != nil {
		return
	}

	for _, post := range posts {
		// See comment about callbacks below..
		if err = post.BeforeDelete(); err != nil {
			return
		} else if err = db.Delete(post).Error; err != nil {
			return
		}
	}

	// Delete files
	if err = db.Where("post_id = ?", p.ID).Find(&files).Error; err != nil {
		return
	}

	for _, file := range files {
		// Apparently GORM won't execute callbacks if you're already in a callback..
		if err = file.BeforeDelete(); err != nil {
			return
		} else if err = db.Delete(file).Error; err != nil {
			return
		}
	}

	return
}

func (p *Post) ParseName(input string) {
	params := strings.SplitN(input, "#", 2)
	p.Name = strings.Trim(params[0], " ")

	if len(params) > 1 && len(params[1]) > 0 {
		if len(params[1]) > 1 && strings.HasPrefix(params[1], "#") {
			p.Tripcode = "!!"+tripcode.SecureTripcode(params[1][1:], config.Secret)
		} else {
			p.Tripcode = "!"+tripcode.Tripcode(params[1])
		}
	}
}

func CountPosts() (count uint64, err error) {
	err = db.Model(&Post{}).Where("parent_id = 0").Count(&count).Error
	return
}

func CountPostsByTagNames(tags []string) (count uint64, err error) {
	var counts []uint64

	// I blame the GORM for allowing this horrible mess..
	err = db.Model(&Post{}).Joins("JOIN post_tags ON posts.id = post_tags.post_id").Joins("JOIN tags ON post_tags.tag_id = tags.id").Where("tags.name IN (?)", tags).Group("posts.id").Having(fmt.Sprintf("COUNT(*) = %d", len(tags))).Pluck("COUNT(DISTINCT posts.id)", &counts).Error
	count = uint64(len(counts))

	return
}

func GetPostPreloads() *gorm.DB {
	return db.Preload("Replies").Preload("Replies.File").Preload("Replies.User").Preload("File").Preload("Tags").Preload("User")
}

func GetPosts(page uint64) (posts []Post, err error) {
	err = GetPostPreloads().Where("parent_id = 0").Offset(int((page - 1) * config.PostsPerPage)).Limit(int(config.PostsPerPage)).Order("stickied, updated_at DESC").Find(&posts).Error
	return
}

func GetPostsByTagNames(page uint64, tags []string) (posts []Post, err error) {
	// arguably not as bad as the count for this function
	err = GetPostPreloads().Model(&Post{}).Select("DISTINCT posts.*").Joins("JOIN post_tags ON posts.id = post_tags.post_id").Joins("JOIN tags ON post_tags.tag_id = tags.id").Where("tags.name IN (?)", tags).Group("posts.id").Having(fmt.Sprintf("COUNT(*) = %d", len(tags))).Offset(int((page - 1) * config.PostsPerPage)).Limit(int(config.PostsPerPage)).Order("stickied, updated_at DESC").Find(&posts).Error
	return
}

type User struct {
	gorm.Model
	Email string `gorm:"not_null;unique" valid:"email"`
	Password string `sql:"-" valid:"len(8|)"`
	HashedPassword []byte `gorm:"not_null"`
	Rank uint `gorm:"not_null;default:0"`
	Posts []Post `gorm:"ForeignKey:UserID"`
}

func (u *User) BeforeCreate() (err error) {
	if _, err = govalidator.ValidateStruct(u); err != nil {
		return
	}

	u.HashedPassword, err = bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)

	return
}

func (u *User) CheckBans() (ban *Ban, err error) {
	var bans []Ban

	if err = db.Preload("Post").Where("user_id = ?", u.ID).Find(&bans).Error; err != nil {
		return
	}

	for _, b := range bans {
		if b.Expires.After(time.Now()) {
			ban = &b
			return
		}
	}

	return
}

type Ban struct {
	gorm.Model
	Creator User `gorm:"ForeignKey:CreatorID"`
	CreatorID uint `gorm:"not_null"`
	User User `gorm:"ForeignKey:UserID"`
	UserID uint `gorm:"not_null"`
	Post Post `gorm:"ForeignKey:PostID"`
	PostID uint
	Expires time.Time
	Permanent bool `gorm:"not_null;default:false"`
	Reason string
}
