package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"strings"
)

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil && v.Name() != "console" {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			} else {
				scrollView(v, 1)
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil && v.Name() != "console" {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			} else {
				scrollView(v, -1)
			}
		}
	}
	return nil
}

func cursorPgDown(g *gocui.Gui, v *gocui.View) error {
	pageSize := 20
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+pageSize); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+pageSize); err != nil {
				return err
			} else {
				scrollView(v, 20)
			}
		}
	}
	return nil
}

func cursorPgUp(g *gocui.Gui, v *gocui.View) error {
	pageSize := 20
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if oy < pageSize {
			pageSize = oy
		}
		if err := v.SetCursor(cx, cy-pageSize); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-pageSize); err != nil {
				return err
			} else {
				scrollView(v, -pageSize)
			}
		}
	}
	return nil
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	nextIndex := (active + 1) % len(viewArr)
	name := viewArr[nextIndex]

	if _, err := setCurrentViewOnTop(g, name); err != nil {
		return err
	}

	g.Cursor = true

	active = nextIndex
	return nil
}

func pauseLogs(g *gocui.Gui, v *gocui.View) error {
	if pause {
		pause = false
	} else {
		pause = true
	}
	renderStatus(g)
	printMsg(g, "Boom!")
	return nil
}

func scrollView(v *gocui.View, dy int) {
	// Get the size and position of the view.
	_, y := v.Size()
	ox, oy := v.Origin()

	// If we're at the bottom...
	if oy+dy > strings.Count(v.ViewBuffer(), "\n")-y-1 {
		// Set autoscroll to normal again.
		v.Autoscroll = true
	} else {
		// Set autoscroll to false and scroll.
		v.Autoscroll = false
		v.SetOrigin(ox, oy+dy)
	}
}

func clearLogs(g *gocui.Gui, v *gocui.View) error {
	lv, err := g.View("logs")
	if err != nil {
		return err
	}
	lv.Clear()
	return nil
}

func drillDown(g *gocui.Gui, msg string) error {
	maxX, maxY := g.Size()
	// if v, err := g.SetView("msg", maxX/2-30, maxY/2, maxX/2+30, maxY/2+4); err != nil {
	if v, err := g.SetView("msg", 23, 3, maxX-3, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Message details"
		v.Highlight = true
		v.Wrap = false
		v.Autoscroll = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		g.Cursor = true

		fmt.Fprintf(v, "%s\n", msg)

		if _, err := g.SetCurrentView("msg"); err != nil {
			return err
		}
	}
	return nil
}

func printMsg(g *gocui.Gui, msg string) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("msg", maxX/2-30, maxY/2, maxX/2+30, maxY/2+4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "MSG"
		v.Highlight = false
		v.Wrap = false
		v.Autoscroll = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		g.Cursor = false

		fmt.Fprintf(v, "%s\n", msg)

		if _, err := g.SetCurrentView("msg"); err != nil {
			return err
		}
	}
	return nil
}

func closeMsg(g *gocui.Gui, v *gocui.View) error {
	// lv, err := g.View("logs")
	// if err != nil {
	// 	return err
	// }

	if err := g.DeleteView("msg"); err == nil {
		nextView := previousView
		previousView = "console"
		if _, err := g.SetCurrentView(nextView); err != nil {
			return err
		}
	}
	return nil
}
