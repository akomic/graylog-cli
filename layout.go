package main

import (
	"fmt"
	gl "graylog-cli/graylog"

	"github.com/jroimartin/gocui"
	log "github.com/sirupsen/logrus"
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("streams", 0, 0, 20, maxY/3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Streams"
		v.Highlight = true
		v.Wrap = true
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		// fmt.Fprintln(v, "Idea1")
		// fmt.Fprintln(v, "Idea2")
		// fmt.Fprintln(v, "Idea2")
		log.Info("before")
		glc := gl.NewBasicAuthClient(GLCFG.BaseURL, GLCFG.Username, GLCFG.Password)
		streams, err := glc.ListStreams()
		log.Info("after")
		if err != nil {
			return err
		}
		log.Info("failed on error")
		for i, s := range streams.Data {
			if i == 0 {
				stream = fmt.Sprintf("%s", s["title"])
			}

			streamIDs[fmt.Sprintf("%s", s["title"])] = fmt.Sprintf("%s", s["id"])
			fmt.Fprintf(v, "%s\n", s["title"])
		}

		if _, err = setCurrentViewOnTop(g, "streams"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("fields", 0, maxY/3+1, 20, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Fields"
		v.Highlight = true
		v.Wrap = true
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		if _, err = setCurrentViewOnTop(g, "fields"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("logs", 20, 0, maxX-1, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Logs"
		v.Highlight = true
		v.Wrap = true
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		// v.Editable = true
		if _, err = setCurrentViewOnTop(g, "logs"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("status", -1, maxY-3, maxX, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		// v.Title = "Console"
		v.Highlight = false
		v.Wrap = false
		v.Autoscroll = false
		// v.SelBgColor = gocui.ColorWhite
		// v.SelFgColor = gocui.ColorBlack0
		v.BgColor = gocui.ColorWhite
		v.FgColor = gocui.ColorBlack
		v.Frame = false
		// v.Editable = true
		renderStatus(g)
		if _, err = setCurrentViewOnTop(g, "status"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("prompt", -1, maxY-2, 7, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		// v.Title = "Console"
		v.Highlight = false
		v.Wrap = false
		v.Autoscroll = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Frame = false
		v.Editable = false
		fmt.Fprintf(v, "search> ")
		if _, err = setCurrentViewOnTop(g, "prompt"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("console", 7, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		// v.Title = "Console"
		v.Highlight = true
		v.Wrap = true
		v.Autoscroll = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		v.Frame = false
		v.Editable = true
		if _, err = setCurrentViewOnTop(g, "console"); err != nil {
			return err
		}
	}

	return nil
}
