package pkg

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/naoina/toml"
	lru "github.com/netroby/fastlru"
	"github.com/netroby/anysay/app/views"
	"github.com/rs/zerolog/log"
	"github.com/ztrue/tracerr"
	"io/ioutil"
	"os"
	"time"
)

type ShowMessage interface {
	ShowMessage(c *gin.Context)
}
type Msg struct {
	Msg string
}
type Umsg struct {
	Msg string
	Url string
}

type VBlogItem struct {
	Aid            int
	Title          sql.NullString
	Content        sql.NullString
	Publish_time   sql.NullString
	Publish_status sql.NullInt64
	Views          int
}

/**
 * Logging error
 */
func LogError(err error) {
	log.Error().Msg(tracerr.Sprint(tracerr.Wrap(err)))
}

/**
 * Logging info
 */
func LogInfo(msg string) {
	log.Info().Msg(msg)
}

/**
 * close rows defer
 */
func CloseRowsDefer(rows *sql.Rows) {
	_ = rows.Close()
}

/*
* ShowMessage with template
 */
func (m *Msg) ShowMessage(c *gin.Context) {

	OutPutHtml(c, views.Message(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"message": m.Msg,
	}))
	return
}

func (m *Umsg) ShowMessage(c *gin.Context) {

	OutPutHtml(c, views.Message(map[string]string{
		"siteName":        Config.Site_name,
		"siteDescription": Config.Site_description,
		"message": m.Msg,
		"url":     m.Url,
	}))
	return
}

func GetMinutes() string {
	return time.Now().Format("200601021504")
}

func GetDB(config *AppConfig) *sql.DB {
	db, err := sql.Open("sqlite3", config.Dbdsn)

	if err != nil {
		panic(err.Error())
	}
	if db == nil {
		panic("db connect failed")
	}
	_, err = db.Exec(`create table if not exists gs_article ( 
	aid            int(11) not null 
	primary key,
		title          varchar(255),
		content        text,
		publish_time   varchar(255),
		publish_status tinyint(1) default '1',
	views          int(11)    default '1'
	);`)
	if err != nil {
		LogError(err)
	}

	return db
}

type AppConfig struct {
	Dbdsn            string
	Admin_user       string
	Admin_password   string
	Site_name        string
	Site_description string
	SrvMode string
	ObjectStorage    struct {
		Aws_access_key_id     string
		Aws_secret_access_key string
		Aws_region            string
		Aws_bucket            string
		Cdn_url               string
	}
}

func GetConfig() *AppConfig {
	//TODO load config from cmd line argument
	f, err := os.Open("./vol/config.toml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	var config AppConfig
	if err := toml.Unmarshal(buf, &config); err != nil {
		panic(err)
	}
	return &config
}



var (
	Config    *AppConfig
	DB        *sql.DB
	Cache     *lru.Cache
	CacheSize int = 512
)

func InitApp() {
	Config = GetConfig()
	gin.SetMode(Config.SrvMode)
	DB = GetDB(Config)
	Cache = lru.New(CacheSize)
}

func OutPutHtml( c *gin.Context, s string) {
	c.Header("Content-Type", "text/html;charset=UTF-8")
	c.String(200, "%s", s)
	return
}
func OutPutText( c *gin.Context, s string) {
	c.Header("Content-Type", "text/plain;charset=UTF-8")
	c.String(200, "%s", s)
	return
}