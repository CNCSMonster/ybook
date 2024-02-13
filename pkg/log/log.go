//go:build !release

package log

import stdlog "log"

func Printf(format string, v ...any) {
	stdlog.Printf(format, v...)
}
func Println(v ...any) {
	stdlog.Println(v...)
}
func Fatal(v ...any) {
	stdlog.Fatal(v...)
}
