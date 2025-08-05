package main

import (
	"context"
	"fmt"
	"io"
	"log"

	pb "clicker/gen/proto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/grpc"
)

func main() {
	a := app.New()
	w := a.NewWindow("clicker")

	w.Resize(fyne.NewSize(600, 400))
	w.SetContent(widget.NewButton("Attack", func() {
		fmt.Println("you attacked, bozo")
	}))

	var opts []grpc.DialOption

	conn, err := grpc.NewClient(":8080", opts...)
	if err != nil {
		log.Fatalf("Could not connect to server: %v", err)
	}

	defer conn.Close()

	client := pb.NewGameServiceClient(conn)
	stream, err := client.PlayGame(context.Background())
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive : %v", err)
			}
			log.Printf("Got a message from server: %v", in.GetInitialState())
		}
	}()

	w.ShowAndRun()
}
