package pkg

import (
	"strings"

	"github.com/blevesearch/bleve"
)

// 使用bleve 对博客建立索引
type BlogIndexer interface {
	// 加入对一个博客内容的索引
	Add(blog *BlogItem) error
	// 删除对一个博客内容的索引
	Delete(blog *BlogItem) error
	// 搜索博客内容
	Search(keyword string, num int) ([]string, error)
	// 把对博客内容建立的索引保存到文件
	Close() error
}
type blogIndexerImpl struct {
	Indexer bleve.Index
}
type BlogIndex struct {
	Path string
	Meta
	File string
}

// 加入对一个博客内容的索引
func (bi *blogIndexerImpl) Add(blog *BlogItem) error {
	if strings.HasPrefix(blog.Path, "/blogg/") {
		panic("Add can not use blogg")
	}
	return bi.Indexer.Index(blog.Path, BlogIndex{Path: blog.Path, Meta: blog.Meta, File: blog.File})
}

// 删除对一个博客内容的索引
func (bi *blogIndexerImpl) Delete(blog *BlogItem) error {
	if strings.HasPrefix(blog.Path, "/blogg/") {
		panic("Delete can not delete blogg")
	}
	return bi.Indexer.Delete(blog.Path)
}

func NewBlogIndexer(indexPath string) BlogIndexer {
	index, err := bleve.Open(indexPath)
	if err != nil {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil
		}
	}
	return &blogIndexerImpl{Indexer: index}
}

// 搜索博客内容
func (bi *blogIndexerImpl) Search(keyword string, num int) ([]string, error) {
	query := bleve.NewFuzzyQuery(keyword)
	search := bleve.NewSearchRequest(query)
	search.Size = num
	searchResults, err := bi.Indexer.Search(search)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		if strings.HasPrefix(hit.ID, "/blogg/") {
			panic("can not use blogg")
		}
		results = append(results, hit.ID)
	}
	return results, nil
}

// 把对博客内容建立的索引保存到文件
func (bi *blogIndexerImpl) Close() error {
	return bi.Indexer.Close()
}
