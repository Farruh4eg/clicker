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
	player.Id = game.GenerateID()

	player.Stats = &pb.PlayerStats{
		Level:        1,
		Experience:   0,
		NextLevelExp: 100,
	}
	player.Resources = &pb.PlayerResources{
		Gold: 0,
	}
	player.Equipment = &pb.PlayerEquipment{
		Weapon: &pb.Weapon{
			ItemId:       "starter_stick",
			Name:         "Деревянная палка",
			Level:        1,
			BaseDamage:   5.0,
			DamageGrowth: 2.0,
		},
	}

	if player == nil {
		return status.Errorf(codes.InvalidArgument, "Handshake failed: client must provide self_info in the first message")
	}

	log.Printf("Player : %s connected", player.GetName())

	updatesChan := make(chan *pb.ServerToClient, 10)

	gs.game.AddPlayer(player, updatesChan)
	defer func() {
		gs.game.RemovePlayer(player.GetId())
		log.Printf("Player with ID = %s removed from the game", player.GetId())
	}()

	go func() {
		for update := range updatesChan {
			if err := stream.Send(update); err != nil {
				log.Printf("Error sending update to player %s: %v", player.GetId(), err)
				return
			}
		}
	}()

	if len(gs.game.Enemies) == 0 {
		return status.Errorf(codes.Unavailable, "No enemies left")
	}

	currentEnemy := gs.game.Enemies[0]
	initState := &pb.ServerToClient{
		Event: &pb.ServerToClient_InitialState{
			InitialState: &pb.InitialState{
				Enemy: &pb.Enemy{
					Id:        currentEnemy.ID,
					Name:      currentEnemy.Name,
					MaxHp:     currentEnemy.MaxHealth,
					CurrentHp: currentEnemy.CurrentHealth,
					Level:     currentEnemy.Level,
					Image:     currentEnemy.Image,
				},
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
			weapon := player.GetEquipment().GetWeapon()
			damage := weapon.GetBaseDamage() + weapon.GetDamageGrowth()*float32(weapon.GetLevel()-1)
			// ID not used yet, hence it being empty
			gs.game.ApplyDamage("", float64(damage), player.GetId())

		default:
			log.Printf("Received unhandled event type %T from player %s", event, player.GetId())
		}
	}
}
