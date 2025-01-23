package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var mapCount int

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file...>")
		os.Exit(1)
	}
	files := os.Args[1:]

	for _, file := range files {
		fixTarget(file)
	}
	fmt.Println("potentially fixed", mapCount, "maps")
}

func fixTarget(file string) {
	// Trim out anything with an extension, as we have some random zip files in the target...
	if strings.Contains(file, ".") {
		return
	}

	stat, err := os.Stat(file)
	if err != nil {
		fmt.Println(err)
	}
	if stat.IsDir() {
		entries, err := os.ReadDir(file)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, entry := range entries {
			fixTarget(filepath.Join(file, entry.Name()))
		}
	} else {
		fixMap(file)
	}
}

func fixMap(name string) {
	fmt.Println("processing", name)
	mapCount++
	// Let's just brute-force it.
	data, err := os.ReadFile(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	lines := strings.Split(string(data), "\n")
	var foundArch bool
	var foundEnd bool
	var fixedInvisible bool
	var processMore bool
	var processMoreStart int
	var archEnd int
	var changed bool
	var maxX, maxY int
	var hasW, hasH bool
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			continue
		}
		// Read in any x/y lines so we can generate width/height as needed if missing...
		if line[0] == 'x' {
			xs := line[2:]
			x, err := strconv.Atoi(xs)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if x > maxX {
				maxX = x
			}
		} else if line[0] == 'y' {
			ys := line[2:]
			y, err := strconv.Atoi(ys)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if y > maxY {
				maxY = y
			}
		}
		// Discover map definition start
		if line == "arch map" {
			fmt.Println("found arch map", i)
			foundArch = true
			continue
		}
		// Discover map end start (-1 position used for appending width/height if missing)
		if line == "end" && foundArch && !foundEnd {
			fmt.Println("found end", i)
			foundEnd = true
			archEnd = i
			continue
		}
		// Remove any "More" entries, as multi-tile entries are not stored per-tile anymore.
		if processMore {
			if lines[i] == "end" {
				processMore = false
				fmt.Println("removing multi-tile", processMoreStart, i)
				lines = append(lines[:processMoreStart], lines[i+1:]...)
				i = processMoreStart - 1
				changed = true
			}
			continue
		}
		// Process things within the map arch entry.
		if foundArch && !foundEnd {
			if line[0] == 'x' {
				xs := line[2:]
				x, err := strconv.Atoi(xs)
				if err != nil {
					fmt.Println(err)
					continue
				}
				maxX = x
				lines[i] = "width" + line[1:]
				changed = true
				fmt.Println("Patched", line, "to", lines[i])
				hasW = true
			} else if line[0] == 'y' {
				ys := line[2:]
				y, err := strconv.Atoi(ys)
				if err != nil {
					fmt.Println(err)
					continue
				}
				maxY = y
				lines[i] = "height" + line[1:]
				changed = true
				fmt.Println("Patched", line, "to", lines[i])
				hasH = true
			} else if strings.HasPrefix(line, "invisible") {
				lines[i] = "darkness" + line[9:]
				fixedInvisible = true
				changed = true
				fmt.Println("Patched", line, "to", lines[i])
			} else if strings.HasPrefix(line, "width") {
				fmt.Println("found width")
				hasW = true
				xs := line[6:]
				x, err := strconv.Atoi(xs)
				if err != nil {
					fmt.Println(err)
					continue
				}
				maxX = x
				fmt.Println("maxX", maxX)
			} else if strings.HasPrefix(line, "height") {
				fmt.Println("found height")
				hasH = true
				ys := line[7:]
				y, err := strconv.Atoi(ys)
				if err != nil {
					fmt.Println(err)
					continue
				}
				maxY = y
				fmt.Println("maxY", maxY)
			}
		} else if line == "More" { // Start processing any "More" entries.
			fmt.Println("found more")
			processMoreStart = i
			processMore = true
		}
	}

	fmt.Println("hasW", hasW, "hasH", hasH)

	if !hasW {
		fmt.Println("missing width")
		if maxX == 0 {
			maxX = maxY
			fmt.Println("maxX is 0, setting to maxY", maxY)
		}
		lines = append(lines[:archEnd], append([]string{"width " + strconv.Itoa(maxX+1)}, lines[archEnd:]...)...)
		changed = true
		fmt.Println("Added width", maxX+1)
		archEnd += 1
	}
	if !hasH {
		fmt.Println("missing height")
		if maxY == 0 {
			maxY = maxX
			fmt.Println("maxY is 0, setting to maxX", maxX)
		}
		lines = append(lines[:archEnd], append([]string{"height " + strconv.Itoa(maxY+1)}, lines[archEnd:]...)...)
		changed = true
		fmt.Println("Added height", maxY+1)
	}

	if changed {
		fmt.Println("Fixed:", name, fixedInvisible)
		err = os.WriteFile(name, []byte(strings.Join(lines, "\n")), 0644)
		if err != nil {
			fmt.Println(err)
		}
	}
}
