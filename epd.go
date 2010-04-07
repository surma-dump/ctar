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
	fmt.Printf("Password: ")
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

func setup() (fio io.ReadWriter, rc bool, e os.Error) {
	fname := flag.String("f", "", "Use a file instead of stdin/stdout")
	c, x := flag.Bool("c", false, "Create an archive"), flag.Bool("x", false, "Extract an archive")
	h := flag.Bool("h", false, "Display this help")
	flag.Parse()

	rc, e = checkFlagValidity(*h, *c, *x)
	if e != nil {
		return
	}

	if len(*fname) > 0 && rc{
		fio, e = os.Open(*fname, os.O_WRONLY | os.O_CREAT, 0600)
	} else if len(*fname) > 0 && !rc {
		fio, e = os.Open(*fname, os.O_RDONLY, 0)
	} else {
		fio = io.ReadWriter(bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout)))
	}

	return
}

func main() {
	fio, create, e := setup()
	surmc.PanicOnError(e, "epd failed")

	password, e := surmc.Sha256hash([]byte(ReadPassword()))
	surmc.PanicOnError(e, "Obtaining password hash failed")


}
