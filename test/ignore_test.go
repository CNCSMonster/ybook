package eb

import (
	"path/filepath"
	"testing"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/assert"
)

// 编写单元测试

func TestIgnore(t *testing.T) {
	s1 := "blog/private"
	s2 := "blog\\private" // windows风格

	// 把windows风格的路径转换为unix风格
	s2 = filepath.ToSlash(s2)

	assert.Equal(t, s1, s2)
	ignore := ignore.CompileIgnoreLines(s1)
	assert.True(t, ignore.MatchesPath(s2), "s1: %s, s2: %s", s1, s2)
}

func TestIgnore2(t *testing.T) {
	ignore := ignore.CompileIgnoreLines("blog/private")
	assert.True(t, ignore.MatchesPath("blog/private"))
}

func TestIgnore3(t *testing.T) {
	ignore := ignore.CompileIgnoreLines("blog/private")
	assert.True(t, ignore.MatchesPath("blog\\private"))
}

func TestIgnore4(t *testing.T) {
	ignore := ignore.CompileIgnoreLines("private")
	assert.True(t, ignore.MatchesPath("blog/private"))
}
