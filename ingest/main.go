// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ingest ingests NBA statistics and generates embeddings.
//
// Usage:
//
//	ingest file
//
// Example:
//
// Generate embeddings for statistics in `stats/player-per-game.csv`:
//
//	$ ingest 'stats/player-per-game.csv'
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/matthewdargan/nba-chatbot/internal/nba"
	"github.com/ollama/ollama/api"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: ingest file\n")
	os.Exit(2)
}

func main() {
	log.SetPrefix("ingest: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	f, err := os.Open(flag.Arg(0))
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
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)
	for i := range ps {
		if err = ps[i].GenerateEmbeddings(ctx, client); err != nil {
			log.Fatalf("failed to generate embeddings: %v", err)
		}
	}
	if err = nba.InsertPlayers(ctx, conn, ps); err != nil {
		log.Fatalf("failed to insert players: %v", err)
	}
}
