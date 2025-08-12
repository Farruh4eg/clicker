package main

import (
	pb "clicker/gen/proto"
	"clicker/pkg/game"
	"clicker/pkg/server"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc"
)

func main() {
	gameInstance := game.NewGame()
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Creating enemies and loading assets...")
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		level := int64(i + 1)

		wg.Add(1)

		go func(lvl int64) {
			defer wg.Done()

			// TODO: change the hardcoded image value
			imagePath := fmt.Sprintf("../../static/images/goblin.webp")

			imageBytes := game.LoadImageAsPNG(imagePath)

			hp := game.CalculateEnemyHp(lvl)
			stats := game.EnemyStats{
				EnemyMaxHp: hp,
				EnemyLevel: lvl,
			}
			name := fmt.Sprintf("Level %d Goblin", lvl)

			gameInstance.CreateEnemy(stats, name, imageBytes)
		}(level)
	}

	wg.Wait()
	log.Println("All assets loaded and enemies are ready")

	gameInstance.Lock()
	fmt.Printf("Создано %d врагов\n", len(gameInstance.Enemies))
	for _, e := range gameInstance.Enemies {
		fmt.Printf("Враг ID: %s\nLevel = %d\nMax HP = %.2f", e.ID, e.Level, e.MaxHealth)
	}
	gameInstance.Unlock()

	lis, err := net.Listen("tcp", "localhost:32228")
	if err != nil {
		log.Fatalf("Could not start listening on port :32228")
	}

	fmt.Println("Game server init")
	gameServer := server.NewGameServer(gameInstance)
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterGameServiceServer(grpcServer, gameServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Error while listening: %v", err)
		}
	}()
	fmt.Println("Game server init successful. Now serving")

	// main game loop here?

	<-closeChan
	log.Println("Shutting down the server")
	grpcServer.GracefulStop()
	log.Println("Server gracefully stopped :)")
}
