package main

import (
	"fmt"
	"flag"
	"io"
	"os"
	"surmc"
	"bufio"
	"archive/tar"
)

const (
	READ_ALL = -1
)

func ReadPassword() string {
	fmt.Fprintf(os.Stderr, "Password: ")
	surmc.SetAttr(surmc.ATTR_FG_BLACK, surmc.ATTR_BG_BLACK)
	passw, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	surmc.SetAttr(surmc.ATTR_RESET)

	return passw[0:len(passw)-1]
}

func checkFlagValidity(h, c, x bool) (rc bool, e os.Error) {
	if h {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if c == x {
		e = os.NewError("Pass either -c or -x")
		return
	}
	rc = !x

	if rc && len(flag.Arg(0)) == 0 {
		e = os.NewError("No directory specified")
		return
	}
	return
}

func setup() (fio io.ReadWriter, rc bool, pass string, e os.Error) {
	fname := flag.String("f", "", "Use a file instead of stdin/stdout")
	c, x := flag.Bool("c", false, "Create an archive"), flag.Bool("x", false, "Extract an archive")
	h := flag.Bool("h", false, "Display this help")
	p := flag.String("p", "", "Use password")
	flag.Parse()
	pass = *p

	if pass == "" {
		fmt.Fprintf(os.Stderr, "WARNING: No password (-p) specified\n")
	}

	rc, e = checkFlagValidity(*h, *c, *x)
	if e != nil {
		return
	}

	if len(*fname) > 0 && rc{
		fio, e = os.Open(*fname, os.O_WRONLY | os.O_CREAT, 0600)
	} else if len(*fname) > 0 && !rc {
		fio, e = os.Open(*fname, os.O_RDONLY, 0)
	} else {
		fio = surmc.Stdinout
	}

	return
}

func TraverseFileTree(path string) ([]string, os.Error) {
}

func TarDirectory(path string, w io.Writer) os.Error {
	dir, e := os.Open(path, os.O_RDONLY, 0)
	if e != nil {
		return e
	}

	list, e := dir.Readdirnames(READ_ALL)
	if e != nil {
		return e
	}

	for _,l := range list {
		fmt.Fprintf(os.Stderr, "Packing: %s\n", l)
	}
	_ = tar.NewWriter(w)
	return nil
}

func main() {
	fio, create, password, e := setup()
	surmc.PanicOnError(e, "epd failed")

	key, e := surmc.Sha256hash([]byte(password))
	surmc.PanicOnError(e, "Calculating password hash failed")

	_ = key
	if create {
		e = TarDirectory(flag.Arg(0), fio)
		surmc.PanicOnError(e, "Opening target dir")
	} else {
		fmt.Print("Duh\n")
	}
}
