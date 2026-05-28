package scrapedo_test

import (
	"fmt"
	"log"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func ExampleNewClient() {
	client, err := scrapedo.NewClient("your-api-token")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(client != nil)
	// Output: true
}

func ExampleExtractLinks() {
	html := `<a href="/about">About</a> <a href="/docs">Docs</a>`
	links := scrapedo.ExtractLinks(html, "https://example.com")
	for _, l := range links {
		fmt.Println(l)
	}
	// Output:
	// https://example.com/about
	// https://example.com/docs
}
