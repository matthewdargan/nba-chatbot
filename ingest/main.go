// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ingest ingests NBA statistics and generates embeddings.
package main

import (
	"database/sql"
	"encoding/csv"
	"log"
	"os"

	"github.com/matthewdargan/nba-chatbot/internal/nba"
	"github.com/ollama/ollama/api"
)

const model = "mxbai-embed-large"

func main() {
	name := "stats/player-per-game.csv"
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	rs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	fields := rs[0][:len(rs[0])-1]
	ps := make([]nba.Player, len(rs)-1)
	for i, r := range rs[1:] {
		r = r[:len(r)-1]
		ps[i], err = nba.NewPlayer(fields, r)
		if err != nil {
			log.Fatal(err)
		}
	}
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	for i := range ps {
		if err = ps[i].GenerateEmbeddings(client, model); err != nil {
			log.Fatalf("failed to generate embeddings: %v", err)
		}
	}
	if err = nba.InsertPlayers(db, ps); err != nil {
		log.Fatalf("failed to insert players: %v", err)
	}
}
