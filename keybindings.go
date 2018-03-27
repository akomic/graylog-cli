package main

import (
	"github.com/jroimartin/gocui"
)

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, cursorPgDown); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, cursorPgUp); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlT, gocui.ModNone, pauseLogs); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlL, gocui.ModNone, clearLogs); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlX, gocui.ModNone, closeMsg); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("streams", gocui.KeyEnter, gocui.ModNone, switchStream); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("fields", gocui.KeyEnter, gocui.ModNone, submitFieldFilter); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("console", gocui.KeyEnter, gocui.ModNone, submitSearch); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("logs", gocui.KeyEnter, gocui.ModNone, processLogLine); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("msg", gocui.KeyEnter, gocui.ModNone, applyFilter); err != nil {
		log.Panicln(err)
	}
	return nil
}
