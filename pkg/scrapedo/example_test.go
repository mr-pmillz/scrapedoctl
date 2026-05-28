package scrapedo_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

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

// ExampleClient_Info demonstrates querying the Scrape.do /info endpoint
// to size per-run concurrency. The httptest server stands in for
// api.scrape.do so the example is deterministic for `go test`.
func ExampleClient_Info() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "IsActive": true,
		  "ConcurrentRequest": 10,
		  "MaxMonthlyRequest": 250000,
		  "RemainingConcurrentRequest": 10,
		  "RemainingMonthlyRequest": 247675
		}`))
	}))
	defer srv.Close()

	client, err := scrapedo.NewClient("your-api-token")
	if err != nil {
		log.Fatal(err)
	}
	client.SetBaseURL(srv.URL) // point at the httptest server for the example

	info, err := client.Info(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("active=%v concurrent=%d remaining=%d/%d\n",
		info.IsActive,
		info.ConcurrentRequest,
		info.RemainingMonthlyRequest,
		info.MaxMonthlyRequest)
	// Output: active=true concurrent=10 remaining=247675/250000
}
