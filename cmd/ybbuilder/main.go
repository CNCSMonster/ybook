package main

import (
	"bytes"
	"fmt"
	"os"

	fsutil "github.com/cncsmonster/gofsutil"
)

func ProcessIncludeFile(includePath string, var_name string, eb *bytes.Buffer) {
	bs, err := os.ReadFile(includePath)
	if err != nil {
		panic(err)
	}
	eb.WriteString(fmt.Sprintf("var %s = []byte{", var_name))
	for _, byte := range bs {
		eb.WriteString(fmt.Sprintf("%d,", byte))
	}
	eb.WriteString("}\n")
}

func main() {
	// 项目构建工具,将一些资源文件转换为go文件
	var configPath string = "./build_resources/yb.toml"
	var templatePath string = "./build_resources/template.html"
	var blogPath string = "./build_resources/intro.md"
	var hidePath string = "./build_resources/hide.md"
	var privatePath string = "./build_resources/private.md"
	var outputPath string = "./cmd/yb/resources.go"
	var helpPath string = "./build_resources/help"
	var versionPath string = "./build_resources/version"
	var keywordPath string = "./build_resources/keyword.md"
	var faviconPath string = "./build_resources/favicon.ico"
	var vueJsPath string = "./build_resources/vue.js"
	eb := bytes.NewBufferString("package main\n\n")
	ProcessIncludeFile(configPath, "DEFAULT_CONFIG", eb)
	ProcessIncludeFile(templatePath, "DEFAULT_TEMPLATE", eb)
	ProcessIncludeFile(blogPath, "DEFAULT_BLOG", eb)
	ProcessIncludeFile(hidePath, "DEFAULT_HIDE", eb)
	ProcessIncludeFile(privatePath, "DEFAULT_PRIVATE", eb)
	ProcessIncludeFile(helpPath, "DEFAULT_HELP", eb)
	ProcessIncludeFile(versionPath, "DEFAULT_VERSION", eb)
	ProcessIncludeFile(keywordPath, "DEFAULT_KEYWORD", eb)
	ProcessIncludeFile(faviconPath, "DEFAULT_FAVICON", eb)
	ProcessIncludeFile(vueJsPath, "DEFAULT_VUE_JS", eb)
	fsutil.MustWrite(outputPath, eb.Bytes())
}
