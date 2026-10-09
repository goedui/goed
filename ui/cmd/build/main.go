// 编译应用并把同目录的 icon.ico 写进 exe。
//
//	go run ./ui/cmd/build -ldflag "-s -w -H windowsgui" -o imgen.exe ./imgen
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	out := flag.String("o", "", "输出的 exe 路径")
	ldflag := flag.String("ldflag", "", "传给 go build -ldflags")
	ldflags := flag.String("ldflags", "", "同 -ldflag")
	flag.Parse()
	pkg := "."
	if flag.NArg() > 0 {
		pkg = flag.Arg(0)
	}
	if *out == "" {
		*out = defaultOut(pkg)
	}
	// Windows 默认是 PIE。BeginUpdateResource 无法给动态基址的 exe 追加图标。
	args := []string{"build", "-buildmode=exe", "-o", *out}
	if flags := firstNonEmpty(*ldflags, *ldflag); flags != "" {
		args = append(args, "-ldflags", flags)
	}
	args = append(args, pkg)
	cmd := exec.Command("go", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
	ico := filepath.Join(pkgDir(pkg), "icon.ico")
	if _, err := os.Stat(ico); err != nil {
		return
	}
	if err := stamp(*out, ico); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func pkgDir(pkg string) string {
	if pkg == "" || pkg == "." {
		return "."
	}
	if strings.Contains(pkg, ":") || strings.HasPrefix(pkg, ".") || strings.HasPrefix(pkg, "/") || strings.HasPrefix(pkg, `\`) {
		return pkg
	}
	return pkg
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func defaultOut(pkg string) string {
	name := filepath.Base(pkgDir(pkg))
	if name == "." || name == "" {
		name = "app"
	}
	return name + ".exe"
}
