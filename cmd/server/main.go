package main

import (
	"fmt"
	"image"
	"sync"
)

type Game struct {
	sync.Mutex
	LastEnemyID int64
	enemies     []*Enemy
}

func NewGame() *Game {
	return &Game{
		enemies: make([]*Enemy, 0, 10),
	}
}

type Enemy struct {
	ID            int64
	Name          string
	MaxHealth     float64
	CurrentHealth float64
	Image         image.Image
}

func (g *Game) CreateEnemy() *Enemy {
	g.Lock()
	defer g.Unlock()

	g.LastEnemyID++
	newEnemy := &Enemy{
		ID:            g.LastEnemyID,
		Name:          "Retard",
		MaxHealth:     100.0,
		CurrentHealth: 100.0,
		Image:         nil,
	}

	g.enemies = append(g.enemies, newEnemy)

	return newEnemy
}

func main() {
	game := NewGame()

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			game.CreateEnemy()
		}()
	}

	wg.Wait()

	game.Lock()
	fmt.Printf("Создано %d врагов\n", len(game.enemies))
	for _, e := range game.enemies {
		fmt.Printf("Враг ID: %d\n", e.ID)
	}
	game.Unlock()
}
