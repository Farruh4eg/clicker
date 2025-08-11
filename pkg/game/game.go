// Package game contains game logic
package game

import (
	pb "clicker/gen/proto"
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

func (g *Game) ApplyDamage(enemyID string, incomingDamage float64) (*Enemy, error) {
	g.Lock()
	defer g.Unlock()
	// calculate enemy armor and resistance values here in future maybe?
	// just substract damage for now
	// also, TODO: find enemy by id
	if len(g.Enemies) == 0 {
		// TODO: spawn more enemies
		return nil, nil
	}

	enemy := g.Enemies[0]
	enemy.CurrentHealth -= incomingDamage

	if enemy.CurrentHealth <= 0 {
		// TODO: destroy the enemy, spawn a new one, award xp, gold, hot wife
		// unoptimized (yet :)
		log.Printf("Enemy %s (Level %d) died", enemy.Name, enemy.Level)

		g.Enemies = g.Enemies[1:]
		if len(g.Enemies) == 0 {
			log.Println("All enemies have been defeated")
			g.Broadcast(&pb.ServerToClient{
				// TODO: add new field to proto for this case?
				Event: &pb.ServerToClient_GameStateUpdate{
					GameStateUpdate: &pb.GameStateUpdate{
						EnemyId:        enemy.ID,
						EnemyCurrentHp: 0.0,
					},
				},
			})

			return nil, nil
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

		g.Broadcast(&pb.ServerToClient{
			Event: &pb.ServerToClient_EnemySpawned{
				EnemySpawned: &pb.NewEnemySpawned{
					Enemy: pbEnemy,
				},
			},
		})
		return newEnemy, nil

	} else {
		hitInfo := &pb.HitInfo{
			DamageDealt: incomingDamage,
		}

		g.Broadcast(&pb.ServerToClient{
			Event: &pb.ServerToClient_GameStateUpdate{
				GameStateUpdate: &pb.GameStateUpdate{
					EnemyCurrentHp: enemy.CurrentHealth,
					EnemyId:        enemy.ID,
					LastHit:        hitInfo,
				},
			},
		})
	}

	return enemy, nil
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

func CalculateEnemyHp(level int64, baseHp float64, multiplier float64) float64 {
	return baseHp * math.Pow(multiplier, float64(level-1))
}
