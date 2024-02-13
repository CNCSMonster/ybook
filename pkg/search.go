package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cncsmonster/fspider"
	"github.com/cncsmonster/ybook/pkg/log"
	"github.com/texttheater/golang-levenshtein/levenshtein"
)

type Searcher interface {
	Search(keyword string, num int) ([]string, error)
	Name() string
	Brief() string
}

type SearcherPlugin struct {
	Name    string
	Brief   string
	Type    string
	Command string
	Disable bool
	Url     string
}

// searcherImpl
type searcherImpl struct {
	f     func(keyword string, num int) ([]string, error)
	name  string
	brief string
}

// SearcherFunc implements Searcher interface
func (s searcherImpl) Search(keyword string, num int) ([]string, error) {
	return s.f(keyword, num)
}

func (s searcherImpl) Name() string {
	return s.name
}
func (s searcherImpl) Brief() string {
	return s.brief
}

// searcher according to title edit distance
func NewSearcherByTitle(name, brief string, spider fspider.Spider, loader *BlogLoader) Searcher {
	f := func(keyword string, num int) ([]string, error) {
		paths := spider.AllPaths()
		results := make([]string, 0, num)
		type _Item struct {
			path string
			dist int
		}
		items := make([]_Item, 0, len(paths))
		for _, path := range paths {
			if PathMatch(path, loader.Hide, loader.Private) {
				continue
			}
			title := filepath.Base(path)
			title = title[:len(title)-len(filepath.Ext(title))]
			dist := levenshtein.DistanceForStrings([]rune(keyword), []rune(title), levenshtein.DefaultOptions)
			items = append(items, _Item{path: path, dist: dist})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].dist < items[j].dist {
				return true
			} else if items[i].dist > items[j].dist {
				return false
			} else {
				return items[i].path < items[j].path
			}
		})

		for _, item := range items {
			results = append(results, item.path)
			if len(results) >= num {
				break
			}
		}
		return results, nil
	}
	return searcherImpl{
		f:     f,
		name:  name,
		brief: brief,
	}
}

// searcher according to plugin ; this func is not thread-safe
func NewSearcherByPlugin(plugin SearcherPlugin, loader *BlogLoader, config *Config) Searcher {
	var f func(keyword string, num int) ([]string, error)
	if plugin.Type == "command" {
		f = func(keyword string, num int) ([]string, error) {
			commands := strings.Split(plugin.Command, "|")
			BLOG_PATH := config.BLOG_PATH
			KEY_WORD := keyword
			NUM := fmt.Sprintf("%d", num)
			var ignoress []string
			ignoress = append(ignoress, config.HIDE_PATHS...)
			ignoress = append(ignoress, config.PRIVATE_PATHS...)
			IGNORE := strings.Join(ignoress, ",")
			var lastStdout io.Reader
			var bs []byte
			var err error
			for i, cmdStr := range commands {
				cmdStr = strings.TrimSpace(cmdStr)
				args := strings.Split(cmdStr, " ")
				for i, arg := range args {
					args[i] = strings.TrimSpace(arg)
					args[i] = strings.ReplaceAll(args[i], "${BLOG_PATH}", BLOG_PATH)
					args[i] = strings.ReplaceAll(args[i], "${KEY_WORD}", KEY_WORD)
					args[i] = strings.ReplaceAll(args[i], "${NUM}", NUM)
					args[i] = strings.ReplaceAll(args[i], "${IGNORE}", IGNORE)
				}
				log.Println("[search by command] command:", args)
				cmd := exec.Command(args[0])
				cmd.Args = args
				if lastStdout != nil {
					cmd.Stdin = lastStdout
				}
				if i == len(commands)-1 {
					bs, err = cmd.Output()
					if err != nil {
						log.Println("[search by command] failed to exec command:", err)
						return nil, err
					}
					break
				}
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					log.Println("[search by command] failed to get stdout pipe:", err)
					return nil, err
				}
				lastStdout = stdout
				err = cmd.Start()
				if err != nil {
					log.Println("[search by command] failed to start command:", err)
					return nil, err
				}
			}
			bs = bytes.TrimSpace(bs)
			bss := bytes.Split(bs, []byte("\n"))
			results := make([]string, 0, len(bss))
			for _, bs := range bss {
				path := string(bs)
				if PathMatch(path, loader.Hide, loader.Private) {
					continue
				}
				results = append(results, path)
			}
			return results, nil
		}
	} else if plugin.Type == "url" {
		f = func(keyword string, num int) ([]string, error) {
			// put a get request to url
			resp, err := http.Get(fmt.Sprintf("%s?keyword=%s&num=%d", plugin.Url, keyword, num))
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			bs, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			// ["/path1", "/path2", ...]
			var results []string
			if err := json.Unmarshal(bs, &results); err != nil {
				return nil, err
			}
			return results, nil
		}
	} else {
		panic("unknown plugin type")
	}
	return searcherImpl{
		f:     f,
		name:  plugin.Name,
		brief: plugin.Brief,
	}
}

