// Package client contains client implementation
package client

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"strconv"

	pb "clicker/gen/proto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
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
	enemyImage     binding.Bytes
}

func NewClickerApp(stream pb.GameService_PlayGameClient, player *pb.Player) *ClickerApp {
	return &ClickerApp{
		stream:         stream,
		player:         player,
		enemyName:      binding.NewString(),
		enemyCurrentHp: binding.NewFloat(),
		enemyMaxHp:     binding.NewFloat(),
		enemyImage:     binding.NewBytes(),
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
				log.Printf("Failed to receive from stream: %v", err)
				return
			}

			switch event := in.GetEvent().(type) {

			case *pb.ServerToClient_GameStateUpdate:
				update := event.GameStateUpdate
				log.Printf("GAME STATE UPDATE: Boss HP = %.2f, ID = %s", update.GetEnemyCurrentHp(), update.GetEnemyId())
				a.enemyCurrentHp.Set(update.GetEnemyCurrentHp())

			case *pb.ServerToClient_InitialState:
				initState := event.InitialState
				log.Printf("INITIAL STATE: Boss Name = %s, HP = %.2f / %.2f", initState.GetEnemy().Name, initState.GetEnemy().GetCurrentHp(), initState.GetEnemy().GetMaxHp())
				a.enemyName.Set(initState.GetEnemy().GetName())
				a.enemyCurrentHp.Set(initState.GetEnemy().GetCurrentHp())
				a.enemyMaxHp.Set(initState.GetEnemy().GetMaxHp())
				a.enemyImage.Set(initState.GetEnemy().GetImage())

			case *pb.ServerToClient_EnemySpawned:
				newEnemy := event.EnemySpawned
				log.Printf("NEW ENEMY SPAWNED: Boss Name = %s, HP = %.2f, Level = %d", newEnemy.GetEnemy().GetName(), newEnemy.GetEnemy().GetMaxHp(), newEnemy.GetEnemy().GetLevel())
				a.enemyName.Set(newEnemy.GetEnemy().GetName())
				a.enemyCurrentHp.Set(newEnemy.GetEnemy().GetMaxHp())
				a.enemyMaxHp.Set(newEnemy.GetEnemy().GetMaxHp())
				a.enemyImage.Set(newEnemy.GetEnemy().GetImage())

			default:
				log.Printf("Received an unknown event type: %T", event)
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

	enemyHpBar := widget.NewProgressBar()

	enemyHpLabel := widget.NewLabel("")
	enemyHpLabel.Alignment = fyne.TextAlignCenter

	canvasImage := &canvas.Image{}
	canvasImage.FillMode = canvas.ImageFillContain
	canvasImage.SetMinSize(fyne.NewSize(150, 150))

	imageListener := binding.NewDataListener(func() {
		imageBytes, err := a.enemyImage.Get()
		if err != nil || len(imageBytes) == 0 {
			return
		}

		go func() {
			log.Println("Decoding image")
			img, err := png.Decode(bytes.NewReader(imageBytes))
			if err != nil {
				log.Printf("Failed to decode image: %v", err)
				return
			}
			log.Println("Image decoded successfully")

			fyne.Do(func() {
				log.Println("Updating image on main thread")
				canvasImage.Image = img
				canvasImage.Refresh()
			})
		}()
	})
	a.enemyImage.AddListener(imageListener)

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
		container.NewCenter(canvasImage),
		enemyHpLabel,
		enemyHpBar,

		layout.NewSpacer(),
		attackButton,
	)

	w.SetContent(enemyLayout)

	hpListener.DataChanged()
	imageListener.DataChanged()

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
