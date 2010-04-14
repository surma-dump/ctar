package main

import (
	"fmt"
	"flag"
	"io"
	"os"
	"surmc"
	"bufio"
	"archive/tar"
	"container/vector"
)

const (
	READ_ALL = -1
)

func ReadPassword() string {
	fmt.Fprintf(os.Stderr, "Password: ")
	surmc.SetAttr(surmc.ATTR_FG_BLACK, surmc.ATTR_BG_BLACK)
	passw, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	surmc.SetAttr(surmc.ATTR_RESET)

	return passw[0 : len(passw)-1]
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

	if len(*fname) > 0 && rc {
		fio, e = os.Open(*fname, os.O_WRONLY|os.O_CREAT|os.O_TRUNC, 0600)
	} else if len(*fname) > 0 && !rc {
		fio, e = os.Open(*fname, os.O_RDONLY, 0)
	} else {
		fio = surmc.Stdinout
	}

	return
}

func IsDirectory(path string) (bool, os.Error) {
	d, e := os.Stat(path)
	if e != nil {
		return false, e
	}

	return d.IsDirectory(), e
}

func GetDirectoryContent(path string) ([]string, os.Error) {
	f, e := os.Open(path, os.O_RDONLY, 0)
	if e != nil {
		return nil, e
	}

	l, e := f.Readdirnames(READ_ALL)
	f.Close()
	return l, e
}

func FilterEmptyStrings(out chan<- string, in <-chan string) {
	for i := range in {
		if len(i) != 0 {
			out <- i
		}
	}
	close(out)
	return
}

func ChannelToSliceString(in <-chan string) []string {
	v := vector.StringVector(make([]string, 1))

	for i := range in {
		v.Push(i)
	}

	r := v.Data()
	return r[1:len(r)]
}

func TraverseFileTree(path string) ([]string, os.Error) {
	l := vector.StringVector(make([]string, 1))
	l.Push(path)
	d, e := IsDirectory(path)
	if e != nil {
		return nil, e
	}

	if d {
		c, e := GetDirectoryContent(path)
		if e != nil {
			return nil, e
		}

		for _, file := range c {
			s, e := TraverseFileTree(path + "/" + file)
			v := vector.StringVector(s)
			if e != nil {
				return nil, e
			}

			l.AppendVector(&v)
		}
	}

	list := l.Iter()
	filt := make(chan string)
	go FilterEmptyStrings(filt, list)
	ret := ChannelToSliceString(filt)
	return ret, nil

}

func AddFileToTar(tw *tar.Writer, filepath string) os.Error {
	d, e := os.Stat(filepath)
	if e != nil {
		return e
	}

	h := tar.Header{
		Name:  filepath,
		Mode:  int64(d.Mode),
		Uid:   int64(d.Uid),
		Gid:   int64(d.Gid),
		Atime: int64(d.Atime_ns / 1e9),
		Ctime: int64(d.Ctime_ns / 1e9),
		Mtime: int64(d.Mtime_ns / 1e9),
	}

	if d.IsDirectory() {
		h.Typeflag = tar.TypeDir
	} else if d.IsRegular() {
		h.Typeflag = tar.TypeReg
	} else {
		fmt.Fprintf(os.Stderr, "Skipped non-regular file: \"%s\"\n", filepath)
		return nil
	}
	tw.WriteHeader(&h)

	if !d.IsDirectory() {
		f, e := os.Open(filepath, os.O_RDONLY, 0)
		if e != nil {
			return e
		}

		io.Copy(tw, f)
		f.Close()
	}
	return nil
}

func TarDirectory(path string, w io.Writer) os.Error {
	rootdir, e := os.Open(path, os.O_RDONLY, 0)
	if e != nil {
		return e
	}
	defer rootdir.Close()

	filelist, e := TraverseFileTree(path)
	if e != nil {
		return e
	}

	tw := tar.NewWriter(w)

	for _, filepath := range filelist {
		fmt.Fprintf(os.Stderr, "Packing: %s\n", filepath)
		e := AddFileToTar(tw, filepath)
		if e != nil {
			return e
		}
	}
	return nil
}

func main() {
	fio, create, password, e := setup()
	surmc.PanicOnError(e, "epd failed")

	key, e := surmc.Sha256hash([]byte(password))
	surmc.PanicOnError(e, "Calculating password hash failed")

	_ = key
	if create {
		for _, dir := range flag.Args() {
			e = TarDirectory(dir, fio)
			surmc.PanicOnError(e, "Taring failed")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Not implemented\n")
	}
}
