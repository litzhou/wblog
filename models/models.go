package models

import (
	"github.com/jinzhu/gorm"
	//_ "github.com/jinzhu/gorm/dialects/sqlite"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"time"
)

// I don't need soft delete,so I use customized BaseModel instead gorm.Model
type BaseModel struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// table pages
type Page struct {
	BaseModel
	Title       string // title
	Body        string // body
	View        int    // view count
	IsPublished string // published or not
}

// table posts
type Post struct {
	BaseModel
	Title       string     // title
	Body        string     // body
	View        int        // view count
	IsPublished string     // published or not
	Tags        []*Tag     `gorm:"-"` // tags of post
	Comments    []*Comment `gorm:"-"` // comments of post
}

// table tags
type Tag struct {
	BaseModel
	Name  string // tag name
	Total int    `gorm:"-"` // count of post
}

// table post_tags
type PostTag struct {
	BaseModel
	PostId uint // post id
	TagId  uint // tag id
}

// table users
type User struct {
	gorm.Model
	Email         string    `gorm:"unique_index;default:null"` //邮箱
	Telephone     string    `gorm:"unique_index;default:null"` //手机号码
	Password      string    `gorm:"default:null"`              //密码
	VerifyState   string    `gorm:"default:'0'"`               //邮箱验证状态
	SecretKey     string    `gorm:"default:null"`              //密钥
	OutTime       time.Time //过期时间
	GithubLoginId string    `gorm:"unique_index"` // github唯一标识
	IsAdmin       bool      //是否是管理员
	AvatarUrl     string    // 头像链接
	NickName      string    // 昵称
}

// table comments
type Comment struct {
	BaseModel
	UserID  uint   // 用户id
	Content string // 内容
	PostID  uint   // 文章id
	//Replies []*Comment // 评论
}

// query result
type QrArchive struct {
	ArchiveDate time.Time //month
	Total       int       //total
	Year        int       // year
	Month       int       // month
}

var DB *gorm.DB

func InitDB() *gorm.DB {
	//db, err := gorm.Open("sqlite3", "wblog.db")
	db, err := gorm.Open("mysql", "root:mysql@/wblog?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	DB = db

	db.AutoMigrate(&Page{}, &Post{}, &Tag{}, &PostTag{}, &User{}, &Comment{})
	db.Model(&PostTag{}).AddUniqueIndex("uk_post_tag", "post_id", "tag_id")

	return db
}

// Page
func (page *Page) Insert() error {
	return DB.Create(page).Error
}

func (page *Page) Update() error {
	return DB.Model(page).Update(Page{Title: page.Title, Body: page.Body, IsPublished: page.IsPublished}).Error
}

func (page *Page) Delete() error {
	return DB.Delete(page).Error
}

func GetPageById(id string) (*Page, error) {
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	var page Page
	err = DB.First(&page, "id = ?", pid).Error
	return &page, err
}

func ListPage() ([]*Page, error) {
	var pages []*Page
	err := DB.First(pages).Error
	return pages, err
}

// Post
func (post *Post) Insert() error {
	return DB.Create(post).Error
}

func (post *Post) Update() error {
	p := Post{Title: post.Title}
	p.Body = post.Body
	return DB.Model(post).Update(p).Error
}

func (post *Post) Delete() error {
	return DB.Delete(post).Error
}

func ListPost(tag string) ([]*Post, error) {
	var posts []*Post
	var err error
	if len(tag) > 0 {
		tagId, err := strconv.ParseUint(tag, 10, 64)
		if err != nil {
			return nil, err
		}
		rows, err := DB.Raw("select p.* from posts p inner join post_tags pt on p.id = pt.post_id where pt.tag_id = ?", tagId).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var post Post
			DB.ScanRows(rows, &post)
			posts = append(posts, &post)
		}
	} else {
		err = DB.Find(&posts).Error
	}
	return posts, err
}

func GetPostById(id string) (*Post, error) {
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	var post Post
	err = DB.First(&post, "id = ?", pid).Error
	return &post, err
}

func ListPostArchives() ([]*QrArchive, error) {
	var archives []*QrArchive
	sql := `select DATE_FORMAT(created_at,'%Y-%m') as month,count(*) as total from posts group by month order by month desc`
	rows, err := DB.Raw(sql).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var archive QrArchive
		var month string
		rows.Scan(&month, &archive.Total)
		//DB.ScanRows(rows, &archive)
		archive.ArchiveDate, _ = time.Parse("2006-01", month)
		archive.Year = archive.ArchiveDate.Year()
		archive.Month = int(archive.ArchiveDate.Month())
		archives = append(archives, &archive)
	}
	return archives, nil
}

func ListPostByArchive(year, month string) ([]*Post, error) {
	if len(month) == 1 {
		month = "0" + month
	}
	condition := fmt.Sprintf("%s-%s", year, month)
	rows, err := DB.Raw("select * from posts where date_format(created_at,'%Y-%m') = ?", condition).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]*Post, 0)
	for rows.Next() {
		var post Post
		DB.ScanRows(rows, &post)
		posts = append(posts, &post)
	}
	return posts, nil
}

// Tag
func (tag *Tag) Insert() error {
	return DB.FirstOrCreate(tag, "name = ?", tag.Name).Error
}

func ListTag() ([]*Tag, error) {
	var tags []*Tag
	rows, err := DB.Raw("select t.*,count(*) total from tags t inner join post_tags pt on t.id = pt.tag_id group by pt.tag_id").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag Tag
		DB.ScanRows(rows, &tag)
		tags = append(tags, &tag)
	}
	return tags, nil
}

func ListTagByPostId(id string) ([]*Tag, error) {
	var tags []*Tag
	pid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}
	rows, err := DB.Raw("select t.* from tags t inner join post_tags pt on t.id = pt.tag_id where pt.post_id = ?", uint(pid)).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag Tag
		DB.ScanRows(rows, &tag)
		tags = append(tags, &tag)
	}
	return tags, nil
}

// user
// insert user
func (user *User) Insert() error {
	return DB.Create(user).Error
}

// update user
func (user *User) Update() error {
	return DB.Save(user).Error
}

//
func GetUserByUsername(username string) (*User, error) {
	var user User
	err := DB.First(&user, "email = ?", username).Error
	return &user, err
}

//
func (user *User) FirstOrCreate() (*User, error) {
	err := DB.FirstOrCreate(user, "github_login_id = ?", user.GithubLoginId).Error
	return user, err
}

func GetUser(id interface{}) (*User, error) {
	var user User
	err := DB.First(&user, id).Error
	return &user, err
}

func (user *User) UpdateProfile(avatarUrl, nickName string) error {
	return DB.Model(user).Update(User{AvatarUrl: avatarUrl, NickName: nickName}).Error
}

func (user *User) UpdateEmail(email string) error {
	return DB.Model(user).Update(User{Email: email}).Error
}

func (user *User) UpdateGithubId(githubId string) error {
	return DB.Model(user).Update(User{GithubLoginId: githubId}).Error
}

// Comment
func (comment *Comment) Insert() error {
	return DB.Create(comment).Error
}

func (comment *Comment) Delete() error {
	return DB.Delete(comment).Error
}

func ListCommentByPostID(postId string) ([]*Comment, error) {
	pid, err := strconv.ParseUint(postId, 10, 64)
	if err != nil {
		return nil, err
	}
	var comments []*Comment
	err = DB.Find(&comments, "post_id = ?", pid).Error
	return comments, err
}

func GetComment(id interface{}) (*Comment, error) {
	var comment Comment
	err := DB.First(&comment, id).Error
	return &comment, err
}