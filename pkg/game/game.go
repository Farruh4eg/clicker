// Package game contains game logic
package game

import (
	"bytes"
	pb "clicker/gen/proto"
	"fmt"
	"image/png"
	"log"
	"math"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
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
	Players     map[string]*PlayerSession // player id -> his session
}

type PlayerSession struct {
	Data    *pb.Player
	Updates chan *pb.ServerToClient
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

	BaseGoldPerKill            = 10
	BaseExpPerKill             = 5
	LastHitGoldBonusMultiplier = 1.5
	LastHitExpBonusMultiplier  = 2.0

	WeaponUpgradeBaseCost       = 50
	WeaponUpgradeCostMultiplier = 1.8
)

func (g *Game) AddPlayer(player *pb.Player, updateChan chan *pb.ServerToClient) {
	g.Lock()
	defer g.Unlock()
	if g.Players == nil {
		g.Players = make(map[string]*PlayerSession)
	}
	g.Players[player.GetId()] = &PlayerSession{
		Data:    player,
		Updates: updateChan,
	}
}

func (g *Game) RemovePlayer(playerID string) {
	g.Lock()
	defer g.Unlock()
	if session, ok := g.Players[playerID]; ok {
		delete(g.Players, playerID)
		close(session.Updates)
	}
}

func (g *Game) broadcastToAll(msg *pb.ServerToClient) {
	for _, session := range g.Players {
		select {
		case session.Updates <- msg:
		default:
			log.Printf("Player %s update channel is full. Message dropped", session.Data.GetId())
		}
	}
}

func (g *Game) Broadcast(msg *pb.ServerToClient, excludePlayerID string) {
	for id, session := range g.Players {
		if id == excludePlayerID {
			continue
		}

		select {
		case session.Updates <- msg:
		default:
			log.Printf("Player %s update channel is full. Message dropped", session.Data.GetId())
		}
	}
}

func (g *Game) sendToPlayer(playerID string, msg *pb.ServerToClient) {
	if session, ok := g.Players[playerID]; ok {
		select {
		case session.Updates <- msg:
		default:
			log.Printf("Player %s update channel is full. Message dropped", session.Data.GetId())
		}
	}
}

func (e *Enemy) ToProto() *pb.Enemy {
	return &pb.Enemy{
		Id:        e.ID,
		Name:      e.Name,
		MaxHp:     e.MaxHealth,
		CurrentHp: e.CurrentHealth,
		Level:     e.Level,
		Image:     e.Image,
	}
}

func InitializePlayer(name string) *pb.Player {
	player := &pb.Player{
		Id:   GenerateID(),
		Name: name,
		Stats: &pb.PlayerStats{
			Level:        1,
			Experience:   0,
			NextLevelExp: 100,
		},
		Resources: &pb.PlayerResources{
			Gold: 2,
		},
		Equipment: &pb.PlayerEquipment{
			Weapon: &pb.Weapon{
				ItemId:       "starter_stick",
				Name:         "Деревянная палка",
				Level:        1,
				BaseDamage:   5.0,
				DamageGrowth: 2.0,
			},
		},
	}
	return player
}

func (g *Game) GetCurrentEnemy() *Enemy {
	g.Lock()
	defer g.Unlock()
	if len(g.Enemies) == 0 {
		return nil
	}
	return g.Enemies[0]
}

func (g *Game) GetAllPlayers() []*pb.Player {
	g.Lock()
	defer g.Unlock()
	players := make([]*pb.Player, 0, len(g.Players))
	for _, session := range g.Players {
		players = append(players, session.Data)
	}
	return players
}

