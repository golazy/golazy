//go:build exclude

package main

import "github.com/golazy/golazy/apptemplate"

func main() {

	fs := apptemplate.MemFS{
		"file1.txt":      "content1",
		"file2.txt":      "content2",
		"dir1/file3.txt": "content3",
		"dir1/file4.txt": "content4",
	}

	template := &apptemplate.Template{Name: "test"}
	template.Copy(fs)

	err := template.Run(apptemplate.RunOpts{
		Dest: "/tmp/asdf",
	})
	if err != nil {
		panic(err)
	}
}
