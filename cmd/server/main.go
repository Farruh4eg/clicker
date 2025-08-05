package main

import (
	pb "clicker/gen/proto"
	"fmt"
	"image"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type Game struct {
	sync.Mutex
	LastEnemyID int64
	enemies     []*Enemy
}

type GameServer struct {
	pb.UnimplementedGameServiceServer
}

func (gs *GameServer) PlayGame(pb.GameService_PlayGameServer) (*pb.ServerToClient, error) {
	return nil, nil
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
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Could not start listening on port :8080")
	}

	gameServer := &GameServer{}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterGameServiceServer(grpcServer, gameServer)
	grpcServer.Serve(lis)

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
