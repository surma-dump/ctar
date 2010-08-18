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
	"crypto/aes"
	"crypto/block"
)

const (
	READ_ALL = -1
	VERSION = "1.1"
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
		fmt.Printf("ctar version %s\n"+
		"by Alexander \"Surma\"Surma\n\n", VERSION)
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

	rc, e = checkFlagValidity(*h, *c, *x)
	if e != nil {
		return
	}

	if pass == "" {
		pass = ReadPassword()
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

	v := vector.StringVector(make([]string, 1, 20))
	for i := range in {
		v.Push(i)
	}

	return v[1:v.Len()]
}

func TraverseFileTree(path string) ([]string, os.Error) {
	l := vector.StringVector(make([]string, 1, 20))
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

	return l, nil

}

func TraverseFileTreeFiltered(path string) ([]string, os.Error) {
	list, e := TraverseFileTree(path)
	if e != nil {
		return nil, e
	}

	v := vector.StringVector(make([]string, 1, 20))
	for _, l := range list {
		if len(l) != 0 {
			v.Push(l)
		}
	}

	return v[1:v.Len()], nil

}

func AddFileToTar(tw *tar.Writer, filepath string) os.Error {
	d, e := os.Stat(filepath)
	if e != nil {
		return e
	}

	h := tar.Header{
		Name:  filepath,
		Mode:  int64(d.Mode),
		Uid:   d.Uid,
		Gid:   d.Gid,
		Size:  int64(d.Size),
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

		_, e = io.Copy(tw, f)
		f.Close()
		if e != nil {
			return e
		}
	}
	return nil
}

func TarDirectory(path string, w io.Writer) os.Error {
	rootdir, e := os.Open(path, os.O_RDONLY, 0)
	if e != nil {
		return e
	}
	defer rootdir.Close()

	filelist, e := TraverseFileTreeFiltered(path)
	if e != nil {
		return e
	}

	tw := tar.NewWriter(w)

	for _, filepath := range filelist {
		if filepath != "." && filepath != ".." {
			fmt.Fprintf(os.Stderr, "Packing: %s\n", filepath)
			e := AddFileToTar(tw, filepath)
			if e != nil {
				return e
			}
		} else {
			fmt.Fprintf(os.Stderr, "Skipping: %s\n", filepath)
		}
	}
	tw.Flush()
	return nil
}

func ExtractFileFromTar(hdr *tar.Header, r io.Reader) os.Error {
	if hdr.Typeflag == tar.TypeDir {
		e := os.Mkdir("./"+hdr.Name, uint32(hdr.Mode))
		if e != nil {
			return e
		}
		e = os.Chown("./"+hdr.Name, int(hdr.Uid), int(hdr.Gid))
		return e
	} else {
		f, e := os.Open("./"+hdr.Name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uint32(hdr.Mode))
		if e != nil {
			return e
		}
		defer f.Close()

		_ = os.Chown("./"+hdr.Name, int(hdr.Uid), int(hdr.Gid))

		_, e = io.Copy(f, r)
		if e != nil {
			return e
		}
		return nil
	}
	return nil // Never reached
}

func UntarArchive(r io.Reader) os.Error {
	tr := tar.NewReader(r)

	for hdr, e := tr.Next(); hdr != nil; hdr, e = tr.Next() {
		if e != nil {
			return e
		}

		fmt.Fprintf(os.Stderr, "Unpacking: %s\n", hdr.Name)
		e = ExtractFileFromTar(hdr, tr)
		if e != nil {
			fmt.Fprintf(os.Stderr, "\tFailed! %s\n", e.String())
		}
	}
	return nil
}

func SetupEncrypt(w io.Writer, key []byte, iv []byte) (io.Writer, os.Error) {
	c, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return block.NewCBCEncrypter(c, iv, w), nil
}

func SetupDecrypt(r io.Reader, key []byte, iv []byte) (io.Reader, os.Error) {
	c, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return block.NewCBCDecrypter(c, iv, r), nil
}

func CheckMagicNumber(r io.Reader) os.Error {
	var b [4]byte

	_, e := r.Read(&b)
	if e != nil {
		return e
	}

	if string([]byte(&b)) != "CTAR" {
		return os.NewError("MagicNumber doesn't match. (Wrong password?)")
	}
	return nil
}

func main() {
	fio, create, password, e := setup()
	surmc.PanicOnError(e, "epd failed")

	key, e := surmc.SHA256hash([]byte(password))
	surmc.PanicOnError(e, "Calculating password hash failed")
	iv, e := surmc.MD5hash([]byte(password))
	surmc.PanicOnError(e, "Calculating password hash failed")

	if create {
		cio, e := SetupEncrypt(fio, key, iv)
		surmc.PanicOnError(e, "Encryption system failed")
		fmt.Fprintf(cio, "CTAR")
		for _, dir := range flag.Args() {
			e = TarDirectory(dir, cio)
			surmc.PanicOnError(e, "Taring failed")
		}
	} else {
		cio, e := SetupDecrypt(fio, key, iv)
		surmc.PanicOnError(e, "Decryption system failed")
		e = CheckMagicNumber(cio)
		surmc.PanicOnError(e, "Not a valid CTAR")
		e = UntarArchive(cio)
		surmc.PanicOnError(e, "Extracting failed")
	}
}
