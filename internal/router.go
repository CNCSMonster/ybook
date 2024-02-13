package internal

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cncsmonster/fspider"
	"github.com/cncsmonster/ybook/pkg"
	"github.com/cncsmonster/ybook/pkg/log"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	r                        *gin.Engine
	config                   *pkg.Config
	blogCache, searcherCache pkg.Cache
	hide, private            pkg.GitIgnorer
	slmt, mlmt, hlmt         *limiter.Limiter
	blogLoader               *pkg.BlogLoader
	blogIndexer              pkg.BlogIndexer
	searchers                *sync.Map
	done                     func() error
}

func NewApp(configPath string) *App {
	config := pkg.LoadConfig(configPath)
	if config == nil {
		return nil
	}
	spider := fspider.NewSpider()
	spider.Spide(config.BLOG_PATH)

	blogCache := pkg.NewCache(1000)
	searcherCache := pkg.NewCache(1000)
	hide := pkg.NewBlogIgnorer().AddPatterns(config.HIDE_PATHS...)
	private := pkg.NewBlogIgnorer().AddPatterns(config.PRIVATE_PATHS...)
	// set visit rate limit for each ip and each path
	slmt := tollbooth.NewLimiter(float64(config.RATE_LIMITE_SECOND), &limiter.ExpirableOptions{DefaultExpirationTTL: time.Second}) // 每秒最多5次
	mlmt := tollbooth.NewLimiter(float64(config.RATE_LIMITE_MINUTE), &limiter.ExpirableOptions{DefaultExpirationTTL: time.Minute}) // 每分钟最多30次
	hlmt := tollbooth.NewLimiter(float64(config.RATE_LIMITE_HOUR), &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})     // 每小时最多1000次
	blogLoader := pkg.NewBlogLoader(config)
	blogIndexer := pkg.NewBlogIndexer(config.APP_DATA_PATH + "/blog.bleve")
	searchers := &sync.Map{}
	//往searchers中加入内置搜索
	searchers.Store("title", pkg.NewSearcherByTitle("title", "根据标题编辑距离搜索", spider, blogLoader))
	searchers.Store("content", pkg.NewSearchByContentMatch("content", "根据文本内容匹配搜索", spider, blogCache, blogLoader))
	searchers.Store("keyword", pkg.NewSearcherByKeywork("keyword", "根据关键词搜索", spider, blogCache, blogLoader))
	searchers.Store("bleve", pkg.NewSearcherByBleve("bleve", "根据bleve搜索", blogIndexer))
	//往searchers中加入插件搜索
	for _, plugin := range config.SEARCH_PLUGINS {
		if plugin.Disable {
			searchers.Delete(plugin.Name)
			continue
		}
		searcher := pkg.NewSearcherByPlugin(plugin, blogLoader, config)
		searchers.Store(plugin.Name, searcher)
	}
	//启动文件变化监听
	// 处理缓存更新,当文件更新时,更新博客缓存
	go func() {
		Changed := spider.FilesChanged()
		for path := range Changed {
			path = pkg.SimplifyPath(path)
			log.Println("[cache] remove:", path)
			blogCache.Remove(path)
			dir := filepath.Dir(path)
			log.Println("[cache] remove:", dir)
			blogCache.Remove(dir)
		}
		log.Println("[cache] finished")
	}()
	// 处理博客索引更新,当文件更新时,更新博客索引,以及更新搜索缓存
	go func() {
		for _, path := range spider.AllPaths() {
			path = pkg.SimplifyPath(path)
			if pkg.PathMatch(path, hide, private) {
				continue
			}
			blog, err := blogLoader.LoadBlog(path)
			if err == nil {
				blogIndexer.Add(blog)
			}
		}
		Changed := spider.FilesChanged()
		for path := range Changed {
			searcherCache.RemoveAll()
			path := pkg.SimplifyPath(path)
			if pkg.PathMatch(path, hide, private) {
				blogIndexer.Delete(&pkg.BlogItem{Path: path})
				continue
			}
			blog, err := blogLoader.LoadBlog(path)
			if err == nil {
				blogIndexer.Add(blog)
			} else {
				blogIndexer.Delete(&pkg.BlogItem{Path: path})
			}
		}
	}()
	// 处理配置文件更新,当配置文件更新时,配置相关的各种组件的更新
	configSpider := fspider.NewSpider()
	configSpider.Spide(configPath)
	go func() {
		Changed := configSpider.FilesChanged()
		for range Changed {
			if config.Update(configPath) {
				log.Println("[config] update success")
				config.RLock()
				slmt.SetMax(float64(config.RATE_LIMITE_SECOND))
				mlmt.SetMax(float64(config.RATE_LIMITE_MINUTE))
				hlmt.SetMax(float64(config.RATE_LIMITE_HOUR))
				hide.UpdatePatterns(config.HIDE_PATHS...)
				private.UpdatePatterns(config.PRIVATE_PATHS...)
				config.RUnlock()
			} else {
				log.Println("[config] update fail")
			}
		}
	}()
	var done = func() error {
		spider.Stop()
		configSpider.Stop()
		return nil
	}
	return &App{
		r:             gin.Default(),
		config:        config,
		blogCache:     blogCache,
		searcherCache: searcherCache,
		hide:          hide,
		private:       private,
		slmt:          slmt,
		mlmt:          mlmt,
		hlmt:          hlmt,
		blogLoader:    blogLoader,
		blogIndexer:   blogIndexer,
		searchers:     searchers,
		done:          done,
	}
}

func (app *App) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recover from", r)
			err = app.done()
		}
	}()
	r := app.r
	config := app.config
	slmt, mlmt, hlmt := app.slmt, app.mlmt, app.hlmt
	private, hide := app.private, app.hide
	blogCache := app.blogCache
	searcherCache := app.searcherCache
	blogLoader := app.blogLoader
	searchers := app.searchers
	_ = hide

	r.Use(cors.Default())
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || c.Request.URL.Path == "/favicon.ico" {
			config.RLock()
			newUrl := config.BLOG_ROUTER + "/" + c.Request.URL.Path
			config.RUnlock()
			c.Redirect(http.StatusMovedPermanently, newUrl)
			c.Abort()
			return
		}
	})
	r.Use(LimitMiddleware(slmt, mlmt, hlmt))
	// blog
	blog := r.Group(config.BLOG_ROUTER)
	blog.Use(PrivateMiddleWare(private, config))
	blog.Use(BlogCacheMiddleware(blogCache, config))
	blog.Use(GenMiddleWare(blogCache, config))
	blog.Use(LoadBlogMiddleware(blogCache, blogLoader))
	blog.GET("/*any")
	// api
	api := r.Group(config.API_ROUTER)
	api.GET("/search", SearchMiddleWare(searchers, searcherCache, config))
	api.GET("/searchers", func(c *gin.Context) {
		type JsonSearcher struct {
			Type  string `json:"type"`
			Brief string `json:"brief"`
		}
		jsonSearchers := make([]JsonSearcher, 0)
		searchers.Range(func(key, value interface{}) bool {
			searcher := value.(pkg.Searcher)
			jsonSearchers = append(jsonSearchers, JsonSearcher{
				Type:  searcher.Name(),
				Brief: searcher.Brief(),
			})
			return true
		})
		sort.Slice(jsonSearchers, func(i, j int) bool {
			return jsonSearchers[i].Type < jsonSearchers[j].Type
		})
		c.JSON(http.StatusOK, jsonSearchers)
	})
	config.RLock()
	port := fmt.Sprintf(":%d", config.PORT)
	config.RUnlock()
	return r.Run(port)
}
