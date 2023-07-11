package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/web3-storage/go-w3s-client"
)

var (
	filename = "proof.json"
)

func init() {
	flag.StringVar(&filename, "filename", filename, "path of the file to upload")
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("storeProof: ")
	log.SetOutput(os.Stderr)
	flag.Usage = usage
	flag.Parse()

	c, err := w3s.NewClient(w3s.WithToken(mustTokenFromEnv()))
	if err != nil {
		panic(err)
	}

	cid := putSingleFile(c)
	fmt.Println(cid)
}

func putSingleFile(c w3s.Client) cid.Cid {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	return putFile(c, file)
}

func putFile(c w3s.Client, f fs.File, opts ...w3s.PutOption) cid.Cid {
	cid, err := c.Put(context.Background(), f, opts...)
	if err != nil {
		panic(err)
	}
	fmt.Printf("https://%v.ipfs.dweb.link\n", cid)
	return cid
}

func mustTokenFromEnv() string {
	value := os.Getenv("W3FS_API_KEY")
	if value == "" {
		log.Fatal("the environment variable W3FS_API_KEY must be set")
	}

	return value
}

func usage() {
	usageString := `Usage: storeProof
Upload proof.json to IPFS and print the file's CID.

This utility read the authentication token from
the W3FS_API_KEY environment variable.
See  https://web3.storage.com/ for more information.
`
	_, _ = fmt.Fprintln(os.Stderr, usageString)

	flag.PrintDefaults()
}
