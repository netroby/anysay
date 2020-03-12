package controller

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	_ "github.com/mattn/go-sqlite3"
	lru "github.com/netroby/fastlru"
	"github.com/netroby/anysay/app/views"
	"github.com/netroby/anysay/app/views/admin"
	. "github.com/netroby/anysay/pkg"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AdminLoginForm is the login form for Admin
type AdminLoginForm struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// BlogItem is the blog item
type BlogItem struct {
	Title   string `form:"title" binding:"required"`
	Content string `form:"content" binding:"required"`
}
type EditBlogItem struct {
	Aid     string `form:"aid" binding:"required"`
	Title   string `form:"title" binding:"required"`
	Content string `form:"content" binding:"required"`
}

type AdminController struct {
}

// ListBlogCtr is list blogs for admin
func (ac *AdminController) ListBlogCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	fmt.Println(username)
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		log.Fatal(err)
	}
	page -= 1
	if page < 0 {
		page = 0
	}

	prev_page := page
	if prev_page < 1 {
		prev_page = 1
	}
	next_page := page + 2

	var blogList string
	rpp := 20
	offset := page * rpp
	log.Println(rpp)
	log.Println(offset)
	pSql := "Select aid, title, views from gs_article where publish_status = 1 order by aid desc limit ?, ? "
	rows, err := DB.Query(pSql, &offset, &rpp)
	if err != nil {
		LogError(err)
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		var (
			aid   int
			title sql.NullString
			views int
		)
		for rows.Next() {
			err := rows.Scan(&aid, &title, &views)
			if err != nil {
				LogError(err)
			}
			blogList += fmt.Sprintf(
				"<li><a href=\"/view/%d\">%s</a> 点击数(%d)    [<a href=\"/admin/editblog/%d\">编辑</a>] </li>",
				aid,
				title.String,
				views,
				aid,
			)
		}
		err = rows.Err()
		if err != nil {
			LogError(err)
		}
	}

	OutPutHtml(c, admin.AdminListBlog(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"blogList":        blogList,
		"username":        username.(string),
		"prevPage":        fmt.Sprintf("%d", prev_page),
		"nextPage":        fmt.Sprintf("%d", next_page),
	}))
	return
}

// Export
func (ac *AdminController) ExportCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	fmt.Println(username)
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}

	type blogItemNode struct {
		Aid          int
		Title        string
		Content      string
		Publish_time string
		Views        int
	}
	var blogList []blogItemNode
	rows, err := DB.Query("Select aid, title, content,publish_time, views from gs_article where publish_status = 1 order by aid desc")
	if err != nil {
		LogError(err)
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		var (
			aid          int
			title        sql.NullString
			content      sql.NullString
			publish_time sql.NullString
			views        int
		)
		for rows.Next() {
			err := rows.Scan(&aid, &title, &content, &publish_time, &views)
			if err != nil {
				LogError(err)
			} else {
				blogList = append(blogList, blogItemNode{
					Aid:          aid,
					Title:        title.String,
					Content:      content.String,
					Publish_time: publish_time.String,
					Views:        views,
				})
			}
		}
		err = rows.Err()
		if err != nil {
			LogError(err)
		}
	}
	c.JSON(http.StatusOK, blogList)
}

func (ac *AdminController) EditBlogCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	id := c.Param("id")
	var blog VBlogItem
	rows, err := DB.Query("select aid, title, content, publish_time, publish_status, views  from gs_article where aid = ?", &id)
	if err != nil {
		LogError(err)
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		var ()
		for rows.Next() {
			err := rows.Scan(&blog.Aid, &blog.Title, &blog.Content, &blog.Publish_time, &blog.Publish_status, &blog.Views)
			if err != nil {
				LogError(err)
			}
		}
		err = rows.Err()
		if err != nil {
			LogError(err)
		}
	}

	OutPutHtml(c, views.EditBlog(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"username":        username.(string),
		"aid":             fmt.Sprintf("%d", blog.Aid),
		"title":           blog.Title.String,
		"content":         blog.Content.String,
		"publishTime":     blog.Publish_time.String,
		"views":           fmt.Sprintf("%d", blog.Views),
	}))
	return
}

