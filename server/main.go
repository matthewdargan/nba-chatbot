// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Server is an HTTP server for NBA statistics.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/matthewdargan/nba-chatbot/internal/nba"
	"github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

type playerPerGameRequest struct {
	Question   string `json:"question"`
	PlayerName string `json:"player_name"`
}

type prompt struct {
	Question   string            `json:"question"`
	Embeddings []pgvector.Vector `json:"embeddings"`
}

const model = "llama3:8b"

func main() {
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	http.HandleFunc("/player-per-game", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		var req playerPerGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Question == "" || req.PlayerName == "" {
			http.Error(w, "missing question or player_name", http.StatusBadRequest)
			return
		}
		player, err := nba.PlayerByName(db, req.PlayerName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		client, err := api.ClientFromEnvironment()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		// TODO: put in nba package?
		pt := prompt{Question: req.Question}
		for _, p := range player {
			pt.Embeddings = append(pt.Embeddings, p.Embedding)
		}
		js, err := json.Marshal(pt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		genReq := &api.GenerateRequest{
			Model:  model,
			Prompt: string(js),
		}
		log.Printf("genReq: %+v", genReq)
		var bs []byte
		if err := client.Generate(context.Background(), genReq, func(r api.GenerateResponse) error {
			bs = append(bs, r.Response...)
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		bs = append(bs, '\n')
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(bs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