func NewGame() *Game {
	return &Game{
		Enemies: make([]*Enemy, 0, 10),
		Players: make(map[string]*PlayerSession),
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

		g.broadcastToAll(&pb.ServerToClient{
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

	baseGold := int64(BaseGoldPerKill * enemy.Level)
	baseExp := int64(BaseExpPerKill * enemy.Level)

	lastHitBonusGold := int64(float64(baseGold) * (LastHitGoldBonusMultiplier - 1))
	lastHitBonusExp := int64(float64(baseExp) * (LastHitExpBonusMultiplier - 1))

	for _, session := range g.Players {
		player := session.Data
		player.Resources.Gold += baseGold
		player.Stats.Experience += baseExp

		if player.GetId() == attackerID {
			player.Resources.Gold += lastHitBonusGold
			player.Stats.Experience += lastHitBonusExp
			log.Printf("Player %s received a last hit bonus: +%d Gold, +%d Exp\n", player.GetName(), lastHitBonusGold, lastHitBonusExp)
		}

		g.checkForLevelUp(player)

		g.sendToPlayer(player.GetId(), &pb.ServerToClient{
			Event: &pb.ServerToClient_PlayerStateUpdate{
				PlayerStateUpdate: &pb.PlayerStateUpdate{
					Player: player,
				},
			},
		})
	}

	// fix the memory leak?
	g.Enemies[0] = nil

	g.Enemies = g.Enemies[1:]
	if len(g.Enemies) == 0 {
		log.Println("All enemies have been defeated")
		g.broadcastToAll(&pb.ServerToClient{
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

	g.broadcastToAll(&pb.ServerToClient{
		Event: &pb.ServerToClient_EnemySpawned{
			EnemySpawned: &pb.NewEnemySpawned{
				Enemy: pbEnemy,
			},
		},
	})
}

func (g *Game) UpgradeWeapon(playerID string) {
	g.Lock()
	defer g.Unlock()

	session, ok := g.Players[playerID]
	if !ok {
		log.Printf("Attempted to upgrade weapon for a non-existent player3: %s\n", playerID)
		return
	}

	player := session.Data
	weapon := player.GetEquipment().GetWeapon()

	upgradeCost := int64(float64(WeaponUpgradeBaseCost) * math.Pow(WeaponUpgradeCostMultiplier, float64(weapon.GetLevel()-1)))

	if player.GetResources().GetGold() < upgradeCost {
		log.Printf("Player %s has not enough gold to upgrade weapon. Needs %d, has %d\n", player.GetName(), upgradeCost, player.GetResources().GetGold())

		// TODO: send INSUFFICIENT GOLD message to player
		return
	}

	player.Resources.Gold -= upgradeCost
	weapon.Level++

	log.Printf("Player %s upgraded '%s' to level %d for %d gold\n", player.GetName(), weapon.GetName(), weapon.GetLevel(), upgradeCost)

	g.sendToPlayer(playerID, &pb.ServerToClient{
		Event: &pb.ServerToClient_PlayerStateUpdate{
			PlayerStateUpdate: &pb.PlayerStateUpdate{
				Player: player,
			},
		},
	})
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

func LoadAndProcessImage(filePath string, width uint, height uint) []byte {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Could not open image file: %v", err)
		return nil
	}
	defer file.Close()

	// TODO: detect the file type and use appropriate decoding logic
	img, err := webp.Decode(file)
	if err != nil {
		log.Printf("Could not decode image file '%s': %v", filePath, err)
		return nil
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)

	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, resizedImg); err != nil {
		log.Printf("Could not encode image to PNG: %v", err)
		return nil
	}

	return buffer.Bytes()
}

func (g *Game) CreateAndPrepareEnemy(level int64, imagePath string) *Enemy {
	imageBytes := LoadAndProcessImage(imagePath, 384, 384)
	hp := CalculateEnemyHp(level)
	stats := EnemyStats{
		EnemyMaxHp: hp,
		EnemyLevel: level,
	}

	name := fmt.Sprintf("Level %d Goblin", level)

	return &Enemy{
		ID:            GenerateID(),
		Name:          name,
		MaxHealth:     stats.EnemyMaxHp,
		CurrentHealth: stats.EnemyMaxHp,
		Level:         stats.EnemyLevel,
		Image:         imageBytes,
	}
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

func (g *Game) checkForLevelUp(player *pb.Player) {
	stats := player.GetStats()
	if stats.Experience >= stats.GetNextLevelExp() {
		stats.Level++

		// maybe subject to change in future
		stats.Experience = 0

		stats.NextLevelExp = int64(float64(stats.GetNextLevelExp()) * 1.5)

		log.Printf("Player %s has reached Level %d", player.GetName(), stats.GetLevel())

		// maybe send some message to the player here?
	}
}
