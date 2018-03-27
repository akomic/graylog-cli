package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	gl "graylog-cli/graylog"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	viewArr = []string{"console", "streams", "logs"}
	active  = 0

	// Flags
	fCfgFile, fGraylogHostname, fUserName, fPassword string

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

// GLCliConfig struct represent minimum configuration required for
// program to function
type GLCliConfig struct {
	Username         string
	Password         string
	BaseURL          string
	AllowInsecureTLS bool
}

// NewGLCliConfig returns new GLCliConfig struct
func NewGLCliConfig(username, password, baseurl string, allowinsecuretls bool) *GLCliConfig {
	return &GLCliConfig{
		Username:         username,
		Password:         password,
		BaseURL:          baseurl,
		AllowInsecureTLS: allowinsecuretls,
	}
}

// GLCFG Global configuration
var GLCFG *GLCliConfig

var mainCmd = &cobra.Command{
	Use:   "graylog-cli",
	Short: "Shows streaming logs from remote graylog server",
	Run:   runTail,
}

func main() {
	mainCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	mainCmd.PersistentFlags().StringVarP(&fCfgFile, "config", "c", "", "config file (default is $HOME/.config/graylog-cli.yml)")
	mainCmd.PersistentFlags().StringVarP(&fGraylogHostname, "graylog-host-url", "g", "admin", "Graylog Server URL")
	mainCmd.PersistentFlags().StringVarP(&fUserName, "username", "u", "admin", "Username user for graylog access")
	mainCmd.PersistentFlags().StringVarP(&fPassword, "password", "p", "password", "Password for graylog access")
	viper.SetDefault("username", "admin")
	viper.SetDefault("password", "password")
	viper.SetDefault("graylog-host-url", "http://127.0.0.1/api")
}

func initConfig() {
	// Don't forget to read config either from fCfgFile or from home directory!
	viper.SetConfigType("yaml")
	if fCfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(fCfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in ~/.config directory
		viper.AddConfigPath(home + "/.config/")
		viper.SetConfigName("graylog-cli")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
	GLCFG = NewGLCliConfig(viper.GetString("username"), viper.GetString("password"), viper.GetString("baseurl"), viper.GetBool("allowinsecuretls"))
	//log.Infof("%v\n", GLCFG)
}

func runTail(cmd *cobra.Command, args []string) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalln(err)
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
		log.Warnf("%v\n", err)
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

func getMD5Hash(text string) string {
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
