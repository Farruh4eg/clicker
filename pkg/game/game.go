// Package game contains game logic
package game

import (
	pb "clicker/gen/proto"
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/google/uuid"
)

type EnemyStats struct {
	EnemyMaxHp float64
	EnemyLevel int64
	// TODO: add armor, hp regen etc. in future
}

type Game struct {
	sync.Mutex
	LastEnemyID string
	Enemies     []*Enemy
	Players     map[string]chan *pb.ServerToClient // player id -> his channel
}

type Enemy struct {
	ID            string
	Name          string
	MaxHealth     float64
	CurrentHealth float64
	Level         int64
	Image         []byte
	// some fine grained mutex for future generations, maybe
	// sync.Mutex
}

const (
	BaseHp     = 100.0
	Multiplier = 1.1
)

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

func (g *Game) broadcast(msg *pb.ServerToClient) {
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

func (g *Game) ApplyDamage(enemyID string, incomingDamage float64, attackerID string) {
	g.Lock()
	defer g.Unlock()
	// calculate enemy armor and resistance values here in future maybe?
	// just substract damage for now
	// also, TODO: find enemy by id
	if len(g.Enemies) == 0 {
		// TODO: spawn more enemies
		log.Println("Attack ignored, no enemies to attack")
		return
	}

	enemy := g.Enemies[0]
	enemy.CurrentHealth -= incomingDamage

	if enemy.CurrentHealth > 0 {
		hitInfo := &pb.HitInfo{
			DamageDealt: incomingDamage,
			AttackerId:  attackerID,
		}

		g.broadcast(&pb.ServerToClient{
			Event: &pb.ServerToClient_GameStateUpdate{
				GameStateUpdate: &pb.GameStateUpdate{
					EnemyCurrentHp: enemy.CurrentHealth,
					EnemyId:        enemy.ID,
					LastHit:        hitInfo,
				},
			},
		})
		return
	}

	// destroy the enemy, spawn a new one, award xp, gold, hot wife
	log.Printf("Enemy %s (Level %d) died", enemy.Name, enemy.Level)

	// fix the memory leak?
	g.Enemies[0] = nil

	g.Enemies = g.Enemies[1:]
	if len(g.Enemies) == 0 {
		log.Println("All enemies have been defeated")
		g.broadcast(&pb.ServerToClient{
			// TODO: add new field to proto for this case?
			Event: &pb.ServerToClient_GameStateUpdate{
				GameStateUpdate: &pb.GameStateUpdate{
					EnemyId:        enemy.ID,
					EnemyCurrentHp: 0.0,
				},
			},
		})
		return
	}

	newEnemy := g.Enemies[0]
	log.Printf("Enemy died. Spawning next enemy: %s with %.2f HP", newEnemy.Name, newEnemy.MaxHealth)

	pbEnemy := &pb.Enemy{
		Id:        newEnemy.ID,
		Name:      newEnemy.Name,
		MaxHp:     newEnemy.MaxHealth,
		CurrentHp: newEnemy.MaxHealth,
		Level:     newEnemy.Level,
		Image:     newEnemy.Image,
	}

	g.broadcast(&pb.ServerToClient{
		Event: &pb.ServerToClient_EnemySpawned{
			EnemySpawned: &pb.NewEnemySpawned{
				Enemy: pbEnemy,
			},
		},
	})
	return
}

func (g *Game) CreateEnemyForLevel(level int64) *Enemy {
	hp := CalculateEnemyHp(level)
	stats := EnemyStats{
		EnemyMaxHp: hp,
		EnemyLevel: level,
	}
	name := fmt.Sprintf("Level %d monster", level)

	return g.CreateEnemy(stats, name, nil)
}

func (g *Game) CreateEnemy(enemyStats EnemyStats, name string, image []byte) *Enemy {
	g.Lock()
	defer g.Unlock()

	g.LastEnemyID = GenerateID()
	newEnemy := &Enemy{
		ID:            g.LastEnemyID,
		Name:          name,
		MaxHealth:     enemyStats.EnemyMaxHp,
		CurrentHealth: enemyStats.EnemyMaxHp,
		Level:         enemyStats.EnemyLevel,
		Image:         image,
	}

	g.Enemies = append(g.Enemies, newEnemy)

	return newEnemy
}

func GenerateID() string {
	return uuid.New().String()
}

func CalculateEnemyHp(level int64) float64 {
	return BaseHp * math.Pow(Multiplier, float64(level-1))
}
