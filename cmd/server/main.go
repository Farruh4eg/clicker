package main

import (
	pb "clicker/gen/proto"
	"fmt"
	"image"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Game struct {
	sync.Mutex
	LastEnemyID int64
	enemies     []*Enemy
	players     map[int64]chan *pb.ServerToClient // player id -> his channel
}

type GameServer struct {
	pb.UnimplementedGameServiceServer
	game *Game
}

func (g *Game) AddPlayer(playerID int64, updateChan chan *pb.ServerToClient) {
	g.Lock()
	defer g.Unlock()
	if g.players == nil {
		g.players = make(map[int64]chan *pb.ServerToClient)
	}
	g.players[playerID] = updateChan
}

func (g *Game) RemovePlayer(playerID int64) {
	g.Lock()
	defer g.Unlock()
	if channel, ok := g.players[playerID]; ok {
		delete(g.players, playerID)
		close(channel)
	}
}

func (g *Game) Broadcast(msg *pb.ServerToClient) {
	g.Lock()
	defer g.Unlock()

	for id, channel := range g.players {
		select {
		case channel <- msg:
		default:
			log.Printf("Player %d update channel is full. Message dropped", id)
		}
	}
}

func (gs *GameServer) PlayGame(stream pb.GameService_PlayGameServer) error {
	initialReq, err := stream.Recv()
	if err != nil {
		log.Printf("Failed to receive init handshake: %v", err)
	}

	player := initialReq.GetSelfInfo()
	if player == nil {
		return status.Errorf(codes.InvalidArgument, "Handshake failed: client must provide self_info in the first message")
	}

	log.Printf("Player : %s (ID: %d) connected", player.GetName(), player.GetId())

	updatesChan := make(chan *pb.ServerToClient, 10)
	gs.game.AddPlayer(player.GetId(), updatesChan)
	defer func() {
		gs.game.RemovePlayer(player.GetId())
		log.Printf("Player with ID = %d removed from the game", player.GetId())
	}()

	go func() {
		for update := range updatesChan {
			if err := stream.Send(update); err != nil {
				log.Printf("Error sending update to player %d: %v", player.GetId(), err)
			}
		}
	}()

	// TODO: send init state to player
	currentEnemy := gs.game.enemies[0]
	initState := &pb.ServerToClient{
		InitialState: &pb.InitialState{
			Enemy: &pb.Enemy{
				Id:        currentEnemy.ID,
				Name:      currentEnemy.Name,
				MaxHp:     currentEnemy.MaxHealth,
				CurrentHp: currentEnemy.CurrentHealth,
				Image:     nil,
			},
		},
	}

	updatesChan <- initState
	log.Println("Initial state sent?")

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		switch event := req.GetEvent().(type) {

		case *pb.ClientToServer_Attack:
			log.Printf("Player %s attacked", player.GetName())

			gs.game.ApplyDamage(currentEnemy.ID, player.GetAttackDamage())
			hitInfo := &pb.HitInfo{
				AttackerId:  player.GetId(),
				DamageDealt: player.GetAttackDamage(),
			}
			serverUpdate := &pb.ServerToClient{
				GameStateUpdate: &pb.GameStateUpdate{
					EnemyCurrentHp: currentEnemy.CurrentHealth,
					EnemyId:        currentEnemy.ID,
					LastHit:        hitInfo,
				},
			}
			gs.game.Broadcast(serverUpdate)

		default:
			log.Printf("Received unhandled event type %T from player %d", event, player.GetId())
		}
	}
}

func NewGame() *Game {
	return &Game{
		enemies: make([]*Enemy, 0, 10),
		players: make(map[int64]chan *pb.ServerToClient),
	}
}

type Enemy struct {
	ID            int64
	Name          string
	MaxHealth     float64
	CurrentHealth float64
	Image         image.Image
	// some fine grained mutex for future generations, maybe
	// sync.Mutex
}

func (g *Game) ApplyDamage(enemyID int64, incomingDamage float64) (*Enemy, error) {
	g.Lock()
	defer g.Unlock()
	// calculate enemy armor and resistance values here in future maybe?
	// just substract damage for now
	// also, TODO: find enemy by id
	enemy := g.enemies[0]
	enemy.CurrentHealth -= incomingDamage

	if enemy.CurrentHealth <= 0 {
		// TODO: destroy the enemy, spawn a new one, award xp, gold, hot wife
	}
	return enemy, nil
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
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, syscall.SIGINT, syscall.SIGTERM)

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

	lis, err := net.Listen("tcp", "localhost:32228")
	if err != nil {
		log.Fatalf("Could not start listening on port :32228")
	}

	fmt.Println("Game server init")
	gameServer := &GameServer{game: game}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterGameServiceServer(grpcServer, gameServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Error while listening: %v", err)
		}
	}()
	fmt.Println("Game server init successful. Now serving")

	// main game loop here?

	<-closeChan
	log.Println("Shutting down the server")
	grpcServer.GracefulStop()
	log.Println("Server gracefully stopped :)")
}
