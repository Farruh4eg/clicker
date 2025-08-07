package main

import (
	"context"
	"fmt"
	"log"

	pb "clicker/gen/proto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClickerApp struct {
	stream    pb.GameService_PlayGameClient
	player    *pb.Player
	enemyName binding.String
	enemyHp   binding.String
}

func main() {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(":32228", opts...)
	if err != nil {
		log.Fatalf("Could not connect to server: %v", err)
	}
	fmt.Println("Successfully connected to grpc server ", conn.GetState())

	defer conn.Close()

	client := pb.NewGameServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.PlayGame(ctx)
	if err != nil {
		log.Fatalf("Failed to start game stream: %v", err)
	}
	log.Println("Game has started")

	myPlayer := &pb.Player{Id: 1, Name: "Farruh4eg", AttackDamage: 2.0}
	appState := &ClickerApp{
		stream:    stream,
		player:    myPlayer,
		enemyName: binding.NewString(),
		enemyHp:   binding.NewString(),
	}

	err = stream.Send(&pb.ClientToServer{
		SelfInfo: myPlayer,
	})
	if err != nil {
		log.Fatalf("Could not send self info to server: %v", err)
	}
	log.Println("Sent self info to server")

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				log.Printf("Failed to receive : %v", err)
				return
			}
			log.Printf("Got a message from server: %v", in.GetInitialState())
			if update := in.GetGameStateUpdate(); update != nil {
				log.Printf("GAME STATE UPDATE: Boss HP = %.2f, Last Attacker ID = %d", update.GetEnemyCurrentHp(), update.LastHit.GetAttackerId())
				appState.enemyHp.Set(fmt.Sprintf("%.2f", update.GetEnemyCurrentHp()))
			}
			if initState := in.GetInitialState(); initState != nil {
				log.Printf("INITIAL STATE: Boss Name = %s, HP = %.2f", initState.GetEnemy().Name, initState.Enemy.GetMaxHp())
				appState.enemyName.Set(initState.GetEnemy().GetName())
				appState.enemyHp.Set(fmt.Sprintf("%.2f", initState.GetEnemy().GetCurrentHp()))
			}
		}
	}()

	a := app.New()
	w := a.NewWindow("clicker")
	w.Resize(fyne.NewSize(600, 400))

	attackButton := widget.NewButton("Attack", func() {
		log.Println("Attacking now!")

		err := appState.stream.Send(&pb.ClientToServer{
			Event: &pb.ClientToServer_Attack{
				Attack: &pb.AttackAction{},
			},
		})
		if err != nil {
			log.Printf("Could not send attack: %v", err)
		}
	})

	enemyNameLabel := widget.NewLabelWithData(appState.enemyName)

	enemyHpLabel := widget.NewLabelWithData(appState.enemyHp)

	enemyLayout := container.NewGridWithRows(2, enemyNameLabel, enemyHpLabel, attackButton)

	w.SetContent(enemyLayout)

	w.ShowAndRun()
	log.Println("Application shutting down")
}
