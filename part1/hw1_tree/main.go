package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

const filePrefix = "├───"
const filePrefixLast = "└───"
const subPrefix = "│	"
const subPrefixLast = "	"

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

func dirTree(out io.Writer, path string, printFiles bool) error {
	root := FileNode{
		Path: path,
	}
	err := loadChildren(&root, printFiles)
	if err != nil {
		return err
	}
	printChildren(out, &root)
	return nil
}

func printChildren(out io.Writer, parent *FileNode) {
	for _, node := range parent.Children {
		prefix := getPrefix(node)
		if node.FileInfo.IsDir() {
			fmt.Fprintln(out, prefix+node.FileInfo.Name())
		} else {
			fmt.Fprintf(out, prefix+node.FileInfo.Name()+" (%v)\n", GetSizeStr(node.FileInfo))
		}
		printChildren(out, node)
	}
}

func getPrefix(node *FileNode) (prefix string) {
	prefix = ""
	if node.IsLast {
		prefix += filePrefixLast
	} else {
		prefix += filePrefix
	}
	current := node.Parent
	for true {
		if current == nil || current.FileInfo == nil {
			break
		}
		if current.IsLast {
			prefix = subPrefixLast + prefix
		} else {
			prefix = subPrefix + prefix
		}
		current = current.Parent
	}
	return
}

func loadChildren(parent *FileNode, printFiles bool) error {
	files, err := ioutil.ReadDir(parent.Path)
	if err != nil {
		return err
	}
	parent.Children = make([]*FileNode, 0)
	for _, file := range files {
		if !file.IsDir() && !printFiles {
			continue
		}
		node := FileNode{
			Path:     parent.Path + string(os.PathSeparator) + file.Name(),
			FileInfo: file,
			Parent:   parent,
		}
		parent.Children = append(parent.Children, &node)
		if file.IsDir() {
			err = loadChildren(&node, printFiles)
			if err != nil {
				return err
			}
		}
	}
	if len(parent.Children) > 0 {
		parent.SortChildren()
		parent.Children[len(parent.Children)-1].IsLast = true
	}
	return nil
}

//SortChildren - sort children alphabetically
func (p *FileNode) SortChildren() {
	fileList := FileList(p.Children)
	sort.Sort(fileList)
	p.Children = []*FileNode(fileList)
}

//FileNode - node of file tree, can contains children (or not)
type FileNode struct {
	FileInfo os.FileInfo
	Path     string
	Children []*FileNode
	Parent   *FileNode
	IsLast   bool
}

//FileList - array of FileInfo for sort
type FileList []*FileNode

func (p FileList) Len() int           { return len(p) }
func (p FileList) Less(i, j int) bool { return p[i].FileInfo.Name() < p[j].FileInfo.Name() }
func (p FileList) Swap(i, j int)      { *p[i], *p[j] = *p[j], *p[i] }

//GetSizeStr - get size of file/dir in bytes. If 0 - returns "empty"
func GetSizeStr(file os.FileInfo) string {
	size := file.Size()
	if size > 0 {
		return strconv.FormatInt(size, 10) + "b"
	}
	return "empty"
}
