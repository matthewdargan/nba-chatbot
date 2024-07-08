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
	"strconv"
	"strings"

	"github.com/lib/pq"
	ollama "github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

type player struct {
	row                row
	rank               int
	name               string
	position           string
	age                int
	team               string
	games              int
	gamesStarted       int
	minutesPlayed      float64
	fieldGoals         float64
	fieldGoalAttempts  float64
	fieldGoalPct       *float64
	threePointers      float64
	threePointAttempts float64
	threePointPct      *float64
	twoPointers        float64
	twoPointAttempts   float64
	twoPointPct        *float64
	effectiveFGPct     *float64
	freeThrows         float64
	freeThrowAttempts  float64
	freeThrowPct       *float64
	offensiveRebounds  float64
	defensiveRebounds  float64
	totalRebounds      float64
	assists            float64
	steals             float64
	blocks             float64
	turnovers          float64
	personalFouls      float64
	points             float64
	Embedding          pgvector.Vector `pg:"type:vector(4096)"`
}

type row struct {
	Content string `json:"content"`
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
	fields := rs[0][:len(rs[0])-1]
	ps := make([]player, len(rs)-1)
	for i, r := range rs[1:] {
		r = r[:len(r)-1]
		var c string
		c, err = content(fields, r)
		if err != nil {
			log.Fatal(err)
		}
		ps[i], err = newPlayer(r)
		if err != nil {
			log.Fatal(err)
		}
		ps[i].row = row{Content: c}
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

func newPlayer(r []string) (player, error) {
	if len(r) != 30 {
		return player{}, fmt.Errorf("expected row of length 30, got %d", len(r))
	}
	p := player{
		name:     r[1],
		position: r[2],
		team:     r[4],
	}
	ints := map[int]*int{
		0: &p.rank,
		3: &p.age,
		5: &p.games,
		6: &p.gamesStarted,
	}
	floats := map[int]*float64{
		7:  &p.minutesPlayed,
		8:  &p.fieldGoals,
		9:  &p.fieldGoalAttempts,
		11: &p.threePointers,
		12: &p.threePointAttempts,
		14: &p.twoPointers,
		15: &p.twoPointAttempts,
		18: &p.freeThrows,
		19: &p.freeThrowAttempts,
		21: &p.offensiveRebounds,
		22: &p.defensiveRebounds,
		23: &p.totalRebounds,
		24: &p.assists,
		25: &p.steals,
		26: &p.blocks,
		27: &p.turnovers,
		28: &p.personalFouls,
		29: &p.points,
	}
	optionalFloats := map[int]**float64{
		10: &p.fieldGoalPct,
		13: &p.threePointPct,
		16: &p.twoPointPct,
		17: &p.effectiveFGPct,
		20: &p.freeThrowPct,
	}
	for i, ptr := range ints {
		v, err := strconv.Atoi(r[i])
		if err != nil {
			fmt.Printf("r: %v\n", r)
			return player{}, fmt.Errorf("failed to parse integer at index %d: %v", i, err)
		}
		*ptr = v
	}
	for i, ptr := range floats {
		v, err := strconv.ParseFloat(r[i], 64)
		if err != nil {
			fmt.Printf("r: %v\n", r)
			return player{}, fmt.Errorf("failed to parse float at index %d: %v", i, err)
		}
		*ptr = v
	}
	for i, ptr := range optionalFloats {
		if r[i] == "" {
			continue
		}
		v, err := strconv.ParseFloat(r[i], 64)
		if err != nil {
			fmt.Printf("r: %v\n", r)
			return player{}, fmt.Errorf("failed to parse optional float at index %d: %v", i, err)
		}
		*ptr = &v
	}
	return p, nil
}

func (p *player) embedding(c *ollama.Client) error {
	js, err := json.Marshal(p.row)
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
	stmt, err := txn.Prepare(pq.CopyIn(
		"player_per_game", "rank", "name", "position", "age", "team", "games",
		"games_started", "minutes_played", "field_goals", "field_goal_attempts",
		"field_goal_pct", "three_pointers", "three_point_attempts", "three_point_pct",
		"two_pointers", "two_point_attempts", "two_point_pct", "effective_fg_pct",
		"free_throws", "free_throw_attempts", "free_throw_pct", "offensive_rebounds",
		"defensive_rebounds", "total_rebounds", "assists", "steals", "blocks",
		"turnovers", "personal_fouls", "points", "embedding",
	))
	if err != nil {
		return err
	}
	for _, p := range ps {
		_, err = stmt.Exec(
			p.rank, p.name, p.position, p.age, p.team, p.games, p.gamesStarted,
			p.minutesPlayed, p.fieldGoals, p.fieldGoalAttempts, p.fieldGoalPct,
			p.threePointers, p.threePointAttempts, p.threePointPct, p.twoPointers,
			p.twoPointAttempts, p.twoPointPct, p.effectiveFGPct, p.freeThrows,
			p.freeThrowAttempts, p.freeThrowPct, p.offensiveRebounds, p.defensiveRebounds,
			p.totalRebounds, p.assists, p.steals, p.blocks, p.turnovers,
			p.personalFouls, p.points, p.Embedding,
		)
		if err != nil {
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
