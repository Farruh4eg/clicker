// Package client contains client implementation
package client

import (
	"bytes"
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
	stream  pb.GameService_PlayGameClient
	fyneApp fyne.App
	mainWin fyne.Window
	player  *pb.Player

	enemyName      binding.String
	enemyCurrentHp binding.Float
	enemyMaxHp     binding.Float
	enemyImage     binding.Bytes

	playerGold           binding.Int
	playerLevel          binding.Int
	playerExp            binding.Int
	playerExpToNextLevel binding.Int

	weaponName   binding.String
	weaponDamage binding.Float
	weaponLevel  binding.Int

	otherPlayers binding.StringList
}

func NewClickerApp(stream pb.GameService_PlayGameClient, player *pb.Player) *ClickerApp {
	a := &ClickerApp{
		stream:  stream,
		player:  player,
		fyneApp: app.New(),

		enemyName:      binding.NewString(),
		enemyCurrentHp: binding.NewFloat(),
		enemyMaxHp:     binding.NewFloat(),
		enemyImage:     binding.NewBytes(),

		playerGold:           binding.NewInt(),
		playerLevel:          binding.NewInt(),
		playerExp:            binding.NewInt(),
		playerExpToNextLevel: binding.NewInt(),

		weaponName:   binding.NewString(),
		weaponDamage: binding.NewFloat(),
		weaponLevel:  binding.NewInt(),

		otherPlayers: binding.NewStringList(),
	}
	a.mainWin = a.fyneApp.NewWindow("Clicker")
	return a
}

func (a *ClickerApp) Run() {
	err := a.stream.Send(&pb.ClientToServer{
		Event: &pb.ClientToServer_SelfInfo{SelfInfo: a.player},
	})
	if err != nil {
		log.Fatalf("Could not send self info: %v", err)
	}
	log.Println("Sent self info to server")

	go a.listenForServerUpdates()

	a.mainWin.SetContent(a.createContent())
	a.mainWin.Resize(fyne.NewSize(800, 600))
	a.mainWin.ShowAndRun()
	log.Println("Application shutting down")
}

func (a *ClickerApp) updatePlayerData(playerData *pb.Player) {
	if playerData == nil {
		return
	}
	a.playerGold.Set(int(playerData.GetResources().GetGold()))
	a.playerLevel.Set(int(playerData.GetStats().GetLevel()))
	a.playerExp.Set(int(playerData.GetStats().GetExperience()))
	a.playerExpToNextLevel.Set(int(playerData.GetStats().GetNextLevelExp()))
	if weapon := playerData.GetEquipment().GetWeapon(); weapon != nil {
		a.weaponName.Set(weapon.GetName())
		a.weaponLevel.Set(int(weapon.GetLevel()))
		damage := weapon.GetBaseDamage() + weapon.GetDamageGrowth()*float32(weapon.GetLevel()-1)
		a.weaponDamage.Set(float64(damage))
	}
}

func (a *ClickerApp) updateEnemyData(enemyData *pb.Enemy) {
	if enemyData == nil {
		return
	}
	a.enemyName.Set(enemyData.GetName())
	a.enemyCurrentHp.Set(enemyData.GetCurrentHp())
	a.enemyMaxHp.Set(enemyData.GetMaxHp())
	a.enemyImage.Set(enemyData.GetImage())
}

