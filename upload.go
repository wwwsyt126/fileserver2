package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	ospath "path"
	"regexp"
	"strings"
)

//concern for name collision, filename need to be unique
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	targetFile := r.FormValue("file")
	ip := strings.Split(r.RemoteAddr, ":")[0]
	uri := strings.NewReplacer("/uploadapi", "").Replace(r.RequestURI)
	fileurl := ospath.Join(path, uri, targetFile)

	var validPath = regexp.MustCompile(`.*\.\.\/.*`)
	if validPath.MatchString(targetFile) {
		fmt.Fprintf(w, "file path should not contain the two dot\n")
		return
	}

	delete := r.FormValue("delete")
	if delete == "yes" {
		if targetFile == "" {
			fmt.Fprintln(w, "filename not specified")
			return
		}
		err := os.RemoveAll(fileurl)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		fmt.Fprintf(w, "file: %v deleted\n", targetFile)
		log.Println(ip, uri, targetFile, "deleted")
		return
	}

	truncate := r.FormValue("truncate")

	flags := os.O_APPEND | os.O_WRONLY | os.O_CREATE
	if truncate == "yes" {
		flags = flags | os.O_TRUNC
	}

	data := r.FormValue("data")
	if data != "" {
		if targetFile == "" {
			fmt.Fprintln(w, "filename not specified")
			return
		}
		d := strings.NewReader(data)

		dir := ospath.Dir(fileurl)
		err := os.MkdirAll(dir, 755)
		if err != nil {
			fmt.Fprintln(w, "mkdir error %v\n", err)
			return
		}

		out, err := os.OpenFile(fileurl, flags, 0644)
		if err != nil {
			fmt.Fprintf(w, "Unable to create the file for writing")
			return
		}
		defer out.Close()

		n, err := io.Copy(out, d)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		var note string
		if truncate == "yes" {
			note = "( truncated )"
		}
		fmt.Fprintf(w, "Files uploaded successfully : %v %v bytes %v\n", targetFile, n, note)
		log.Println(ip, uri, targetFile, "created")
		return
	}

	// no bigger than 10G
	err := r.ParseMultipartForm(10000000000)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	formdata := r.MultipartForm

	// multipart parameter "file" need to specified when upload
	files := formdata.File["file"]

	if len(files) == 0 {
		fmt.Fprintf(w, "need to provide file(multipart form) or data\n")
		return
	}

	for i, _ := range files {
		file, err := files[i].Open()
		if err != nil {
			fmt.Fprintln(w, err)
			continue
		}
		defer file.Close()

		filename := files[i].Filename

		var f string
		if targetFile == "" {
			f = filename
		} else {
			f = targetFile
		}

		var fileurl string
		if ospath.Dir(targetFile) == "." {
			fileurl = ospath.Join(path, uri, f)
		} else {
			fileurl = ospath.Join(path, f)
		}

		dir := ospath.Dir(fileurl)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Fprintln(w, "mkdir error %v\n", err)
			continue
		}

		out, err := os.OpenFile(fileurl, flags, 0644)
		if err != nil {
			fmt.Fprintf(w, "Unable to create the file for writing")
			continue
		}
		defer out.Close()

		n, err := io.Copy(out, file)
		if err != nil {
			fmt.Fprintln(w, err)
			continue
		}

		var note string
		if truncate == "yes" {
			note = "( truncated )"
		}
		fmt.Fprintf(w, "Files uploaded successfully : %v %v bytes %v\n", f, n, note)
		log.Println(ip, uri, f, "created")
	}
}
