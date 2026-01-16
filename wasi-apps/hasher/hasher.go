package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// parse commandline flags, accepts -h / --help to print usage
	zeros := flag.Int("zeros", 0, "how many leading zeros to look for")
	message := flag.String("message", "Hello, World!", "the message to hash")
	flag.Parse()

	// verify we've passed useful values
	if *zeros <= 0 {
		log.Fatalln("use a positive integer for -zeros")
	}
	if *message == "" {
		log.Fatalln("give me a -message to hash")
	}

	// format the desired string of zeros once
	zerostr := strings.Repeat("0", *zeros)

	// try hashes in an infinite loop until we find one with leading zeros
	fmt.Fprintf(os.Stderr, "Searching hash for %q with %d leading zeros ...\n", *message, *zeros)
	for i := 0; ; i++ {

		// format the leading part of the message
		msg := fmt.Sprintf("%s|%d", *message, i)

		// check if its hash has leading zeros
		if hash, ok := HashAndCheckZeros(msg, zerostr); ok {
			// found a hash with leading zeros! print and exit
			fmt.Fprintf(os.Stderr, "%d!\n", i)
			fmt.Printf("%s => %s\n", msg, hash)
			os.Exit(0)
		}

		// print progress every few million iterations
		if i%1_000_000 == 0 {
			fmt.Fprintf(os.Stderr, "%d .. ", i)
		}

	}

}

func HashAndCheckZeros(message, zerostr string) (string, bool) {

	// compute the SHA256 hash and encode to hexadecimal string
	hash := sha256.Sum256([]byte(message))
	hashstr := hex.EncodeToString(hash[:])

	// check if the hash has leading zeros
	_, ok := strings.CutPrefix(hashstr, zerostr)
	return hashstr, ok

}
