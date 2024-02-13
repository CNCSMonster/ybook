package pkg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	fsutil "github.com/cncsmonster/gofsutil"
	"github.com/cncsmonster/ybook/pkg/log"
	"github.com/google/shlex"
	"gopkg.in/yaml.v3"
)

// === meta data for md ===
type Meta struct {
	Title       string   `yaml:"title"`
	KeyWords    []string `yaml:"keywords"`
	Description string   `yaml:"description"`
}
type BlogItem struct {
	// Path 作为唯一标识符
	Path string
	Meta
	Kind int
	File string
	Html string
}

// === blog loader ===

type BlogLoader struct {
	*sync.RWMutex
	BlogPath, BlogRouter        string
	TemplatePath, RenderCommand string
	Hide, Private               GitIgnorer
}

func NewBlogLoader(config *Config) *BlogLoader {
	config.RLock()
	defer config.RUnlock()
	return &BlogLoader{
		RWMutex:       &sync.RWMutex{},
		BlogPath:      config.BLOG_PATH,
		BlogRouter:    config.BLOG_ROUTER,
		TemplatePath:  config.TEMPLATE_PATH,
		RenderCommand: config.RENDER_COMMAND,
		Hide:          NewBlogIgnorer().AddPatterns(config.HIDE_PATHS...),
		Private:       NewBlogIgnorer().AddPatterns(config.PRIVATE_PATHS...),
	}
}

func (loader *BlogLoader) Update(config *Config) {
	loader.Lock()
	defer loader.Unlock()
	config.RLock()
	defer config.RUnlock()
	loader.BlogPath = config.BLOG_PATH
	loader.BlogRouter = config.BLOG_ROUTER
	loader.TemplatePath = config.TEMPLATE_PATH
	loader.RenderCommand = config.RENDER_COMMAND
	loader.Hide.UpdatePatterns(config.HIDE_PATHS...)
	loader.Private.UpdatePatterns(config.PRIVATE_PATHS...)
}

func (loader *BlogLoader) LoadBlog(path string) (*BlogItem, error) {
	loader.RLock()
	defer loader.RUnlock()
	var blogRouter, blogPath, templatePath, renderCommand string = loader.BlogRouter, loader.BlogPath, loader.TemplatePath, loader.RenderCommand
	var hide, private GitIgnorer = loader.Hide, loader.Private
	path = SimplifyPath(path)
	if !fsutil.IsExist(path) {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	var blogItemType int
	var meta Meta
	var file []byte
	var html []byte
	var err error
	var stat os.FileInfo
	stat, err = os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		file, err = RenderDir(path, hide, private, blogRouter, blogPath)
		blogItemType = BLOG_ITEM_KIND_DIR
	} else if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".markdown") {
		file, err = os.ReadFile(path)
		blogItemType = BLOG_ITEM_KIND_OTHER
	} else {
		file, err = os.ReadFile(path)
		blogItemType = BLOG_ITEM_KIND_MD
	}
	if err != nil {
		return nil, err
	}
	if (blogItemType&BLOG_ITEM_KIND_DIR + blogItemType&BLOG_ITEM_KIND_MD) != 0 {
		if meta, err = MdMeta(file); err != nil {
			return nil, err
		} else if meta.Title == "" {
			meta.Title = filepath.Base(path)
			meta.Title = meta.Title[:len(meta.Title)-len(filepath.Ext(meta.Title))]
		}
		if html, err = Md2Html(file, meta.Title, templatePath, renderCommand); err != nil {
			return nil, err
		}
	} else {
		html = file
	}
	return &BlogItem{
		Path: path,
		Meta: meta,
		Kind: blogItemType,
		File: string(file),
		Html: string(html),
	}, nil
}
func (loader *BlogLoader) Url2Path(url string) string {
	loader.RLock()
	defer loader.RUnlock()
	path := loader.BlogPath + "/" + url[len(loader.BlogRouter):]
	return SimplifyPath(path)
}
func (loader *BlogLoader) Path2Url(path string) string {
	loader.RLock()
	defer loader.RUnlock()
	return loader.BlogRouter + path[len(loader.BlogPath):]
}

const (
	BLOG_ITEM_KIND_DIR = 1 << iota
	BLOG_ITEM_KIND_MD
	BLOG_ITEM_KIND_OTHER
)

func (item *BlogItem) IsDir() bool {
	return (item.Kind & BLOG_ITEM_KIND_DIR) != 0
}
func (item *BlogItem) IsMd() bool {
	return (item.Kind & BLOG_ITEM_KIND_MD) != 0
}

func MdMeta(md []byte) (meta Meta, err error) {
	// 使用正则表达式匹配 md 中 开头的--- ---之间的内容
	re := regexp.MustCompile(`(?s)^\s*---(.*?)---`)
	metaBytes := re.Find(md)
	if err := yaml.Unmarshal(metaBytes, &meta); err != nil {
		return meta, err
	}
	return
}

// === md2html ===

// use pandoc to convert md to html
func Md2Html(md []byte, title string, templatePath, renderCommand string) (html []byte, err error) {
	// pandoc -s --template=template.html --toc  --mathjax -f markdown -t html --metadata title="title"
	args := []string{"pandoc", "-s", "--template=" + templatePath, "--toc", "--mathjax", "-f", "markdown", "-t", "html", "--metadata", "title=" + title}
	if renderCommand != "" {
		args, err = shlex.Split(renderCommand)
		if err != nil {
			return nil, err
		}
	}
	log.Println("[md2html] render command:", args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = bytes.NewReader(md)
	bs, err := cmd.Output()
	return bs, err
}

// load md
func RenderDir(path string, hide, private GitIgnorer, blogRouter, blogPath string) (md []byte, err error) {
	path = SimplifyPath(path)
	log.Println("[load md] path is dir:", path)
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dir bytes.Buffer
	for _, item := range items {
		name := item.Name()
		full_path := path + "/" + name
		full_path = SimplifyPath(full_path)
		if PathMatch(full_path, hide, private) {
			log.Println("[load md] path in dir ignored:", full_path)
			continue
		}
		isDir := item.IsDir()
		if isDir {
			full_path += "/"
			name += "/"
		}
		url := blogRouter + full_path[len(blogPath):]
		name = filepath.ToSlash(name)
		dir.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a><br>", url, name))
	}
	md = dir.Bytes()
	return md, err
}
