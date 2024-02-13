package pkg

import (
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
)

type GitIgnorer interface {
	AddPatterns(patterns ...string) GitIgnorer
	UpdatePatterns(patterns ...string) GitIgnorer
	Match(path string) bool
}

type gitIgnorerImpl struct {
	mux      *sync.RWMutex
	ignore   *ignore.GitIgnore
	patterns map[string]struct{}
}

func NewBlogIgnorer() GitIgnorer {
	return &gitIgnorerImpl{mux: &sync.RWMutex{}, ignore: ignore.CompileIgnoreLines(), patterns: make(map[string]struct{})}
}

func (bi *gitIgnorerImpl) AddPatterns(patterns ...string) GitIgnorer {
	bi.mux.Lock()
	for _, pattern := range patterns {
		bi.patterns[pattern] = struct{}{}
	}
	patterns = make([]string, 0, len(bi.patterns))
	for pattern := range bi.patterns {
		patterns = append(patterns, pattern)
	}
	bi.ignore = ignore.CompileIgnoreLines(patterns...)
	bi.mux.Unlock()
	return bi
}

func (bi *gitIgnorerImpl) UpdatePatterns(patterns ...string) GitIgnorer {
	bi.mux.Lock()
	bi.patterns = make(map[string]struct{})
	for _, pattern := range patterns {
		bi.patterns[pattern] = struct{}{}
	}
	bi.ignore = ignore.CompileIgnoreLines(patterns...)
	bi.mux.Unlock()
	return bi
}

func (bi *gitIgnorerImpl) Match(path string) bool {
	bi.mux.RLock()
	match := bi.ignore.MatchesPath(path)
	bi.mux.RUnlock()
	return match
}
