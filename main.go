// Grabs a folder and generates a go file with a map
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cheggaaa/pb"
	"github.com/gohxs/folder2go/assets"
)

var (
	flagHandler bool
	flagNoBak   bool
)

func main() {

	flag.BoolVar(&flagHandler, "handler", false, "Generates http handlerFunc")
	flag.BoolVar(&flagNoBak, "nobackup", false, "Does not write a .bak if .go file exists")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "This will create a folder named by [pkgname] and add the generated file")
		fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[folder] [pkgname] <destination>")
		flag.PrintDefaults()
		return
	}

	var folder, err = filepath.Abs(flag.Args()[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error on dir:", err)
		return
	}
	var pkg = filepath.Base(flag.Args()[1]) // Remove trailing '/'
	dst := pkg
	if flag.NArg() >= 3 {
		dst = flag.Args()[2]
	}
	// Data where it will be transformed to a Go file
	var data = map[string]string{}

	_, err = os.Stat(folder)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening dir", folder, err)
		return
	}

	// For every file
	filepath.Walk(folder, func(fpath string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil // Continue
		}
		// Get ABSPATH
		absFolder, err := filepath.Abs(folder)
		if err != nil {
			return err
		}
		absPath, err := filepath.Abs(fpath)
		if err != nil {
			return err
		}

		kfname := absPath[len(absFolder)+1:] // remove slash?
		buf := bytes.NewBuffer(nil)
		writeHexFile(buf, absFolder, kfname, f)

		data[kfname] = buf.String()

		return nil
	})

	targetFile := fmt.Sprintf("%s/%s.go", dst, pkg)

	// Check if file exists
	if !flagNoBak {
		_, err = os.Stat(targetFile)
		if err == nil || !os.IsNotExist(err) { // File exists
			err := os.Rename(targetFile, targetFile+".bak") // can fail
			if err != nil {
				panic(err)
			}
		}
	}

	// Write golang file
	os.MkdirAll(dst, os.FileMode(0755))
	f, err := os.OpenFile(fmt.Sprintf("%s/%s.go", dst, pkg), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl, err := template.New("").Parse(string(assets.Data["package.go.tmpl"]))
	if err != nil {
		panic(err)
	}
	tmpl.Execute(f, map[string]interface{}{
		"Package": pkg,
		"Data":    data,
		"Handler": flagHandler,
	})

}

func writeHexFile(w io.Writer, curDir, fname string, f os.FileInfo) error {
	// Send file through channel?
	// Open file
	fin, err := os.Open(filepath.Join(curDir, fname))
	if err != nil {
		log.Fatal(err)
		return err
	}

	// File Processor
	buf := make([]byte, 4096)
	fmt.Fprintln(os.Stderr, "Processing file:", fname)
	bar := pb.New(int(f.Size()))
	bar.Output = os.Stderr
	bar.Start()

	totN := 0
	for {
		n, err := fin.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, "Err:", err)
			break
		}
		bar.Add(n)
		fmt.Fprintf(w, "\t\t\t")
		for i, v := range buf[:n] {
			if i != 0 && (i+totN)%(80/4) == 0 {
				fmt.Fprintf(w, "\n\t\t\t")
			}
			fmt.Fprintf(w, "0x%02X, ", v)
		}
		if err == io.EOF {
			break
		}
		totN += n
	}
	bar.Finish()
	return nil

}
