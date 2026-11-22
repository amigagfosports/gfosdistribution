// GFOS Imager, the app: a real window on Windows, macOS and Linux (no browser, no web page).
// This first file only proves the window and the build; the writer comes across from imager/ next.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("org.amigagfosports.gfosimager")
	w := a.NewWindow("GFOS Imager")
	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("GFOS Imager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("The app's window is up. The image list, drive list and writer come next."),
		widget.NewButton("Close", func() { a.Quit() }),
	))
	w.Resize(fyne.NewSize(720, 480))
	w.ShowAndRun()
}