func (ac *AdminController) DeleteBlogCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	var BI EditBlogItem
	err := c.MustBindWith(&BI, binding.Form)
	if err != nil {
		LogError(err)
		(&Umsg{err.Error(), "/"}).ShowMessage(c)
		return
	} else {
		if BI.Aid == "" {
			(&Umsg{"文章未找到", "/"}).ShowMessage(c)
			return
		}
		_, err := DB.Exec("delete from gs_article where aid = ?", BI.Aid)
		if err == nil {
			Cache = lru.New(CacheSize)
			(&Umsg{"删除成功", "/"}).ShowMessage(c)
		} else {
			LogError(err)
			(&Umsg{"删除失败", "/"}).ShowMessage(c)
		}
	}
}

func (ac *AdminController) AddBlogCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"没有权限", "/"}).ShowMessage(c)
		return
	}
	OutPutHtml(c, views.AddBlog(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
	}))
	return
}

func (ac *AdminController) SaveBlogEditCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "javascript:history.go(-1)"}).ShowMessage(c)
		return
	}
	var BI EditBlogItem
	err := c.MustBindWith(&BI, binding.Form)
	if err != nil {
		LogError(err)
		(&Umsg{err.Error(), "javascript:history.go(-1)"}).ShowMessage(c)
		return
	}
	if BI.Aid == "" {
		(&Umsg{"文章未找到", "javascript:history.go(-1)"}).ShowMessage(c)
		return
	}
	if BI.Title == "" {
		(&Umsg{"标题不能为空", "javascript:history.go(-1)"}).ShowMessage(c)
		return
	}
	if BI.Content == "" {
		(&Umsg{"内容不能为空", "javascript:history.go(-1)"}).ShowMessage(c)
		return
	}
	_, err = DB.Exec("update gs_article set title=?, content=? where aid = ? ", BI.Title, BI.Content, BI.Aid)
	if err == nil {
		CKey := fmt.Sprintf("blogitem-%s", BI.Aid)
		LogInfo("Remove cache Key:" + CKey)
		//清除缓存
		Cache.Remove(CKey)
		Cache.Remove("home-page")

		(&Umsg{"成功", "/view/" + BI.Aid}).ShowMessage(c)
	} else {
		LogError(err)
		(&Umsg{"保存失败", "javascript:history.go(-1)"}).ShowMessage(c)
	}

}

func (ac *AdminController) getLastAid() int {
	rows, err := DB.Query("select max(aid) as aid  from gs_article ")
	if err != nil {
		LogError(err)
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		aid := 0
		if rows.Next() {
			err := rows.Scan(&aid)
			if err != nil {
				LogError(err)
			}
		} else {
			return 0
		}
		err = rows.Err()
		if err != nil {
			LogError(err)
		}
		return aid
	}
	return 0
}
func (ac *AdminController) SaveBlogAddCtr(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	var BI BlogItem
	err := c.MustBindWith(&BI, binding.Form)
	if err != nil {
		(&Umsg{err.Error(), "/"}).ShowMessage(c)
		LogError(err)
		return
	}
	if BI.Title == "" {
		(&Umsg{"标题不能为空", "/"}).ShowMessage(c)
		return
	}
	if BI.Content == "" {
		(&Umsg{"内容不能为空", "/"}).ShowMessage(c)
		return
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		LogError(err)
		(&Umsg{"获取时间错误", "/"}).ShowMessage(c)
		return
	}
	aid := ac.getLastAid()
	nextAid := aid + 1
	_, err = DB.Exec(
		"insert into gs_article (aid, title, content, publish_time, publish_status) values (?, ?, ?, ?, 1)", nextAid,
		BI.Title, BI.Content, time.Now().In(loc).Format("2006-01-02 15:04:05"))
	if err == nil {
		Cache.Remove("home-page")
		(&Umsg{"成功", "/"}).ShowMessage(c)
	} else {
		LogError(err)
		(&Umsg{"失败", "/"}).ShowMessage(c)
	}

}

