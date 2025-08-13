package game

//
// import (
// 	pb "clicker/gen/proto"
// 	"testing"
//
// 	"github.com/stretchr/testify/assert"
// )
//
// func TestNewGame(t *testing.T) {
// 	game := NewGame()
// 	assert.NotNil(t, game)
// 	assert.NotNil(t, game.Enemies)
// 	assert.Empty(t, game.Enemies)
// 	assert.NotNil(t, game.Players)
// 	assert.Empty(t, game.Players)
// }
//
// func TestAddPlayer(t *testing.T) {
// 	game := NewGame()
// 	playerID := "player1"
// 	updateChan := make(chan *pb.ServerToClient)
//
// 	game.AddPlayer(playerID, updateChan)
//
// 	assert.Contains(t, game.Players, playerID)
// 	assert.Equal(t, updateChan, game.Players[playerID])
// }
//
// func TestRemovePlayer(t *testing.T) {
// 	game := NewGame()
// 	playerID := "player1"
// 	updateChan := make(chan *pb.ServerToClient, 1)
//
// 	game.AddPlayer(playerID, updateChan)
// 	game.RemovePlayer(playerID)
//
// 	assert.NotContains(t, game.Players, playerID)
//
// 	// Check if the channel is closed
// 	_, ok := <-updateChan
// 	assert.False(t, ok)
// }
//
// func TestBroadcast(t *testing.T) {
// 	game := NewGame()
// 	player1ID := "player1"
// 	player2ID := "player2"
// 	updateChan1 := make(chan *pb.ServerToClient, 1)
// 	updateChan2 := make(chan *pb.ServerToClient, 1)
//
// 	game.AddPlayer(player1ID, updateChan1)
// 	game.AddPlayer(player2ID, updateChan2)
//
// 	message := &pb.ServerToClient{
// 		PlayerJoined: &pb.PlayerJoined{
// 			Player: &pb.Player{Id: "player3"},
// 		},
// 	}
// 	game.broadcast(message)
//
// 	receivedMsg1 := <-updateChan1
// 	receivedMsg2 := <-updateChan2
//
// 	assert.Equal(t, message, receivedMsg1)
// 	assert.Equal(t, message, receivedMsg2)
// }
//
// func TestCreateEnemy(t *testing.T) {
// 	game := NewGame()
// 	enemyStats := EnemyStats{
// 		EnemyMaxHp: 100.0,
// 	}
// 	enemy := game.CreateEnemy(enemyStats, "Retard", nil)
//
// 	assert.NotNil(t, enemy)
// 	assert.NotEmpty(t, enemy.ID)
// 	assert.Equal(t, "Retard", enemy.Name)
// 	assert.Equal(t, 100.0, enemy.MaxHealth)
// 	assert.Equal(t, 100.0, enemy.CurrentHealth)
// 	assert.Nil(t, enemy.Image)
// 	assert.Contains(t, game.Enemies, enemy)
// }
//
// func TestApplyDamage(t *testing.T) {
// 	t.Run("when enemy survives", func(t *testing.T) {
// 		game := NewGame()
// 		initialHp := 100.0
// 		damage := 25.0
//
// 		game.CreateEnemyForLevel(1)
//
// 		enemyBeforeAttack := game.Enemies[0]
//
// 		enemyBeforeAttack.MaxHealth = initialHp
// 		enemyBeforeAttack.CurrentHealth = initialHp
//
// 		game.ApplyDamage("", damage, "")
// 		assert.Len(t, game.Enemies, 1, "Enemy should not be removed from the game")
//
// 		enemyAfterAttack := game.Enemies[0]
//
// 		expectedHp := initialHp - damage
// 		assert.Equal(t, expectedHp, enemyAfterAttack.CurrentHealth, "Enemy health should be reduced be the damage amount")
// 	})
//
// 	t.Run("when enemy dies", func(t *testing.T) {
// 		game := NewGame()
// 		initialHp := 100.0
// 		lethalDamage := 125.0
//
// 		game.CreateEnemy(EnemyStats{EnemyMaxHp: initialHp, EnemyLevel: 1}, "GoblinToDie", nil)
// 		game.CreateEnemy(EnemyStats{EnemyMaxHp: 150.0, EnemyLevel: 2}, "NextGoblin", nil)
//
// 		enemyThatWillSurvive := game.Enemies[1]
//
// 		assert.Len(t, game.Enemies, 2, "Should start with two enemies")
//
// 		game.ApplyDamage("", lethalDamage, "")
//
// 		assert.Len(t, game.Enemies, 1, "One enemy should be removed from the game")
//
// 		remainingEnemy := game.Enemies[0]
// 		assert.Equal(t, enemyThatWillSurvive.ID, remainingEnemy.ID, "The correct enemy should remain in the game")
// 	})
//
// 	t.Run("when the last enemy dies", func(t *testing.T) {
// 		game := NewGame()
// 		game.CreateEnemy(EnemyStats{EnemyMaxHp: 50.0, EnemyLevel: 1}, "LastOne", nil)
//
// 		game.ApplyDamage("", 100.0, "")
//
// 		assert.Empty(t, game.Enemies, "Enemies slice should be empty after the last enemy dies")
// 	})
//
// 	t.Run("when attacking no enemies", func(t *testing.T) {
// 		game := NewGame()
// 		assert.Empty(t, game.Enemies, "Game should start with no enemies")
//
// 		assert.NotPanics(t, func() {
// 			game.ApplyDamage("", 100.0, "")
// 		}, "ApplyDamage should not panic when there are no enemies")
// 	})
// }
