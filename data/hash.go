package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"repo/storage"
)

func main() {
	h := sha256.New()
	h.Write([]byte("Password"))
	fmt.Println(base64.StdEncoding.EncodeToString((h.Sum(nil))))

	fmt.Println(storage.User{
		Username:     "ivan",
		Email:        "ivanivanov@mail.ru",
		PasswordHash: "588+9PF8OZmpTyxvYS6KiI5bECaHjk4ZOYsjvTjsIho=",
	})
}