func (ac *AdminController) Files(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	objectLists := make([]string, 0)
	s, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(Config.ObjectStorage.Aws_region),
		Credentials: credentials.NewStaticCredentials(
			Config.ObjectStorage.Aws_access_key_id,
			Config.ObjectStorage.Aws_secret_access_key,
			"",
		),
	})
	if err != nil {
		LogError(err)
		(&Umsg{err.Error(), "/"}).ShowMessage(c)
		return
	}
	s3o := s3.New(s)
	params := &s3.ListObjectsInput{
		Bucket: aws.String(Config.ObjectStorage.Aws_bucket),
	}
	resp, err := s3o.ListObjects(params)
	if err != nil {
		LogError(err)
	} else {
		for _, key := range resp.Contents {
			if strings.Contains(*key.Key, ".") {
				objectLists = append(objectLists, *key.Key)
				fmt.Println(*key.Key)
			}
		}
	}

	OutPutHtml(c, admin.AdminFiles(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"cdnurl":          Config.ObjectStorage.Cdn_url,
		"username":        username.(string),
	}, objectLists))
	return
}
func (ac *AdminController) FileUpload(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		(&Umsg{"需要登录", "/"}).ShowMessage(c)
		return
	}
	s := awsSession.New(&aws.Config{
		Region: aws.String(Config.ObjectStorage.Aws_region),
		Credentials: credentials.NewStaticCredentials(
			Config.ObjectStorage.Aws_access_key_id,
			Config.ObjectStorage.Aws_secret_access_key,
			"",
		),
	})
	s3o := s3.New(s)

	file, fileHeader, err := c.Request.FormFile("uploadfile")
	if err != nil {
		LogError(err)
		(&Msg{"失败"}).ShowMessage(c)
		return
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		LogError(err)
		(&Msg{"失败"}).ShowMessage(c)
		return
	}
	prefix := time.Now().In(loc).Format("2006/01/02")
	body, err := ioutil.ReadAll(file)
	params := &s3.PutObjectInput{
		Bucket:      aws.String(Config.ObjectStorage.Aws_bucket),
		Key:         aws.String(fmt.Sprintf("%s/%s", prefix, fileHeader.Filename)),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(fileHeader.Header.Get("content-type")),
	}
	_, err = s3o.PutObject(params)
	if err != nil {
		LogError(err)
		(&Msg{"失败"}).ShowMessage(c)
		return
	}
	(&Umsg{"成功", "/admin/files"}).ShowMessage(c)
}

func (ac *AdminController) LoginCtr(c *gin.Context) {

	OutPutHtml(c, admin.AdminLogin(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
	}))
	return
}

func (ac *AdminController) LoginProcessCtr(c *gin.Context) {
	var form AdminLoginForm
	err := c.MustBindWith(&form, binding.Form)
	if err != nil {
		LogError(err)
		(&Msg{"登录失败"}).ShowMessage(c)
		return
	}
	session := sessions.Default(c)
	if form.Username == Config.Admin_user && form.Password == Config.Admin_password {
		session.Set("username", Config.Admin_user)
		err := session.Save()
		if err != nil {
			LogError(err)
		}
		c.Redirect(301, "/")
	} else {
		session.Delete("username")
		err := session.Save()
		if err != nil {
			LogError(err)
		}
		(&Umsg{"登录失败", "/"}).ShowMessage(c)
	}
}

func (ac *AdminController) LogoutCtr(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("username")
	err := session.Save()
	if err != nil {
		LogError(err)
	}
	c.Redirect(301, "/")
}