// searcher according to search-keyword and keywords in meta
func NewSearcherByKeywork(name, brief string, spider fspider.Spider, cache Cache, blogLoader *BlogLoader) Searcher {
	var hide, private GitIgnorer = blogLoader.Hide, blogLoader.Private
	f := func(keyword string, num int) ([]string, error) {
		log.Println("[search by keyword] keyword:", keyword)
		results := make([]string, 0, num)
		paths := spider.AllPaths()
		for _, path := range paths {
			if PathMatch(path, hide, private) {
				continue
			}
			url := blogLoader.Path2Url(path)
			blog, found := cache.Get("blog:" + url)
			var blogItem *BlogItem
			if !found {
				blog, err := blogLoader.LoadBlog(path)
				if err != nil {
					continue
				}
				cache.Set("blog:"+url, blog)
				blogItem = blog
			} else {
				blogItem = blog.(*BlogItem)
			}
			for _, kw := range blogItem.Meta.KeyWords {
				if strings.Contains(kw, keyword) {
					results = append(results, path)
					break
				}
			}
			if len(results) >= num {
				break
			}
		}
		return results, nil
	}
	return searcherImpl{
		f:     f,
		name:  name,
		brief: brief,
	}
}

// according to the times of keyword in {content, title, meta}
func NewSearchByContentMatch(name, brief string, spider fspider.Spider, cache Cache, blogLoader *BlogLoader) Searcher {
	f := func(keyword string, num int) ([]string, error) {
		paths := spider.AllPaths()
		results := make([]string, 0, num)
		type _Item struct {
			path string
			num  int
		}
		items := make([]_Item, 0, len(paths))
		for _, path := range paths {
			if PathMatch(path, blogLoader.Hide, blogLoader.Private) {
				continue
			}
			url := blogLoader.Path2Url(path)
			blog, found := cache.Get("blog:" + url)
			var blogItem *BlogItem
			if !found {
				blog, err := blogLoader.LoadBlog(path)
				if err != nil {
					continue
				}
				cache.Set("blog:"+url, blog)
				blogItem = blog
			} else {
				blogItem = blog.(*BlogItem)
			}
			num := strings.Count(blogItem.File, keyword)
			num += strings.Count(blogItem.Meta.Title, keyword)
			num += strings.Count(blogItem.Meta.Description, keyword)
			items = append(items, _Item{path: path, num: num})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].num > items[j].num {
				return true
			} else if items[i].num < items[j].num {
				return false
			} else {
				return items[i].path < items[j].path
			}
		})
		for _, item := range items {
			results = append(results, item.path)
			if len(results) >= num {
				break
			}
		}
		return results, nil
	}
	return searcherImpl{
		f:     f,
		name:  name,
		brief: brief,
	}
}

// searcher according to bleve file index engine
func NewSearcherByBleve(name, brief string, blogIndexer BlogIndexer) Searcher {
	f := func(keyword string, num int) ([]string, error) {
		results, err := blogIndexer.Search(keyword, num)
		if err != nil {
			return nil, err
		}
		return results, nil
	}
	return searcherImpl{
		f:     f,
		name:  name,
		brief: brief,
	}
}
