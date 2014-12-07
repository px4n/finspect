package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func scan(path string, f os.FileInfo, err error) error {
	fmt.Printf("Scanned: %s\n", path)
	return nil
}

func main() {
	flag.Parse()
	root := flag.Arg(0)
	err := filepath.Walk(root, scan)
	fmt.Printf("filepath.Walk() returned %v\n", err)
}
