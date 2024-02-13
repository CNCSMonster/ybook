package main

import (
	"fmt"
	"log"
	"os"

	fsutil "github.com/cncsmonster/gofsutil"
	"github.com/cncsmonster/ybook/internal"
)

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "-s")
	}
	switch os.Args[1] {
	case "-h":
		Help()
	case "-n":
		New()
	case "-s":
		Serve("yb.toml")
	case "-v":
		Version()
	default:
		fmt.Println("unknown command")
	}
}

func Help() {
	help_message := string(DEFAULT_HELP)
	fmt.Println(help_message)
}

func New() {
	fsutil.MustWrite("yb.yaml", DEFAULT_CONFIG)
	fsutil.MustWrite("blog/intro.md", DEFAULT_BLOG)
	fsutil.MustWrite("blog/private.md", DEFAULT_PRIVATE)
	fsutil.MustWrite("blog/keyword.md", DEFAULT_KEYWORD)
	fsutil.MustWrite("blog/hide.md", DEFAULT_HIDE)
	fsutil.MustWrite("./template.html", DEFAULT_TEMPLATE)
	fsutil.MustWrite("blog/favicon.ico", DEFAULT_FAVICON)
	fsutil.MustWrite("blog/vue.js", DEFAULT_VUE_JS)
}

func Serve(configPath string) {
	app := internal.NewApp(configPath)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func Version() {
	fmt.Println(string(DEFAULT_VERSION))
}
