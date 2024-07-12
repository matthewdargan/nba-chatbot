// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Server is an HTTP server for NBA statistics.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/matthewdargan/nba-chatbot/internal/nba"
	"github.com/ollama/ollama/api"
)

type playerPerGameRequest struct {
	Question string `json:"question"`
}

type playerPerGameResponse struct {
	Response string `json:"response"`
}

func main() {
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
	http.HandleFunc("/player-per-game", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		var req playerPerGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Question == "" {
			http.Error(w, "Missing question", http.StatusBadRequest)
			return
		}
		p, err := nba.NearestPlayer(ctx, client, conn, req.Question)
		if err != nil {
			http.Error(w, "Error finding nearest player", http.StatusInternalServerError)
		}
		prompt := fmt.Sprintf("Using these Player Per Game statistics: %s. Respond to this prompt: %s", p, req.Question)
		log.Println(prompt)
		stream := false
		genReq := &api.GenerateRequest{
			Model:  "llama3:8b",
			Prompt: prompt,
			Stream: &stream,
		}
		var resp playerPerGameResponse
		if err := client.Generate(ctx, genReq, func(r api.GenerateResponse) error {
			resp.Response = r.Response
			return nil
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Invalid response body", http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
