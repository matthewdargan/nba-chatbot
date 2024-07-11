// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Server is an HTTP server for NBA statistics.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/matthewdargan/nba-chatbot/internal/nba"
	"github.com/ollama/ollama/api"
)

type playerPerGameRequest struct {
	Question string `json:"question"`
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
		if req.Question == "" {
			http.Error(w, "missing question", http.StatusBadRequest)
			return
		}
		client, err := api.ClientFromEnvironment()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		// TODO: put in nba package?
		p, err := nba.NearestPlayer(client, db, req.Question)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		prompt := fmt.Sprintf("Using these Player Per Game statistics: %s. Respond to this prompt: %s", p, req.Question)
		fmt.Println(prompt)
		genReq := &api.GenerateRequest{Model: model, Prompt: prompt}
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
