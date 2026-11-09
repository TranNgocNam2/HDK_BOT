package main

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

func main() {
	r, err := zip.OpenReader("templates/1.docx")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				panic(err)
			}
			b, _ := io.ReadAll(rc)
			rc.Close()
			s := string(b)
			idx := strings.Index(s, "F06F")
			if idx != -1 {
				fmt.Println(s[idx-100:idx+200])
			}
		}
	}
}
