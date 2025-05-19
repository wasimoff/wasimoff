package main

import (
	"fmt"
	"wasimoff/broker/config"
)

// Print a figlet "wasimoff" banner.
// figlet -f small wasimoff | sed -e 's/\\/\\\\/g' -e 's/.*/log.Println("&")/'
func printBanner() {
	fmt.Println("                  _            __  __ ")
	fmt.Println("  __ __ ____ _ __(_)_ __  ___ / _|/ _|")
	fmt.Println("  \\ V  V / _` (_-< | '  \\/ _ \\  _|  _|")
	fmt.Println("   \\_/\\_/\\__,_/__/_|_|_|_\\___/_| |_|  ")
	fmt.Println()
}

func printVersion() {
	fmt.Printf("   %s (%s) %s\n", config.Version.Package, config.Version.Revision, config.Version.GoVersion)
	fmt.Println()
}

// termclear clears the terminal for easier development with watchers
func termclear() {
	fmt.Print("\033[H\033[2J")
}
