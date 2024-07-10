// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nba provides facilities for vectorizing NBA data.
package nba

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"
	"github.com/matthewdargan/nba-chatbot/internal/token"
	"github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

// A Player represents an NBA player.
type Player struct {
	tokens             string
	Embedding          pgvector.Vector `pg:"type:vector(4096)"`
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
}

const rLen = 30

// NewPlayer returns a new [Player] from the given fields and row.
func NewPlayer(fields, row []string) (Player, error) {
	if len(row) != rLen {
		return Player{}, fmt.Errorf("expected row of length %d, got %d", rLen, len(row))
	}
	p := Player{
		name:     row[1],
		position: row[2],
		team:     row[4],
	}
	var err error
	p.tokens, err = token.New(fields, row)
	if err != nil {
		return Player{}, err
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
		*ptr, err = strconv.Atoi(row[i])
		if err != nil {
			return Player{}, fmt.Errorf("failed to parse integer at index %d: %v", i, err)
		}
	}
	for i, ptr := range floats {
		*ptr, err = strconv.ParseFloat(row[i], 64)
		if err != nil {
			return Player{}, fmt.Errorf("failed to parse float at index %d: %v", i, err)
		}
	}
	for i, ptr := range optionalFloats {
		if row[i] == "" {
			continue
		}
		var v float64
		v, err = strconv.ParseFloat(row[i], 64)
		if err != nil {
			return Player{}, fmt.Errorf("failed to parse optional float at index %d: %v", i, err)
		}
		*ptr = &v
	}
	return p, nil
}

// GenerateEmbeddings generates player embeddings.
func (p *Player) GenerateEmbeddings(c *api.Client, model string) error {
	req := &api.EmbeddingRequest{
		Model:  model,
		Prompt: p.tokens,
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

// InsertPlayers inserts players into a database.
func InsertPlayers(db *sql.DB, ps []Player) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := txn.Prepare(pq.CopyIn(
		"player_per_game", "embedding", "rank", "name", "position", "age", "team",
		"games", "games_started", "minutes_played", "field_goals", "field_goal_attempts",
		"field_goal_pct", "three_pointers", "three_point_attempts", "three_point_pct",
		"two_pointers", "two_point_attempts", "two_point_pct", "effective_fg_pct",
		"free_throws", "free_throw_attempts", "free_throw_pct", "offensive_rebounds",
		"defensive_rebounds", "total_rebounds", "assists", "steals", "blocks",
		"turnovers", "personal_fouls", "points",
	))
	if err != nil {
		return err
	}
	for _, p := range ps {
		if _, err = stmt.Exec(
			p.Embedding, p.rank, p.name, p.position, p.age, p.team, p.games, p.gamesStarted,
			p.minutesPlayed, p.fieldGoals, p.fieldGoalAttempts, p.fieldGoalPct,
			p.threePointers, p.threePointAttempts, p.threePointPct, p.twoPointers,
			p.twoPointAttempts, p.twoPointPct, p.effectiveFGPct, p.freeThrows,
			p.freeThrowAttempts, p.freeThrowPct, p.offensiveRebounds, p.defensiveRebounds,
			p.totalRebounds, p.assists, p.steals, p.blocks, p.turnovers,
			p.personalFouls, p.points,
		); err != nil {
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

const playerByNameQuery = `
    SELECT
        embedding, rank, name, position, age, team, games, games_started,
        minutes_played, field_goals, field_goal_attempts, field_goal_pct,
        three_pointers, three_point_attempts, three_point_pct, two_pointers,
        two_point_attempts, two_point_pct, effective_fg_pct, free_throws,
        free_throw_attempts, free_throw_pct, offensive_rebounds, defensive_rebounds,
        total_rebounds, assists, steals, blocks, turnovers, personal_fouls, points
    FROM player_per_game
    WHERE name = $1
`

// PlayerByName retrieves player data by name. Multiple entries may be returned
// if the player was traded mid-season.
func PlayerByName(db *sql.DB, name string) ([]Player, error) {
	rows, err := db.Query(playerByNameQuery, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(
			&p.Embedding, &p.rank, &p.name, &p.position, &p.age, &p.team, &p.games, &p.gamesStarted,
			&p.minutesPlayed, &p.fieldGoals, &p.fieldGoalAttempts, &p.fieldGoalPct,
			&p.threePointers, &p.threePointAttempts, &p.threePointPct, &p.twoPointers,
			&p.twoPointAttempts, &p.twoPointPct, &p.effectiveFGPct, &p.freeThrows,
			&p.freeThrowAttempts, &p.freeThrowPct, &p.offensiveRebounds, &p.defensiveRebounds,
			&p.totalRebounds, &p.assists, &p.steals, &p.blocks, &p.turnovers,
			&p.personalFouls, &p.points,
		); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}
