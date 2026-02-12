//go:build linux && !tray

package main

import "fmt"

func main() {
	fmt.Println("dotctl tray source is present but excluded in this build.")
	fmt.Println("Build with: go build -tags tray ./linux/tray")
}
