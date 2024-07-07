// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Nba-chatbot uses the Ollama API to generate embeddings for NBA statistics.
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lib/pq"
	ollama "github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

type player struct {
	Content   string          `json:"content"`
	Data      metadata        `json:"metadata"`
	Embedding pgvector.Vector `pg:"type:vector(4096)"`
}

type metadata struct {
	Source string `json:"source"`
	Row    int    `json:"row"`
}

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
	fields := rs[0]
	ps := make([]player, len(rs)-1)
	for i, r := range rs[1:] {
		var c string
		c, err = content(fields, r)
		if err != nil {
			log.Fatal(err)
		}
		ps[i] = player{Content: c, Data: metadata{Source: name, Row: i + 1}}
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
		if err = ps[i].embedding(client); err != nil {
			log.Fatalf("failed to generate embeddings: %v", err)
		}
	}
	if err = insertPlayers(db, ps); err != nil {
		log.Fatalf("failed to insert players: %v", err)
	}
}

func content(fields, row []string) (string, error) {
	if len(fields) != len(row) {
		return "", errors.New("fields and row must have the same length")
	}
	cs := make([]string, len(fields))
	for i, f := range fields {
		cs[i] = fmt.Sprintf("%q: %s", f, row[i])
	}
	return strings.Join(cs, "\n"), nil
}

func (p *player) embedding(c *ollama.Client) error {
	js, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req := &ollama.EmbeddingRequest{
		Model:  model,
		Prompt: string(js),
	}
	resp, err := c.Embeddings(context.Background(), req)
	if err != nil {
		return err
	}
	embed := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		embed[i] = float32(v)
	}
	p.Embedding = pgvector.NewVector(embed)
	return nil
}

func insertPlayers(db *sql.DB, ps []player) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := txn.Prepare(pq.CopyIn("player", "embedding", "source", "row_number"))
	if err != nil {
		return err
	}
	for _, p := range ps {
		if _, err = stmt.Exec(p.Embedding, p.Data.Source, p.Data.Row); err != nil {
			return err
		}
	}
	if _, err = stmt.Exec(); err != nil {
		return err
	}
	if err = stmt.Close(); err != nil {
		return err
	}
	return txn.Commit()
}
