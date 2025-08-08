package main

import (
	"clicker/pkg/client"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	pb "clicker/gen/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

	grpcClient := pb.NewGameServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := grpcClient.PlayGame(ctx)
	if err != nil {
		log.Fatalf("Failed to start game stream: %v", err)
	}
	log.Println("Game has started")

	randomID, err := rand.Int(rand.Reader, big.NewInt(150))
	if err != nil {
		log.Fatalf("Could not create random id: %v", err)
	}

	randomName := rand.Text()

	myPlayer := &pb.Player{Id: randomID.Int64(), Name: randomName, AttackDamage: 2.0}
	app := client.NewClickerApp(stream, myPlayer)
	app.Run()
}
