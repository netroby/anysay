package controller

import (
	"database/sql"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/netroby/anysay/app/views"
	. "github.com/netroby/anysay/pkg"
	"github.com/ztrue/tracerr"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type FrontController struct {
}

func (fc *FrontController) AboutCtr(c *gin.Context) {
	var Config = GetConfig();
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		username = ""
	}
	OutPutHtml(c, views.About(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"username":         username.(string),
	}))
	return
}
func (fc *FrontController) PingCtr(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}
func (fc *FrontController) HomeCtr(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		fmt.Println(err)
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

	rpp := 20
	offset := page * rpp
	CKey := "home-page"
	var blogList string
	val, ok := Cache.Get(CKey)
	if page < 2 && val != nil && ok == true {
		log.Debug().Msg("缓存命中, 当前缓存数: " + fmt.Sprintf("%d", Cache.Len()))
		blogList = val.(string)
	} else {
		rows, err := DB.Query("Select aid, title from gs_article where publish_status = 1 order by aid desc limit ?, ? ", &offset, &rpp)
		defer CloseRowsDefer(rows)
		if err != nil {
			log.Error().Msg(tracerr.Sprint(tracerr.Wrap(err)))
		} else {
			var (
				aid   int
				title sql.NullString
			)
			if rows != nil {
				for rows.Next() {
					err := rows.Scan(&aid, &title)
					if err != nil {
						log.Error().Msg(tracerr.Sprint(tracerr.Wrap(err)))
					} else {
						blogList += fmt.Sprintf(
							"<li><a href=\"/view/%d\">%s</a></li>",
							aid,
							title.String,
						)
					}
				}

				err = rows.Err()
				if err != nil {
					LogError(err)
				}
			}
		}
		if page < 2 {
			go func(CKey string, blogList string) {
				Cache.Add(CKey, blogList)
			}(CKey, blogList)
		}
	}
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		username = ""
	}

	OutPutHtml(c, views.Home(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"blogList":         blogList,
		"username":         username.(string),
		"prevPage":        fmt.Sprintf("%d", prev_page),
		"nextPage":        fmt.Sprintf("%d", next_page),
	}))
	return
}

func (fc *FrontController) SearchCtr(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		fmt.Println(err)
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
	keyword := c.DefaultQuery("keyword", "")
	fmt.Println(keyword)
	if len(keyword) <= 0 {
		(&Msg{"搜索关键字不能为空"}).ShowMessage(c)
		return
	}
	orig_keyword := keyword
	keyword = strings.Trim(keyword, "%20")
	keyword = strings.TrimSpace(keyword)
	keyword = strings.Replace(keyword, " ", "%", -1)
	keyword = strings.Replace(keyword, "%20", "%", -1)

	var blogList string
	rpp := 20
	offset := page * rpp
	rows, err := DB.Query(
		"Select aid, title from gs_article where publish_status = 1 and (title like ? or content like ?) order by aid desc limit ? offset ? ",
		"%"+keyword+"%", "%"+keyword+"%", &rpp, &offset)
	if err != nil {
		fmt.Println(err)
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		var (
			aid   int
			title sql.NullString
		)
		for rows.Next() {
			err := rows.Scan(&aid, &title)
			if err != nil {
				fmt.Println(err)
			}
			blogList += fmt.Sprintf(
				"<li><a href=\"/view/%d\">%s</a></li>",
				aid,
				title.String,
			)
		}
		err = rows.Err()
		if err != nil {
			fmt.Println(err)
		}
	}
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		username = ""
	}

	 OutPutHtml(c, views.Search(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"bloglist":         blogList,
		"keyword":          orig_keyword,
		"username":         username.(string),
		"prevPage":        fmt.Sprintf("%d", prev_page),
		"nextPage":        fmt.Sprintf("%d", next_page),
	}))
	return
}

func (fc *FrontController) ViewAltCtr(c *gin.Context) {
	id := c.DefaultQuery("id", "0")
	c.Redirect(301, fmt.Sprintf("/view/%s", id))
}

func (fc *FrontController) CountViewCtr(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fmt.Println("ID不能为空")
		return
	}
	log.Info().Msg("更新统计, 文章id: " + fmt.Sprintf("%d",  id))
	_, err = DB.Exec("update gs_article set views=views+1 where aid = ? ", &id)
	if err != nil {
		fmt.Println(err)
	}

	vct := 0
	rows, err := DB.Query("select views from gs_article where aid = ? limit 1", &id)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	defer CloseRowsDefer(rows)
	if rows != nil {
		for rows.Next() {
			err := rows.Scan(&vct)
			if err != nil {
				log.Error().Msg(err.Error())
			}
		}
		err = rows.Err()
		if err != nil {
			log.Error().Msg(err.Error())
		}
	}
	c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 UTC")
	c.Header("Cache-Control",  "no-cache, no-store, no-transform, must-revalidate, private, max-age=0")
	c.Header("Pragma", "no-cache")
	c.String(http.StatusOK, fmt.Sprintf("document.getElementById('vct').innerHTML=%d", vct))
}

func (fc *FrontController) ViewCtr(c *gin.Context) {
	id := c.Param("id")
	var blog VBlogItem
	CKey := fmt.Sprintf("blogitem-%s", id)
	val, ok := Cache.Get(CKey)
	if val != nil && ok == true {
		log.Info().Msg("缓存命中, 文章id:" + fmt.Sprintf("%s", id))
		blog = val.(VBlogItem)
	} else {
		rows, err := DB.Query("select aid, title, content, publish_time, publish_status from gs_article where aid = ? limit 1", &id)
		if err != nil {
			fmt.Println(err)
		}
		defer CloseRowsDefer(rows)
		if rows != nil {
			for rows.Next() {
				err := rows.Scan(&blog.Aid, &blog.Title, &blog.Content, &blog.Publish_time, &blog.Publish_status)
				if err != nil {
					fmt.Println(err)
				}
			}
			err = rows.Err()
			if err != nil {
				fmt.Println(err)
			}
			go func(CKey string, blog VBlogItem) {
				Cache.Add(CKey, blog)
			}(CKey, blog)
		}
	}
	session := sessions.Default(c)
	username := session.Get("username")
	if username == nil {
		username = ""
	}
	OutPutHtml(c, views.View(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"aid":              fmt.Sprintf("%d", blog.Aid),
		"title":            blog.Title.String,
		"content":          blog.Content.String,
		"publishTime":     blog.Publish_time.String,
		"username":         username.(string),
	}))
	return

}
