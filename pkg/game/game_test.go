package game

import (
	pb "clicker/gen/proto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	game := NewGame()
	assert.NotNil(t, game)
	assert.NotNil(t, game.Enemies)
	assert.Empty(t, game.Enemies)
	assert.NotNil(t, game.Players)
	assert.Empty(t, game.Players)
}

func TestAddPlayer(t *testing.T) {
	game := NewGame()
	playerID := "player1"
	updateChan := make(chan *pb.ServerToClient)

	game.AddPlayer(playerID, updateChan)

	assert.Contains(t, game.Players, playerID)
	assert.Equal(t, updateChan, game.Players[playerID])
}

func TestRemovePlayer(t *testing.T) {
	game := NewGame()
	playerID := "player1"
	updateChan := make(chan *pb.ServerToClient, 1)

	game.AddPlayer(playerID, updateChan)
	game.RemovePlayer(playerID)

	assert.NotContains(t, game.Players, playerID)

	// Check if the channel is closed
	_, ok := <-updateChan
	assert.False(t, ok)
}

func TestBroadcast(t *testing.T) {
	game := NewGame()
	player1ID := "player1"
	player2ID := "player2"
	updateChan1 := make(chan *pb.ServerToClient, 1)
	updateChan2 := make(chan *pb.ServerToClient, 1)

	game.AddPlayer(player1ID, updateChan1)
	game.AddPlayer(player2ID, updateChan2)

	message := &pb.ServerToClient{
		PlayerJoined: &pb.PlayerJoined{
			Player: &pb.Player{Id: "player3"},
		},
	}
	game.Broadcast(message)

	receivedMsg1 := <-updateChan1
	receivedMsg2 := <-updateChan2

	assert.Equal(t, message, receivedMsg1)
	assert.Equal(t, message, receivedMsg2)
}

func TestCreateEnemy(t *testing.T) {
	game := NewGame()
	enemyStats := EnemyStats{
		EnemyMaxHp: 100.0,
	}
	enemy := game.CreateEnemy(enemyStats, "Retard", nil)

	assert.NotNil(t, enemy)
	assert.NotEmpty(t, enemy.ID)
	assert.Equal(t, "Retard", enemy.Name)
	assert.Equal(t, 100.0, enemy.MaxHealth)
	assert.Equal(t, 100.0, enemy.CurrentHealth)
	assert.Nil(t, enemy.Image)
	assert.Contains(t, game.Enemies, enemy)
}

func TestApplyDamage(t *testing.T) {
	game := NewGame()
	enemyStats := EnemyStats{
		EnemyMaxHp: 100.0,
	}
	enemy := game.CreateEnemy(enemyStats, "Retard", nil)

	damage := 25.0

	updatedEnemy, err := game.ApplyDamage(enemy.ID, damage)

	assert.NoError(t, err)
	assert.NotNil(t, updatedEnemy)
	assert.Equal(t, enemy.MaxHealth-damage, updatedEnemy.CurrentHealth)
}
