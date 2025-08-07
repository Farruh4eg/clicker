// Package server contains server implementation
package server

import (
	pb "clicker/gen/proto"
	"clicker/pkg/game"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GameServer struct {
	pb.UnimplementedGameServiceServer
	game *game.Game
}

func NewGameServer(game *game.Game) *GameServer {
	return &GameServer{game: game}
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
	currentEnemy := gs.game.Enemies[0]
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
