package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func main() {
	h := sha256.New()
	h.Write([]byte("Password"))
	fmt.Println(base64.StdEncoding.EncodeToString((h.Sum(nil))))

}
