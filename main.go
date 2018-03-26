package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/jroimartin/gocui"
	gl "graylog-cli/graylog"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	viewArr = []string{"console", "streams", "logs"}
	active  = 0

	done = make(chan struct{})
	wg   sync.WaitGroup

	mu  sync.Mutex // protects ctr
	ctr = 0

	pause     = true
	stream    = ""
	streamIDs = map[string]string{}
	query     = ""

	messageIDs = make([]string, 1000)
	messages   = map[string]map[string]interface{}{}

	fields = []string{}

	previousView = "console"
)

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	wg.Add(1)
	go doLogs(g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

	wg.Wait()
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
	lineID := GetMD5Hash(strings.TrimSpace(line))
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

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
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
	ident := GetMD5Hash(identString)
	messageIDs = append(messageIDs, ident)
	if len(messageIDs) > 999 {
		copy(messageIDs, messageIDs[1:])
		messageIDs = messageIDs[:len(messageIDs)-1]
	}
	messages[ident] = m

	for k, _ := range m {
		if !fieldExists(k) {
			fields = append(fields, k)
		}
	}
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

		glc := gl.NewBasicAuthClient("admin", "pass")
		msgs, err := glc.SearchLogs(query, streamIDs[stream])
		if err != nil {
			return err
		} else {
			for _, s := range msgs.Data {
				msg := s["message"].(map[string]interface{})
				lineToDisplay := fmt.Sprintf("%s %s %s", msg["timestamp"], msg["source"], msg["message"])
				fmt.Fprintf(lv, "%s\n", lineToDisplay)
				// fmt.Fprintf(lv, "%s\n", reflect.TypeOf(messageIDs))
				recordMessage(lineToDisplay, msg)
			}
			renderFields(g)
		}
	}

	return nil
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

func doLogs(g *gocui.Gui) {
	defer wg.Done()

	for {
		select {
		case <-done:
			return
		case <-time.After(100 * time.Millisecond):
			if !pause && stream != "" {
				mu.Lock()
				n := ctr
				ctr++
				mu.Unlock()

				g.Update(func(g *gocui.Gui) error {
					v, err := g.View("logs")
					if err != nil {
						return err
					}
					// v.Clear()
					fmt.Fprintf(v, "[%d] Results for %s stream %s\n", n, query, stream)
					// } else {
					// 	fmt.Fprintf(v, "Idle ...\n")
					return nil
				})
			}
		}
	}
}
