package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/raoulh/binky-server/app"
	"github.com/raoulh/binky-server/config"
	"github.com/raoulh/binky-server/model"

	"github.com/fatih/color"
	cli "github.com/jawher/mow.cli"

	logger "github.com/raoulh/binky-server/log"
	"github.com/sirupsen/logrus"
)

const (
	CONFIG_FILENAME = "binky.toml"

	CharStar     = "\u2737"
	CharAbort    = "\u2718"
	CharCheck    = "\u2714"
	CharWarning  = "\u26A0"
	CharArrow    = "\u2012\u25b6"
	CharVertLine = "\u2502"
)

var (
	blue       = color.New(color.FgBlue).SprintFunc()
	errorRed   = color.New(color.FgRed).SprintFunc()
	errorBgRed = color.New(color.BgRed, color.FgBlack).SprintFunc()
	green      = color.New(color.FgGreen).SprintFunc()
	cyan       = color.New(color.FgCyan).SprintFunc()
	bgCyan     = color.New(color.FgWhite).SprintFunc()

	logging *logrus.Entry

	conffile *string
)

func exit(err error, exit int) {
	fmt.Fprintln(os.Stderr, errorRed(CharAbort), err)
	cli.Exit(exit)
}

func handleSignals() {
	sigint := make(chan os.Signal, 1)

	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sigint

	logging.Println("Shuting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		exit(err, 1)
	}
}

func main() {
	logging = logger.NewLogger("binky")

	runtime.GOMAXPROCS(runtime.NumCPU())

	cApp := cli.App("binky-server", "Binky, NFC squeezebox radio")
	cApp.Spec = "[-c]"

	conffile = cApp.StringOpt("c config", CONFIG_FILENAME, "Set config file")

	//Main action starts the server
	cApp.Action = func() {
		config.InitConfig(conffile)

		if err := app.Init(); err != nil {
			exit(err, 1)
		}

		app.Run()

		// This will block until a signal is received
		handleSignals()
	}

	cApp.Command("db", "Database functions", func(cmd *cli.Cmd) {
		cmd.Command("list", "list NFC/Playlist associations", cmdNFCPlaylistList)
		cmd.Command("add", "add new NFC/Playlist association", cmdNFCPlaylistAdd)
		cmd.Command("delete", "delete an NFC/Playlist association", cmdNFCPlaylistDel)
		cmd.Command("list-playlist", "List all available playlist and their ID", cmdPlaylistList)
	})

	if err := cApp.Run(os.Args); err != nil {
		exit(err, 1)
	}
}

func strAtLeast(s string, minSize int) string {
	for {
		if len(s) >= minSize {
			break
		}
		s += " "
	}
	return s
}

func cmdNFCPlaylistList(cmd *cli.Cmd) {
	cmd.Action = func() {
		config.InitConfig(conffile)

		if err := model.Init(); err != nil {
			exit(err, 1)
		}

		pls, err := model.GetAllPlaylistAssoc()
		if err != nil {
			logging.Errorf("Failed to list all playlist")
			exit(err, 1)
		}

		plsLms, _ := app.ListPlaylists(false)
		lmsMap := make(map[int]string)
		for _, p := range plsLms {
			lmsMap[p.PlaylistId] = p.Name
		}

		fmt.Println("List of NFC/Playlist:")
		fmt.Println("┌────────────┬─────────────┬───────────────────────────────────────────────────────┐")
		fmt.Println("│   NFC ID   │ Playlist ID │                                                       │")
		fmt.Println("├────────────┼─────────────┼───────────────────────────────────────────────────────┤")
		for _, p := range pls {
			fmt.Printf("│%s│%s│%s│\n", strAtLeast(p.NFCID, 12), strAtLeast(strconv.Itoa(p.PlaylistId), 13), strAtLeast(lmsMap[p.PlaylistId], 55))
		}
		fmt.Println("└────────────┴─────────────┴───────────────────────────────────────────────────────┘")
	}
}

func cmdNFCPlaylistAdd(cmd *cli.Cmd) {
	cmd.Spec = "NFC PLAYLIST_ID"
	var (
		nfc         = cmd.StringArg("NFC", "", "NFC Tag")
		playlist_id = cmd.IntArg("PLAYLIST_ID", 0, "LMS Playlist ID")
	)

	cmd.Action = func() {
		config.InitConfig(conffile)

		if err := model.Init(); err != nil {
			exit(err, 1)
		}

		err := model.AddPlaylistAssoc(*nfc, *playlist_id)
		if err != nil {
			logging.Errorf("Failed to add playlist assoc")
			exit(err, 1)
		}

		fmt.Println("🮥  Done " + green("🮱"))
	}
}

func cmdNFCPlaylistDel(cmd *cli.Cmd) {
	cmd.Spec = "NFC"
	var (
		nfc = cmd.StringArg("NFC", "", "NFC Tag")
	)

	cmd.Action = func() {
		config.InitConfig(conffile)

		if err := model.Init(); err != nil {
			exit(err, 1)
		}

		err := model.DeletePlaylistAssoc(*nfc)
		if err != nil {
			logging.Errorf("Failed to delete playlist assoc")
			exit(err, 1)
		}

		fmt.Println("🮥  Done " + green("🮱"))
	}
}

func cmdPlaylistList(cmd *cli.Cmd) {
	cmd.Spec = "[-s]"
	var (
		spotify = cmd.BoolOpt("s spotify", false, "If arg passed, show also spotify playlists")
	)

	cmd.Action = func() {
		config.InitConfig(conffile)

		fmt.Println("List LMS playlists:")
		pls, _ := app.ListPlaylists(!*spotify)
		for _, p := range pls {
			fmt.Printf("%s [%d]\n", p.Name, p.PlaylistId)
		}
	}
}
