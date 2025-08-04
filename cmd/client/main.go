package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("clicker")

	w.Resize(fyne.NewSize(600, 400))
	w.SetContent(widget.NewButton("Attack", func() {
		fmt.Println("you attacked, bozo")
	}))

	a.Run()
}
