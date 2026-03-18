package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

var (
	targetURL  = flag.String("url", "http://localhost:8080", "Base URL of the Koskidex Server")
	indexName  = flag.String("index", "massive", "Name of the index to create and seed")
	docCount   = flag.Int("count", 100000, "Total number of documents to seed")
	chunkSize  = flag.Int("chunk", 5000, "Number of documents per HTTP request")
)

var (
	adjectives = []string{"Red", "Blue", "Quantum", "Cyber", "Ultra", "Mega", "Hyper", "Super", "Neon", "Dark", "Light", "Solar"}
	nouns      = []string{"Phone", "Laptop", "Watch", "Tablet", "Monitor", "Keyboard", "Mouse", "Speaker", "Headphones", "Camera"}
	brands     = []string{"Apple", "Samsung", "Sony", "Logitech", "Asus", "Dell", "HP", "Lenovo", "Razer", "Corsair"}
	categories = []string{"Electronics", "Audio", "Computers", "Accessories", "Photography"}
)

func randomString(arr []string) string {
	return arr[rand.Intn(len(arr))]
}

func main() {
	flag.Parse()

	fmt.Printf("🎯 Target API: %s\n", *targetURL)
	fmt.Printf("📦 Creating Index: '%s'\n", *indexName)

	// 1. Create Index
	createBody := map[string]string{"name": *indexName}
	createJson, _ := json.Marshal(createBody)
	resp, err := http.Post(*targetURL+"/indexes", "application/json", bytes.NewBuffer(createJson))
	if err != nil {
		fmt.Printf("❌ Failed to create index: %v\n", err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
		fmt.Println("✅ Index ready.")
	} else {
		fmt.Printf("⚠️ Unexpected response on create index: %s\n", resp.Status)
	}

	// 2. Generate and Seed in Chunks
	fmt.Printf("🚀 Seeding %d documents in chunks of %d...\n", *docCount, *chunkSize)
	
	start := time.Now()
	var docs []map[string]interface{}
	
	for i := 1; i <= *docCount; i++ {
		docs = append(docs, map[string]interface{}{
			"id":       fmt.Sprintf("doc_%d", i),
			"name":     fmt.Sprintf("%s %s %d", randomString(adjectives), randomString(nouns), rand.Intn(9999)),
			"brand":    randomString(brands),
			"category": randomString(categories),
			"price":    float64(rand.Intn(2000)) + 0.99,
			"rating":   rand.Float64()*4 + 1.0, // 1.0 to 5.0
			"stock":    rand.Intn(500),
		})

		// Flush chunk
		if len(docs) == *chunkSize || i == *docCount {
			chunkJson, _ := json.Marshal(docs)
			url := fmt.Sprintf("%s/indexes/%s/documents", *targetURL, *indexName)
			
			reqStart := time.Now()
			chunkResp, err := http.Post(url, "application/json", bytes.NewBuffer(chunkJson))
			
			if err != nil {
				fmt.Printf("\n❌ Error sending chunk: %v\n", err)
				return
			}
			_ = chunkResp.Body.Close()
			
			fmt.Printf("📤 Seeded %d/%d documents (Took %v)\n", i, *docCount, time.Since(reqStart))
			docs = nil // Reset chunk
		}
	}

	totalTime := time.Since(start)
	fmt.Printf("\n🎉 Success! Seeded %d documents in %v.\n", *docCount, totalTime)
	fmt.Printf("🔍 Test the speed: %s/indexes/%s/search?q=%s\n", *targetURL, *indexName, "Apple")
}
