// Package client contains client implementation
package client

import (
	"fmt"
	"log"

	pb "clicker/gen/proto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ClickerApp struct {
	stream    pb.GameService_PlayGameClient
	player    *pb.Player
	enemyName binding.String
	enemyHp   binding.String
}

func NewClickerApp(stream pb.GameService_PlayGameClient, player *pb.Player) *ClickerApp {
	return &ClickerApp{
		stream:    stream,
		player:    player,
		enemyName: binding.NewString(),
		enemyHp:   binding.NewString(),
	}
}

func (a *ClickerApp) Run() {
	err := a.stream.Send(&pb.ClientToServer{
		SelfInfo: a.player,
	})
	if err != nil {
		log.Fatalf("Could not send self info to server: %v", err)
	}
	log.Println("Sent self info to server")

	go func() {
		for {
			in, err := a.stream.Recv()
			if err != nil {
				log.Printf("Failed to receive : %v", err)
				return
			}
			log.Printf("Got a message from server: %v", in.GetInitialState())
			if update := in.GetGameStateUpdate(); update != nil {
				log.Printf("GAME STATE UPDATE: Boss HP = %.2f, Last Attacker ID = %s", update.GetEnemyCurrentHp(), update.LastHit.GetAttackerId())
				a.enemyHp.Set(fmt.Sprintf("%.2f", update.GetEnemyCurrentHp()))
			}
			if initState := in.GetInitialState(); initState != nil {
				log.Printf("INITIAL STATE: Boss Name = %s, HP = %.2f", initState.GetEnemy().Name, initState.Enemy.GetMaxHp())
				a.enemyName.Set(initState.GetEnemy().GetName())
				a.enemyHp.Set(fmt.Sprintf("%.2f", initState.GetEnemy().GetCurrentHp()))
			}
		}
	}()

	fyneApp := app.New()
	w := fyneApp.NewWindow("clicker")
	w.Resize(fyne.NewSize(600, 400))

	attackButton := widget.NewButton("Attack", func() {
		log.Println("Attacking now!")

		err := a.stream.Send(&pb.ClientToServer{
			Event: &pb.ClientToServer_Attack{
				Attack: &pb.AttackAction{},
			},
		})
		if err != nil {
			log.Printf("Could not send attack: %v", err)
		}
	})

	enemyNameLabel := widget.NewLabelWithData(a.enemyName)

	enemyHpLabel := widget.NewLabelWithData(a.enemyHp)

	enemyLayout := container.NewGridWithRows(3, enemyNameLabel, enemyHpLabel, attackButton)

	w.SetContent(enemyLayout)

	w.ShowAndRun()
	log.Println("Application shutting down")
}
