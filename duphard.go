//
// Copyright 2015 Marios Andreopoulos
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

/*
Command godedup finds duplicates files and turns them to hardlinks if possible.

*Be careful*, godedup is dump. It won't check if the files reside on the same filesystem.
You are supposed to run it on folders and files in the same filesystem.

Usage, test run, only report what we find:

    godedup <DIR> [<FILE> <DIR>...]

Usage, full run, replace duplicates with hard links:

    godedup -d=0 <DIR> [<FILE> <DIR>...]
*/
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
)

var dryrun = true

func init() {
	flag.BoolVar(&dryrun, "d", true, "dry run, do not make any changes to the filesystem")
	flag.Parse()
}

func main() {
	// a list containing the regular files we found, mapped by size
	sizeList := make(map[int64]map[string]bool)

	// check is used for the filepathWalk function
	// if it finds a regular file, it adds it to sizeList
	check := func(path string, f os.FileInfo, err error) error {
		if f, err := os.Stat(path); err == nil { // catch all erros, not only IsNotExist
			if f.Mode().IsRegular() {
				if _, exists := sizeList[f.Size()]; !exists {
					sizeList[f.Size()] = make(map[string]bool)
					sizeList[f.Size()][path] = true
				} else {
					// be a bit smart and check if we have hard links
					// thus no need to run checksum on them (makes 2nd run *fast*)
					for k := range sizeList[f.Size()] {
						other, err := os.Stat(k)
						if err != nil {
							fmt.Println(err)
						}
						if !os.SameFile(f, other) {
							sizeList[f.Size()][path] = true
						}
					}
				}
			}
		}
		return nil
	}

	// recursively walk the given files/paths and add regular files to sizeList
	for _, v := range flag.Args() {
		_ = filepath.Walk(v, check)
	}

	// if a size key in sizeList has only one element, it isn't duplicate, remove it
	for k, v := range sizeList {
		if len(v) == 1 {
			delete(sizeList, k)
		}
	}

	if len(sizeList) == 0 {
		fmt.Println("No duplicates found.")
		os.Exit(0)
	}

	fmt.Printf("Found %d candidate groups with common size.\n", len(sizeList))
	// for _, v := range sizeList {
	// 	fmt.Println(v)
	// }

	// a list containing the regular files we found that had common size, mapped by checksum
	checksumList := make(map[string][]string)
	// checksum calculates the md5 hash for the given file and adds it to the checksumList
	checksum := func(path string) {
		file, err := os.Open(path)
		if err != nil {
			fmt.Println(path, err.Error)
			return
		}
		defer file.Close()

		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			fmt.Println(path, err.Error)
			return
		}

		var res []byte
		c := string(hash.Sum(res))
		if _, exists := checksumList[c]; !exists {
			checksumList[c] = make([]string, 0)
		}
		checksumList[c] = append(checksumList[c], path)
	}

	// run checksum for each file we think may be a duplicate
	for _, v := range sizeList {
		for k2 := range v {
			checksum(k2)
		}
	}

	// if a checksum key in checksumList has only one element, it isn't duplicate
	for k, v := range checksumList {
		if len(v) == 1 {
			delete(checksumList, k)
		}
	}

	if len(checksumList) == 0 {
		fmt.Println("No duplicates found.")
		os.Exit(0)
	}

	fmt.Printf("Found %d candidate groups with same checksum.\n", len(checksumList))

	var spaceFreed int64
	// For each group try to check if files are already hard links. This isn't very
	// smart, a group of three files could easily fool it but it should be adequate.
	// Even if we remove this block, the result should be the same.
	// Also we already have a very good hard link check in the file size stage.
	for k, v := range checksumList {
		src, err := os.Stat(v[0])
		if err != nil {
			fmt.Println(err)
		}

		for i := len(v) - 1; i > 0; i-- {
			dst, err := os.Stat(v[i])
			if err != nil {
				fmt.Println(err)
			}
			if os.SameFile(src, dst) {
				if len(v) > 2 {
					v = v[:len(v)-1]
				} else {
					delete(checksumList, k)
				}
			} else {
				spaceFreed += dst.Size()
			}
		}
	}

	fmt.Printf("Found %d groups that are duplicates and probably not hard links:\n", len(checksumList))
	for _, v := range checksumList {
		fmt.Println(v)
	}
	fmt.Printf("Space freed will be about %s.\n", humanize.Bytes(uint64(spaceFreed)))

	if !dryrun {
		fmt.Println("No dry-run. Proceeding to filesystem modifications.")

		for _, v := range checksumList {
			_, err := os.Stat(v[0])
			if err == nil {
				for i := len(v) - 1; i > 0; i-- {
					fmt.Println("deleting", v[i])
					err := os.Remove(v[i])
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					fmt.Println("linking", v[0], "to", v[i])
					err = os.Link(v[0], v[i])
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}
			}
		}
		fmt.Printf("Space freed is about %s.\n", humanize.Bytes(uint64(spaceFreed)))
	}
}
