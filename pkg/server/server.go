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
		return err
	}

	selfInfo := initialReq.GetSelfInfo()
	if selfInfo == nil {
		return status.Errorf(codes.InvalidArgument, "Handshake failed: client must provide self_info")
	}

	player := game.InitializePlayer(selfInfo.GetName())
	log.Printf("Player '%s' connecting with generated ID: %s", player.GetName(), player.GetId())

	updatesChan := make(chan *pb.ServerToClient, 10)
	gs.game.AddPlayer(player, updatesChan)
	defer func() {
		gs.game.RemovePlayer(player.GetId())
		gs.game.Broadcast(&pb.ServerToClient{
			Event: &pb.ServerToClient_PlayerLeft{
				PlayerLeft: &pb.PlayerLeft{
					PlayerId: player.GetId(),
				},
			},
		}, "")
		log.Printf("Player %s (ID: %s) disconnected\n", player.GetName(), player.GetId())
	}()

	go func() {
		for update := range updatesChan {
			if err := stream.Send(update); err != nil {
				log.Printf("Error sending update to player %s: %v", player.GetId(), err)
				return
			}
		}
	}()

	welcomeMsg := &pb.ServerToClient{
		Event: &pb.ServerToClient_Welcome{
			Welcome: &pb.Welcome{
				Player: player,
			},
		},
	}
	updatesChan <- welcomeMsg
	log.Printf("Sent Welcome message to %s\n", player.GetName())

	currentEnemy := gs.game.GetCurrentEnemy()
	if currentEnemy == nil {
		return status.Errorf(codes.Unavailable, "No enemies in the game")
	}
	allPlayers := gs.game.GetAllPlayers()

	initState := &pb.ServerToClient{
		Event: &pb.ServerToClient_InitialState{
			InitialState: &pb.InitialState{
				Enemy:   currentEnemy.ToProto(),
				Players: allPlayers,
			},
		},
	}
	updatesChan <- initState
	log.Printf("Sent initial state to player %s\n", player.GetId())

	playerJoinedMsg := &pb.ServerToClient{
		Event: &pb.ServerToClient_PlayerJoined{
			PlayerJoined: &pb.PlayerJoined{
				Player: player,
			},
		},
	}
	gs.game.Broadcast(playerJoinedMsg, player.GetId())

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Printf("Stream for player %s closed: %v", player.GetId(), err)
			return err
		}

		switch req.GetEvent().(type) {
		case *pb.ClientToServer_Attack:
			weapon := player.GetEquipment().GetWeapon()
			damage := weapon.GetBaseDamage() + weapon.GetDamageGrowth()*float32(weapon.GetLevel()-1)
			gs.game.ApplyDamage("", float64(damage), player.GetId())

		case *pb.ClientToServer_UpgradeWeapon:
			gs.game.UpgradeWeapon(player.GetId())

		default:
			log.Printf("Received unhandled event type from player %s\n", player.GetName())
		}
	}
}
