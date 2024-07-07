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
	"strings"

	ollama "github.com/ollama/ollama/api"
)

type player struct {
	Content string   `json:"content"`
	Data    metadata `json:"metadata"`
}

type metadata struct {
	Source string `json:"source"`
	Row    int    `json:"row"`
}

const model = "llama2:7b"

func main() {
	name := "stats/player-per-game.csv"
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	rs, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	fields := rs[0]
	ps := make([]player, len(rs[1:]))
	for i, row := range rs[1:] {
		p := player{
			Content: content(fields, row),
			Data: metadata{
				Source: name,
				Row:    i + 1,
			},
		}
		ps[i] = p
	}
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range ps {
		js, err := json.Marshal(p)
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

func content(fields, row []string) string {
	var b strings.Builder
	for i, c := range row {
		if i < len(row)-1 {
			fmt.Fprintf(&b, "\"%s\": %s\n", fields[i], c)
		} else {
			fmt.Fprintf(&b, "\"%s\": %s", fields[i], c)
		}
	}
	return b.String()
}
