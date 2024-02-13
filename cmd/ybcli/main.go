package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/cncsmonster/ybook/pkg"
	"github.com/gocolly/colly"
)

func main() {
	// 创建一个新的Collector
	c := colly.NewCollector(
		// 访问的最大深度，0表示无限制
		colly.MaxDepth(0),
	)

	visited := map[string]bool{}
	visitiedLock := sync.RWMutex{}
	// 在找到每个链接时调用
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if strings.HasPrefix(link, "https") || strings.HasPrefix(link, "http") {
			return
		}
		url := e.Request.URL.String()
		if !strings.HasPrefix(link, "/") {
			link = url + link
		}
		visitiedLock.RLock()
		hasVisited := visited[link]
		visitiedLock.RUnlock()
		if hasVisited {
			return
		}
		visitiedLock.Lock()
		visited[link] = true
		visitiedLock.Unlock()
		e.Request.Visit(link)
	})

	// 在请求发送时调用
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// 可以接受一个命令行参数制定使用的config的位置,接受--config,接受缩写-c
	var configPath string
	// 在命令行中使用--config或者-c指定配置文件的位置
	flag.StringVar(&configPath, "config", "eb.yaml", "config file path")
	flag.StringVar(&configPath, "c", "eb.yaml", "config file path")
	flag.Parse()
	var config = pkg.LoadConfig(configPath)
	// 开始访问
	indexURL := fmt.Sprintf("http://localhost:%d/", config.PORT)
	c.Visit(indexURL)
}
