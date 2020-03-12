package main

//go:generate go get -u github.com/valyala/quicktemplate/qtc
//go:generate qtc  -ext=.html  -dir=app/views
//go:generate go run  tools/assets_generate.go

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/netroby/anysay/app/controller"
	. "github.com/netroby/anysay/pkg"
	"github.com/rs/zerolog/log"
)

func main() {
	InitApp()
	r := gin.Default()

	r.StaticFS("/assets", Assets)
	store := cookie.NewStore([]byte("gssecret"))
	r.Use(sessions.Sessions("mysession", store))
	fc := new(FrontController)
	r.GET("/", fc.HomeCtr)
	r.GET("/about", fc.AboutCtr)
	r.GET("/view/:id", fc.ViewCtr)
	r.GET("/view.php", fc.ViewAltCtr)
	r.GET("/ping", fc.PingCtr)
	r.GET("/search", fc.SearchCtr)
	r.GET("/countview/:id", fc.CountViewCtr)

	ac := new(AdminController)
	admin := r.Group("/admin")
	{
		admin.GET("/", ac.ListBlogCtr)
		admin.GET("/login", ac.LoginCtr)
		admin.POST("/login-process", ac.LoginProcessCtr)
		admin.GET("/logout", ac.LogoutCtr)
		admin.GET("/addblog", ac.AddBlogCtr)
		admin.POST("/save-blog-add", ac.SaveBlogAddCtr)
		admin.GET("/listblog", ac.ListBlogCtr)
		admin.GET("/export", ac.ExportCtr)
		admin.GET("/deleteblog/:id", ac.DeleteBlogCtr)
		admin.POST("/save-blog-edit", ac.SaveBlogEditCtr)
		admin.GET("/editblog/:id", ac.EditBlogCtr)
		admin.GET("/files", ac.Files)
		admin.POST("/fileupload", ac.FileUpload)
	}


	log.Info().Msg("Server listen on 127.0.0.1:8073")
	err := r.Run("127.0.0.1:8073")
	if err != nil {
		LogError(err)
	}
}
