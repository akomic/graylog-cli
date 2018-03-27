package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gl "graylog-cli/graylog"
)

var (
	log     = logrus.New()
	viewArr = []string{"console", "streams", "logs"}
	active  = 0

	// Flags
	fCfgFile, fGraylogHostname, fUserName, fPassword string

	done = make(chan struct{})
	wg   sync.WaitGroup

	mu sync.Mutex // protects queryFinished

	tail          = false
	stream        = ""
	streamIDs     = map[string]string{}
	query         = ""
	queryFinished = true
	lastTimestamp = ""

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
	Run:   run,
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

	file, err := os.OpenFile("/tmp/graylog-cli.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	//log.Infof("%v\n", GLCFG)
}

func run(cmd *cobra.Command, args []string) {
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

func doLogs(g *gocui.Gui) {
	defer wg.Done()

	for {
		select {
		case <-done:
			return
		case <-time.After(100 * time.Millisecond):
			if !queryFinished {
				mu.Lock()
				queryFinished = true
				mu.Unlock()

				g.Update(func(g *gocui.Gui) error {
					lv, err := g.View("logs")
					if err != nil {
						return err
					}
					// lv.Clear()
					// fmt.Fprintf(lv, "Results for %s stream %s\n", query, stream)

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
						lastTimestamp = fmt.Sprintf("%s", msg["timestamp"])
					}
					renderFields(g)
					return nil
				})
			}
		}
	}
}
