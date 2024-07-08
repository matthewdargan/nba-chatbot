// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Nba-chatbot uses the Ollama API to generate embeddings for NBA statistics.
package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/matthewdargan/nba-chatbot/internal/player"
	ollama "github.com/ollama/ollama/api"
)

const model = "llama3:8b"

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
	ps := make([]player.Player, len(rs)-1)
	for i, r := range rs[1:] {
		r = r[:len(r)-1]
		var ts string
		ts, err = tokens(fields, r)
		if err != nil {
			log.Fatal(err)
		}
		ps[i], err = player.New(r)
		if err != nil {
			log.Fatal(err)
		}
		ps[i].Input.Tokens = ts
	}
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	for i := range ps {
		if err = ps[i].Embeddings(client, model); err != nil {
			log.Fatalf("failed to generate embeddings: %v", err)
		}
	}
	if err = player.InsertPlayers(db, ps); err != nil {
		log.Fatalf("failed to insert players: %v", err)
	}
}

func tokens(fields, row []string) (string, error) {
	if len(fields) != len(row) {
		return "", errors.New("fields and row must have the same length")
	}
	cs := make([]string, len(fields))
	for i, f := range fields {
		cs[i] = fmt.Sprintf("%q: %s", f, row[i])
	}
	return strings.Join(cs, "\n"), nil
}
