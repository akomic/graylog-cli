package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"crypto/md5"
	"encoding/hex"
	gl "graylog-cli/graylog"

	"github.com/jroimartin/gocui"
)

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil && v.Name() != "console" {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
			scrollView(v, 1)
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
			}
			scrollView(v, -1)
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
			}
			scrollView(v, 20)
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
			}
			scrollView(v, -pageSize)
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

func quit(g *gocui.Gui, v *gocui.View) error {
	close(done)
	return gocui.ErrQuit
}

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func switchStream(g *gocui.Gui, v *gocui.View) error {
	var line string

	lv, err := g.View("logs")
	if err != nil {
		return err
	}

	_, cy := v.Cursor()
	if line, err = v.Line(cy); err != nil {
		line = ""
	}
	// lv.Clear()
	fmt.Fprintf(lv, "Selecting Stream %s with id %s\n", line, streamIDs[line])
	stream = line

	renderStatus(g)
	return nil
}

func submitFieldFilter(g *gocui.Gui, v *gocui.View) error {
	var line string

	lv, err := g.View("logs")
	if err != nil {
		return err
	}

	_, cy := v.Cursor()
	if line, err = v.Line(cy); err != nil {
		line = ""
	}
	fmt.Fprintf(lv, "Selecting Field %s\n", line)
	return nil
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func submitSearch(g *gocui.Gui, v *gocui.View) error {
	var line string

	lv, err := g.View("logs")
	if err != nil {
		return err
	}
	lv.Clear()

	_, cy := v.Cursor()
	if line, err = v.Line(cy); err != nil {
		line = ""
	}

	// lv.Clear()
	if stream == "" {
		fmt.Fprintf(lv, "First select stream")
	} else {
		// fmt.Fprintf(lv, "Searching for %s in stream %s...\n", line, stream)
		query = line
		renderStatus(g)

		glc := gl.NewBasicAuthClient(GLCFG.BaseURL, GLCFG.Username, GLCFG.Password)
		msgs, err := glc.SearchLogs(query, streamIDs[stream])
		if err != nil {
			return err
		}
		for _, s := range msgs.Data {
			msg := s["message"].(map[string]interface{})
			lineToDisplay := fmt.Sprintf("%s %s %s", msg["timestamp"], msg["source"], msg["message"])
			fmt.Fprintf(lv, "%s\n", lineToDisplay)
			// fmt.Fprintf(lv, "%s\n", reflect.TypeOf(messageIDs))
			recordMessage(lineToDisplay, msg)
		}
		renderFields(g)
	}

	return nil
}

func processLogLine(g *gocui.Gui, v *gocui.View) error {
	var line string
	var err error

	// lv, err := g.View("logs")
	// if err != nil {
	// 	return err
	// }

	_, cy := v.Cursor()
	if line, err = v.Line(cy); err != nil {
		line = ""
	}
	lineID := getMD5Hash(strings.TrimSpace(line))
	msg := lineInMessages(lineID)
	// fmt.Fprintf(lv, "Processing line %s\n", lineInMessages(lineID))

	// msgDetails := fmt.Sprintf("%s\n", time.Now().UTC().Format(time.RFC3339))
	now := time.Now()
	then := now.Add(-12 * time.Hour)

	msgDetails := fmt.Sprintf("%s\n", then.UTC().Format(time.RFC3339))
	for k, v := range msg {
		msgDetails = fmt.Sprintf("%s %s:\"%v\"\n", msgDetails, k, v)
	}
	previousView = "logs"
	drillDown(g, fmt.Sprintf("%s", msgDetails))

	return nil
}

func lineInMessages(l string) map[string]interface{} {
	for _, mid := range messageIDs {
		if mid == l {
			if val, ok := messages[mid]; ok {
				fmt.Println(reflect.TypeOf(val))
				return val
			}
		}
	}
	return nil
}

func applyFilter(g *gocui.Gui, v *gocui.View) error {
	var line string

	cv, err := g.View("console")
	if err != nil {
		return err
	}

	_, cy := v.Cursor()
	if line, err = v.Line(cy); err != nil {
		line = ""
	}
	if query != "" {
		query = fmt.Sprintf("%s AND %s", query, strings.TrimSpace(line))
	} else {
		query = line
	}

	cv.Clear()
	fmt.Fprintf(cv, "%s", query)

	closeMsg(g, v)

	submitSearch(g, cv)

	return nil
}

func fieldExists(f string) bool {
	for _, field := range fields {
		if field == f {
			return true
		}
	}
	return false
}

func recordMessage(identString string, m map[string]interface{}) {
	ident := getMD5Hash(identString)
	messageIDs = append(messageIDs, ident)
	if len(messageIDs) > 999 {
		copy(messageIDs, messageIDs[1:])
		messageIDs = messageIDs[:len(messageIDs)-1]
	}
	messages[ident] = m

	for k := range m {
		if !fieldExists(k) {
			fields = append(fields, k)
		}
	}
}

func renderStatus(g *gocui.Gui) error {
	v, err := g.View("status")
	if err != nil {
		return err
	}

	v.Clear()
	fmt.Fprintf(v, "[stream: %s] ", stream)
	fmt.Fprintf(v, "[tail: %t] ", !pause)
	// fmt.Fprintf(v, "[query: %s] ", query)
	return nil
}

func renderFields(g *gocui.Gui) error {
	v, err := g.View("fields")
	if err != nil {
		return err
	}

	v.Clear()
	for _, f := range fields {
		fmt.Fprintf(v, "%s\n", f)
	}
	return nil
}
