package ltdapi

import (
	"fmt"
	"net/http"
	"os"
)

const url = "https://apiv2.legiontd2.com/games?limit=50&sortBy=date&sortDirection=1&includeDetails=true&dateAfter=%v&offset=%v"
const unitsUrl = "https://apiv2.legiontd2.com/units/byVersion/%s?limit=50&enabled=true&offset=%v"

type LtdApi struct {
	Key string
}

type LTDResponse struct {
	Games []Game
}

type Game struct {
	PlayersData []PlayersData
	Date        string
}

type PlayersData struct {
	Cross                      bool
	MercenariesReceivedPerWave [][]string
	LeaksPerWave               [][]string
	BuildPerWave               [][]string
}

type Unit struct {
	UnitId        string
	Name          string
	IconPath      string
	Version       string
	TotalValue    string
	MythiumCost   int
	UnitClass     string
	CategoryClass string
}

func New() *LtdApi {
	key := os.Getenv("apikey")
	return &LtdApi{
		Key: key,
	}
}

func (api *LtdApi) RequestGames(offset int, date string) (*http.Response, error) {
	pUrl := fmt.Sprintf(url, date, offset)
	req, err := http.NewRequest("GET", pUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", api.Key)

	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}

func (api *LtdApi) RequestUnits(offset int, version string) (*http.Response, error) {
	pUrl := fmt.Sprintf(unitsUrl, version, offset)
	req, err := http.NewRequest("GET", pUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", api.Key)

	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}
