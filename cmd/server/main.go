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
	game := game.NewGame()
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			game.CreateEnemy()
		}()
	}

	wg.Wait()

	game.Lock()
	fmt.Printf("Создано %d врагов\n", len(game.Enemies))
	for _, e := range game.Enemies {
		fmt.Printf("Враг ID: %s\n", e.ID)
	}
	game.Unlock()

	lis, err := net.Listen("tcp", "localhost:32228")
	if err != nil {
		log.Fatalf("Could not start listening on port :32228")
	}

	fmt.Println("Game server init")
	gameServer := server.NewGameServer(game)
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
