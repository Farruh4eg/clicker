// Package game contains game logic
package game

import (
	pb "clicker/gen/proto"
	"image"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Game struct {
	sync.Mutex
	LastEnemyID string
	Enemies     []*Enemy
	Players     map[string]chan *pb.ServerToClient // player id -> his channel
}

func (g *Game) AddPlayer(playerID string, updateChan chan *pb.ServerToClient) {
	g.Lock()
	defer g.Unlock()
	if g.Players == nil {
		g.Players = make(map[string]chan *pb.ServerToClient)
	}
	g.Players[playerID] = updateChan
}

func (g *Game) RemovePlayer(playerID string) {
	g.Lock()
	defer g.Unlock()
	if channel, ok := g.Players[playerID]; ok {
		delete(g.Players, playerID)
		close(channel)
	}
}

func (g *Game) Broadcast(msg *pb.ServerToClient) {
	g.Lock()
	defer g.Unlock()

	for id, channel := range g.Players {
		select {
		case channel <- msg:
		default:
			log.Printf("Player %s update channel is full. Message dropped", id)
		}
	}
}

func NewGame() *Game {
	return &Game{
		Enemies: make([]*Enemy, 0, 10),
		Players: make(map[string]chan *pb.ServerToClient),
	}
}

type Enemy struct {
	ID            string
	Name          string
	MaxHealth     float64
	CurrentHealth float64
	Image         image.Image
	// some fine grained mutex for future generations, maybe
	// sync.Mutex
}

func (g *Game) ApplyDamage(enemyID string, incomingDamage float64) (*Enemy, error) {
	g.Lock()
	defer g.Unlock()
	// calculate enemy armor and resistance values here in future maybe?
	// just substract damage for now
	// also, TODO: find enemy by id
	enemy := g.Enemies[0]
	enemy.CurrentHealth -= incomingDamage

	if enemy.CurrentHealth <= 0 {
		// TODO: destroy the enemy, spawn a new one, award xp, gold, hot wife
	}
	return enemy, nil
}

func (g *Game) CreateEnemy() *Enemy {
	g.Lock()
	defer g.Unlock()

	g.LastEnemyID = GenerateID()
	newEnemy := &Enemy{
		ID:            g.LastEnemyID,
		Name:          "Retard",
		MaxHealth:     100.0,
		CurrentHealth: 100.0,
		Image:         nil,
	}

	g.Enemies = append(g.Enemies, newEnemy)

	return newEnemy
}

func GenerateID() string {
	return uuid.New().String()
}
