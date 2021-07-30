package app

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/grandcat/zeroconf"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/raoulh/binky-server/config"
	logger "github.com/raoulh/binky-server/log"
	"github.com/raoulh/binky-server/model"
)

var (
	logging *logrus.Entry

	eMain          *echo.Echo
	zeroconfServer *zeroconf.Server
)

func init() {
	logging = logger.NewLogger("app")
}

type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

//Init main streamr app
func Init() error {
	eMain = echo.New()
	eMain.HideBanner = true
	eMain.HidePort = true
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*")),
	}
	eMain.Renderer = renderer
	eMain.Static("/static", "static")

	//Middlewares
	eMain.Use(logger.Middleware())

	//CORS
	// eMain.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	// AllowOrigins: []string{"*"},
	// AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	// }))

	//Website
	eMain.GET("/", indexHandler)
	eMain.GET("/ws", websocketHandler)

	return model.Init()
}

//Run all background jobs. This func does not block and return
//an error if something failed to start
func Run() {
	addr := config.Config.String("general.http.address") + ":" + strconv.Itoa(config.Config.Int("general.http.port"))

	go eMain.Start(addr)

	logging.Infoln("\u21D2 Main HTTP Listening on", addr)

	var err error
	// Advertise streamr with mDNS
	zeroconfServer, err = zeroconf.Register("Binky", "_binky._tcp", "local.", config.Config.Int("general.http.port"), []string{""}, nil)
	if err != nil {
		log.Printf("failed annouce mdns service:  %v", err)
	}
}

//Shutdown all background jobs
func Shutdown(ctx context.Context) error {
	var e error

	err := eMain.Shutdown(ctx)
	if err != nil {
		e = fmt.Errorf("error closing http server: %v", err)
	}
	logging.Println("main http server stopped")

	zeroconfServer.Shutdown()

	return e
}

func indexHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", nil)
}

type Message struct {
	Msg     string                 `json:"msg"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: websocketOriginCheck,
	}
)

func websocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for {
		_, msg, err := ws.ReadMessage()

		if err != nil {
			c.Logger().Error(err)
			return err
		}

		var message Message
		err = json.Unmarshal([]byte(msg), &message)
		if err != nil {
			logging.Println(err)
		}

		if message.Msg == "nfc" {
			if message.Payload["msg"] == "present" {
				logging.Debugf("got nfc message %s [%s] from %s", message.Payload["msg"], message.Payload["nfc_id"], message.Payload["mac_address"])

				p, err := model.GetPlaylistAssoc(message.Payload["nfc_id"].(string))
				if err != nil {
					log.Println("NFC tag not found")
					continue
				}

				LoadPlaylist(message.Payload["mac_address"].(string), p.PlaylistId)

			} else if message.Payload["msg"] == "removed" {
				logging.Debugf("got nfc message %s from %s", message.Payload["msg"], message.Payload["mac_address"])

				SendAction(message.Payload["mac_address"].(string), "stop")
			}
		}
	}
}

func websocketOriginCheck(r *http.Request) bool {
	//TODO: better check for allowed origin here
	return true
}
