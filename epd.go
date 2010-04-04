package main

import (
	"fmt"
	"flag"
	"io"
	"os"
	"surmc"
	"bufio"
)

func ReadPassword() string {
	passw := make([]byte, 128)
	fmt.Printf("Password: ")
	fmt.Printf("\033[40;30m")
	os.Stdin.Read(passw)
	fmt.Printf("\033[m")
	return string(passw)
}

func setup() (fio io.ReadWriter, rc bool, password string, e os.Error) {
	fname := flag.String("f", "", "Use a file instead of stdin/stdout")
	c, x := flag.Bool("c", false, "Create an archive"), flag.Bool("x", false, "Extract an archive")
	h := flag.Bool("h", false, "Display this help")
	flag.Parse()

	if *h {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *c == *x {
		e = os.NewError("Pass either -c or -x")
		return
	}
	rc = !*x

	if rc && len(flag.Arg(0)) == 0 {
		e = os.NewError("No directory specified")
		return
	}

	if len(*fname) > 0 && rc{
		fio, e = os.Open(*fname, os.O_WRONLY | os.O_CREAT, 0600)
	} else if len(*fname) > 0 && !rc {
		fio, e = os.Open(*fname, os.O_RDONLY, 0)
	} else {
		fio = io.ReadWriter(bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout)))
	}

	password = ReadPassword()
	return
}

func main() {
//	fio, create, password, e := setup()
	_, _, _, e := setup()
	surmc.PanicOnError(e, "epd failed")


}
