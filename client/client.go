package client

import (
	"log"
	"net/http"
	"net/http/cookiejar"
)

func New() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	return &http.Client{Jar: jar}
}
