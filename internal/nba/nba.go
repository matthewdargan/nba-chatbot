// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nba provides facilities for vectorizing NBA data.
package nba

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

// A Player represents an NBA player.
type Player struct {
	tokens             string
	Embedding          pgvector.Vector
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

func (p Player) String() string {
	return fmt.Sprintf(
		`"Name": %s, "Position": %s, "Age": %d, "Team": %s, "Games": %d, `+
			`"Games Started": %d, "Minutes Played": %.2f, "Field Goals": %.2f, `+
			`"Field Goal Attempts": %.2f, "Field Goal %%": %s, "3-Pointers": %.2f, `+
			`"3-Point Attempts": %.2f, "3-Point %%": %s, "2-Pointers": %.2f, `+
			`"2-Point Attempts": %.2f, "2-Point %%": %s, "Effective FG %%": %s, `+
			`"Free Throws": %.2f, "Free Throw Attempts": %.2f, "Free Throw %%": %s, `+
			`"Offensive Rebounds": %.2f, "Defensive Rebounds": %.2f, "Total Rebounds": %.2f, `+
			`"Assists": %.2f, "Steals": %.2f, "Blocks": %.2f, "Turnovers": %.2f, `+
			`"Personal Fouls": %.2f, "Points": %.2f`,
		p.name, p.position, p.age, p.team, p.games, p.gamesStarted, p.minutesPlayed,
		p.fieldGoals, p.fieldGoalAttempts, formatPercentage(p.fieldGoalPct),
		p.threePointers, p.threePointAttempts, formatPercentage(p.threePointPct),
		p.twoPointers, p.twoPointAttempts, formatPercentage(p.twoPointPct),
		formatPercentage(p.effectiveFGPct), p.freeThrows, p.freeThrowAttempts,
		formatPercentage(p.freeThrowPct), p.offensiveRebounds, p.defensiveRebounds,
		p.totalRebounds, p.assists, p.steals, p.blocks, p.turnovers, p.personalFouls,
		p.points,
	)
}

func formatPercentage(p *float64) string {
	if p == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2f%%", *p*100)
}

// NewPlayer returns a new [Player] from the given fields and row.
func NewPlayer(fields, row []string) (Player, error) {
	const rLen = 30
	if len(row) != rLen {
		return Player{}, fmt.Errorf("expected row of length %d, got %d", rLen, len(row))
	}
	p := Player{
		name:     row[1],
		position: row[2],
		team:     row[4],
	}
	var err error
	p.tokens, err = tokens(fields, row)
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

func tokens(fields, row []string) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("empty fields")
	}
	if len(fields) != len(row) {
		return "", errors.New("fields and row must have the same length")
	}
	ts := make([]string, len(fields))
	for i, f := range fields {
		ts[i] = fmt.Sprintf("%q: %s", f, row[i])
	}
	return strings.Join(ts, "\n"), nil
}

const embeddingModel = "mxbai-embed-large"

// GenerateEmbeddings generates player embeddings.
func (p *Player) GenerateEmbeddings(ctx context.Context, c *api.Client) error {
	req := &api.EmbeddingRequest{Model: embeddingModel, Prompt: p.tokens}
	resp, err := c.Embeddings(ctx, req)
	if err != nil {
		return err
	}
	eb := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		eb[i] = float32(v)
	}
	p.Embedding = pgvector.NewVector(eb)
	return nil
}

// InsertPlayers inserts players into a database.
func InsertPlayers(ctx context.Context, conn *pgx.Conn, ps []Player) error {
	b := &pgx.Batch{}
	for _, p := range ps {
		b.Queue(
			`
INSERT INTO player_per_game (
    embedding, rank, name, position, age, team, games, games_started,
    minutes_played, field_goals, field_goal_attempts, field_goal_pct,
    three_pointers, three_point_attempts, three_point_pct, two_pointers,
    two_point_attempts, two_point_pct, effective_fg_pct, free_throws,
    free_throw_attempts, free_throw_pct, offensive_rebounds, defensive_rebounds,
    total_rebounds, assists, steals, blocks, turnovers, personal_fouls, points
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
    $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
)`,
			p.Embedding, p.rank, p.name, p.position, p.age, p.team, p.games,
			p.gamesStarted, p.minutesPlayed, p.fieldGoals, p.fieldGoalAttempts,
			p.fieldGoalPct, p.threePointers, p.threePointAttempts, p.threePointPct,
			p.twoPointers, p.twoPointAttempts, p.twoPointPct, p.effectiveFGPct,
			p.freeThrows, p.freeThrowAttempts, p.freeThrowPct, p.offensiveRebounds,
			p.defensiveRebounds, p.totalRebounds, p.assists, p.steals, p.blocks,
			p.turnovers, p.personalFouls, p.points,
		)
	}
	br := conn.SendBatch(ctx, b)
	for range b.Len() {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return br.Close()
}

// NearestPlayer returns the player whose embedding is closest to the given question.
func NearestPlayer(ctx context.Context, c *api.Client, conn *pgx.Conn, question string) (Player, error) {
	req := &api.EmbeddingRequest{Model: embeddingModel, Prompt: question}
	resp, err := c.Embeddings(ctx, req)
	if err != nil {
		return Player{}, err
	}
	eb := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		eb[i] = float32(v)
	}
	var p Player
	if err := conn.QueryRow(ctx, `
SELECT
    rank, name, position, age, team, games, games_started, minutes_played,
    field_goals, field_goal_attempts, field_goal_pct, three_pointers,
    three_point_attempts, three_point_pct, two_pointers, two_point_attempts,
    two_point_pct, effective_fg_pct, free_throws, free_throw_attempts,
    free_throw_pct, offensive_rebounds, defensive_rebounds, total_rebounds,
    assists, steals, blocks, turnovers, personal_fouls, points
FROM player_per_game
ORDER BY embedding <-> $1
LIMIT 1`, pgvector.NewVector(eb)).Scan(
		&p.rank, &p.name, &p.position, &p.age, &p.team, &p.games, &p.gamesStarted,
		&p.minutesPlayed, &p.fieldGoals, &p.fieldGoalAttempts, &p.fieldGoalPct,
		&p.threePointers, &p.threePointAttempts, &p.threePointPct, &p.twoPointers,
		&p.twoPointAttempts, &p.twoPointPct, &p.effectiveFGPct, &p.freeThrows,
		&p.freeThrowAttempts, &p.freeThrowPct, &p.offensiveRebounds, &p.defensiveRebounds,
		&p.totalRebounds, &p.assists, &p.steals, &p.blocks, &p.turnovers,
		&p.personalFouls, &p.points,
	); err != nil {
		return Player{}, err
	}
	return p, nil
}
