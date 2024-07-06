// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Nba-chatbot uses the Ollama API to generate embeddings for NBA statistics.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"

	ollama "github.com/ollama/ollama/api"
)

const model = "llama2:7b"

func main() {
	f, err := os.Open("stats.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rs, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rs {
		fmt.Printf("row: %v\n", row)
		js, err := json.Marshal(row)
		if err != nil {
			log.Fatal(err)
		}
		req := &ollama.EmbeddingRequest{
			Model:  model,
			Prompt: string(js),
		}
		resp, err := client.Embeddings(context.Background(), req)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("resp: %v\n", resp)
	}
}
