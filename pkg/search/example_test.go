package search_test

import (
	"fmt"
	"log"
	"os"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

func ExampleNewRouter() {
	router := search.NewRouter()
	router.Register(search.NewScrapedoProvider("token"))
	fmt.Println(router.AllEngines())
	// Output: [google]
}

func ExampleRouter_Resolve() {
	router := search.NewRouter()
	router.Register(search.NewScrapedoProvider("token"))

	p, err := router.Resolve("google", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(p.Name())
	// Output: scrapedo
}

func ExampleFormatTable() {
	resp := &search.Response{
		Query: "test", Engine: "google", Provider: "example",
		Results: []search.Result{
			{Position: 1, Title: "Example", URL: "https://example.com", Snippet: "A test result"},
		},
	}
	_ = search.FormatTable(os.Stdout, resp)
	// Output:
	// #  Title    URL
	// 1  Example  example.com
}