func (a *ClickerApp) listenForServerUpdates() {
	for {
		in, err := a.stream.Recv()
		if err != nil {
			log.Printf("Failed to receive from stream: %v", err)
			return
		}

		fyne.Do(func() {
			switch event := in.GetEvent().(type) {

			case *pb.ServerToClient_Welcome:
				playerData := event.Welcome.GetPlayer()
				log.Printf("WELCOME! I am %s with ID %s", playerData.GetName(), playerData.GetId())
				a.player = playerData
				a.updatePlayerData(playerData)

			case *pb.ServerToClient_InitialState:
				initState := event.InitialState
				log.Printf("INITIAL STATE: Got enemy and %d players.", len(initState.GetPlayers()))
				a.updateEnemyData(initState.GetEnemy())

				var otherPlayerNames []string
				for _, p := range initState.GetPlayers() {
					if p.GetId() != a.player.GetId() {
						otherPlayerNames = append(otherPlayerNames, p.GetName())
					}
				}
				a.otherPlayers.Set(otherPlayerNames)

			case *pb.ServerToClient_PlayerStateUpdate:
				playerData := event.PlayerStateUpdate.GetPlayer()
				if playerData.GetId() == a.player.GetId() {
					log.Printf("My state updated: Gold=%d, Lvl=%d", playerData.GetResources().GetGold(), playerData.GetStats().GetLevel())
					a.updatePlayerData(playerData)
				}

			case *pb.ServerToClient_PlayerJoined:
				newPlayer := event.PlayerJoined.GetPlayer()
				log.Printf("Player %s joined the game", newPlayer.GetName())
				a.otherPlayers.Append(newPlayer.GetName())

			case *pb.ServerToClient_PlayerLeft:
				leftPlayerID := event.PlayerLeft.GetPlayerId()
				log.Printf("Player with ID %s left the game", leftPlayerID)
				// TODO: delete by id, not name

			case *pb.ServerToClient_GameStateUpdate:
				update := event.GameStateUpdate
				a.enemyCurrentHp.Set(update.GetEnemyCurrentHp())

			case *pb.ServerToClient_EnemySpawned:
				newEnemy := event.EnemySpawned.GetEnemy()
				a.updateEnemyData(newEnemy)

			default:
				log.Printf("Received an unknown event type: %T", event)
			}
		})
	}
}

func (a *ClickerApp) createContent() fyne.CanvasObject {
	enemyNameLabel := widget.NewLabelWithData(a.enemyName)
	enemyHpBar := widget.NewProgressBar()
	a.enemyCurrentHp.AddListener(binding.NewDataListener(func() {
		cur, _ := a.enemyCurrentHp.Get()
		max, _ := a.enemyMaxHp.Get()
		if max > 0 {
			enemyHpBar.SetValue(cur / max)
		}
	}))
	enemyImage := &canvas.Image{FillMode: canvas.ImageFillContain}
	enemyImage.SetMinSize(fyne.NewSize(256, 256))
	a.enemyImage.AddListener(binding.NewDataListener(func() {
		enemyImageBytes, _ := a.enemyImage.Get()
		if len(enemyImageBytes) == 0 {
			return
		}
		go func() {
			img, _ := png.Decode(bytes.NewReader(enemyImageBytes))
			fyne.Do(func() { enemyImage.Image = img; enemyImage.Refresh() })
		}()
	}))
	attackButton := widget.NewButton("Attack", func() {
		a.stream.Send(&pb.ClientToServer{Event: &pb.ClientToServer_Attack{Attack: &pb.AttackAction{}}})
	})

	enemyBox := container.NewVBox(
		container.NewCenter(enemyNameLabel),
		container.NewCenter(enemyImage),
		enemyHpBar,
		layout.NewSpacer(),
		attackButton,
	)

	playerGoldLabel := widget.NewLabelWithData(binding.IntToStringWithFormat(a.playerGold, "Золото: %d"))
	playerLevelLabel := widget.NewLabelWithData(binding.IntToStringWithFormat(a.playerLevel, "Уровень: %d"))
	playerExpBar := widget.NewProgressBar()
	a.playerExp.AddListener(binding.NewDataListener(func() {
		cur, _ := a.playerExp.Get()
		next, _ := a.playerExpToNextLevel.Get()
		if next > 0 {
			playerExpBar.SetValue(float64(cur) / float64(next))
		}
	}))
	weaponNameLabel := widget.NewLabelWithData(a.weaponName)
	weaponStatsLabel := widget.NewLabelWithData(binding.FloatToStringWithFormat(a.weaponDamage, "Урон: %.1f"))
	upgradeWeaponButton := widget.NewButton("Улучшить", func() {
		a.stream.Send(&pb.ClientToServer{Event: &pb.ClientToServer_UpgradeWeapon{UpgradeWeapon: &pb.UpgradeWeaponRequest{}}})
	})

	playerBox := container.NewVBox(
		widget.NewLabelWithStyle("Персонаж", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		playerGoldLabel,
		playerLevelLabel,
		playerExpBar,
		container.NewHSplit(container.NewVBox(weaponNameLabel, weaponStatsLabel), upgradeWeaponButton),
	)

	othersList := widget.NewListWithData(
		a.otherPlayers,
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		},
	)

	othersBox := container.NewBorder(
		widget.NewLabelWithStyle("Онлайн", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		othersList,
	)

	leftPanel := container.NewVSplit(playerBox, othersBox)
	leftPanel.Offset = 0.6

	mainLayout := container.NewHSplit(leftPanel, enemyBox)
	mainLayout.Offset = 0.3

	return mainLayout
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
