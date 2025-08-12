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

	const numEnemies = 10
	enemies := make([]*game.Enemy, numEnemies)

	for i := 0; i < numEnemies; i++ {
		level := int64(i + 1)
		wg.Add(1)

		go func(lvl int64, index int) {
			defer wg.Done()

			// TODO: change the hardcoded image value
			imagePath := "static/images/goblin.webp"

			preparedEnemy := gameInstance.CreateAndPrepareEnemy(lvl, imagePath)
			enemies[index] = preparedEnemy
		}(level, i)
	}

	wg.Wait()
	log.Println("All assets loaded and enemies are ready")
	gameInstance.Enemies = enemies

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

	<-closeChan
	log.Println("Shutting down the server")
	grpcServer.GracefulStop()
	log.Println("Server gracefully stopped :)")
}
