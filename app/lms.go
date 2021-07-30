package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/raoulh/binky-server/config"
)

type lmsReq struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`

	//Only result
	Result json.RawMessage `json:"result,omitempty"`
}

type lmsPlaylistResult struct {
	PlaylistsLoop []struct {
		ID   int    `json:"id"`
		Name string `json:"playlist"`
	} `json:"playlists_loop"`
	Count int `json:"count"`
}

func LoadPlaylist(mac string, playlistId int) error {
	logging.Debugln("Loading playlist", playlistId)

	_, err := SendAction(mac,
		"playlistcontrol",
		"play_index:0",
		"cmd:load",
		"menu:1",
		"playlist_id:"+strconv.Itoa(playlistId),
		"useContextMenu:1",
	)
	if err != nil {
		return err
	}

	_, err = SendAction(mac, "play")
	return err
}

func SendAction(mac string, actions ...string) (res *lmsReq, err error) {
	logging.Debugln("Send action to player", mac, "Actions:", actions)

	lmsUrl := fmt.Sprintf("http://%s:%s/jsonrpc.js", config.Config.String("lms.address"), config.Config.String("lms.port"))

	req := lmsReq{Method: "slim.request"}
	req.Params = append(req.Params, mac)
	req.Params = append(req.Params, actions)

	data, _ := json.Marshal(req)
	bodyData := bytes.NewBuffer(data)

	resp, err := http.Post(lmsUrl, "Content-Type: application/json", bodyData)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	res = &lmsReq{}
	err = json.Unmarshal(body, res)

	return
}

type lmsPlaylist struct {
	Name       string
	PlaylistId int
}

func ListPlaylists(hideSpotify bool) (pls []*lmsPlaylist, err error) {
	res, err := SendAction("-", "playlists", "0", "999")
	plres := lmsPlaylistResult{}
	if err := json.Unmarshal(res.Result, &plres); err != nil {
		return nil, err
	}

	for _, p := range plres.PlaylistsLoop {
		if strings.HasPrefix(p.Name, "Spotify :") && hideSpotify {
			continue
		}

		playlist := &lmsPlaylist{
			Name:       p.Name,
			PlaylistId: p.ID,
		}
		pls = append(pls, playlist)
	}

	return
}
