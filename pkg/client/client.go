// Package client contains client implementation
package client

import (
	"fmt"
	"log"
	"strconv"

	pb "clicker/gen/proto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type ClickerApp struct {
	stream         pb.GameService_PlayGameClient
	player         *pb.Player
	enemyName      binding.String
	enemyCurrentHp binding.Float
	enemyMaxHp     binding.Float
	enemyHpBar     widget.ProgressBar
}

func NewClickerApp(stream pb.GameService_PlayGameClient, player *pb.Player) *ClickerApp {
	return &ClickerApp{
		stream:         stream,
		player:         player,
		enemyName:      binding.NewString(),
		enemyCurrentHp: binding.NewFloat(),
		enemyMaxHp:     binding.NewFloat(),
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
				a.enemyCurrentHp.Set(update.GetEnemyCurrentHp())
			}
			if initState := in.GetInitialState(); initState != nil {
				log.Printf("INITIAL STATE: Boss Name = %s, HP = %.2f", initState.GetEnemy().Name, initState.Enemy.GetMaxHp())
				a.enemyName.Set(initState.GetEnemy().GetName())
				a.enemyCurrentHp.Set(initState.GetEnemy().GetCurrentHp())
				a.enemyMaxHp.Set(initState.GetEnemy().GetMaxHp())
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

	// TODO: implement enemy hp as a slider
	enemyHpBar := widget.NewProgressBar()

	enemyHpLabel := widget.NewLabel("")
	enemyHpLabel.Alignment = fyne.TextAlignCenter

	hpListener := binding.NewDataListener(func() {
		current, errCurrent := a.enemyCurrentHp.Get()
		max, errMax := a.enemyMaxHp.Get()

		if errCurrent != nil || errMax != nil {
			log.Printf("Error getting current/max hp values: \n%v\n%v", errCurrent, errMax)
			return
		}

		hpText := fmt.Sprintf("%.0f / %.0f", current, max)
		enemyHpLabel.SetText(hpText)

		if max > 0 {
			normalizedHp := current / max
			enemyHpBar.SetValue(normalizedHp)
		}
	})

	a.enemyCurrentHp.AddListener(hpListener)
	a.enemyMaxHp.AddListener(hpListener)

	enemyLayout := container.NewVBox(
		container.NewCenter(enemyNameLabel),
		// TODO: insert image here

		enemyHpBar,

		layout.NewSpacer(),
		attackButton,
	)

	w.SetContent(enemyLayout)

	hpListener.DataChanged()

	w.ShowAndRun()
	log.Println("Application shutting down")
}

func BindingStrToFloat64(s binding.String) float64 {
	data, err := s.Get()
	if err != nil {
		log.Printf("Could not get binding data: %v", err)
	}

	dataAsFloat, err := strconv.ParseFloat(data, 64)
	if err != nil {
		log.Printf("Could not parse data as float: %v", err)
	}

	return dataAsFloat
}
