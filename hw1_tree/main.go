package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// MyDirDict type for saving file names in directory
type MyDirDict map[string][]string

// AppendToDict appends files or directory with files in MyDirDict
func (dictDir MyDirDict) AppendToDict(path string) {
	files, err := ioutil.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			secondFiles, secondErr := ioutil.ReadDir(path + string(os.PathSeparator) + file.Name())

			if secondErr != nil {
				log.Fatal(secondErr)
			}

			dictDir[file.Name()] = make([]string, len(secondFiles))

			for ind, secondFile := range secondFiles {
				dictDir[file.Name()][ind] = secondFile.Name()
			}
		}
	}

}

// GetDirsPathes gets pathes for all directories
func GetDirsPathes(path string) []string {
	var dirsPathes []string
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {
		if file.IsDir() {
			dirsPathes = append(dirsPathes, path+string(os.PathSeparator)+file.Name())
		}
	}
	return dirsPathes
}

// FullFillDict fulfills direcories dict
func FullFillDict(path string) MyDirDict {

	dirsD := MyDirDict(make(map[string][]string))

	var currPath string = path
	dirsLs := GetDirsPathes(currPath)

	dirsD.AppendToDict(path)

	for len(dirsLs) != 0 {
		newDirsLs := []string{}

		for _, onePath := range dirsLs {
			newDirsLs = append(newDirsLs, GetDirsPathes(onePath)...)
			dirsD.AppendToDict(onePath)
		}

		dirsLs = newDirsLs
	}
	return dirsD
}

// PaintDirsTree paints directory tree
func PaintDirsTree(path string, prefix string, printFiles bool) string {

	printFilesMode := printFiles
	var dirsLs []string
	var str string
	sizesDict := make(map[string]string)

	allFiles := FullFillDict(path)

	files, _ := ioutil.ReadDir(path)

	for _, file := range files {

		var fileSize string

		if !file.IsDir() {
			size := file.Size()
			if size == 0 {
				fileSize = " (empty)"
			} else {
				fileSize = fmt.Sprintf(" (%vb)", size)
			}

		} else {
			fileSize = ""
		}

		if printFiles {
			dirsLs = append(dirsLs, file.Name())
		} else {

			if file.IsDir() {
				dirsLs = append(dirsLs, file.Name())
			} else {
				continue
			}

		}
		sizesDict[file.Name()] = fileSize

	}

	for _, dir := range dirsLs {

		var prefixToWrite string = prefix

		str += prefixToWrite

		if dirsLs[len(dirsLs)-1] == dir {
			str += `└───` + dir + sizesDict[dir] + "\n"
		} else {
			str += `├───` + dir + sizesDict[dir] + "\n"
		}

		_, dirExists := allFiles[dir]

		currPath := path + string(os.PathSeparator) + dir

		if dirsLs[len(dirsLs)-1] == dir && dirExists {
			prefixToWrite += "\t"
			str += PaintDirsTree(currPath, prefixToWrite, printFilesMode)
		} else if dirExists && !(dirsLs[len(dirsLs)-1] == dir) {
			prefixToWrite += "│\t"
			str += PaintDirsTree(currPath, prefixToWrite, printFilesMode)
		}

	}
	return str

}

func dirTree(output io.Writer, path string, showFiles bool) error {
	var FilesShow bool = showFiles
	var pathToGet string = path
	var resultTree string = PaintDirsTree(pathToGet, "", FilesShow)

	_, err := ioutil.ReadDir(path)

	fmt.Fprint(output, resultTree)

	return err
}

func main() {
	out := os.Stdout

	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}

	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)

	if err != nil {
		panic(err.Error())
	}
}
