package ltdapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const url = "https://apiv2.legiontd2.com/games?limit=50&sortBy=date&sortDirection=1&includeDetails=true&dateAfter=%v&dateBefore=%v&offset=%v"
const unitsUrl = "https://apiv2.legiontd2.com/units/byVersion/%s?limit=0"

type LtdApi struct {
	Key string
}

type LTDResponse struct {
	Games []Game
}

type Game struct {
	PlayersData []PlayersData
	Date        string
	QueueType   string
	EndingWave  int
	Version     string
}

type PlayersData struct {
	MercenariesReceivedPerWave [][]string
	LeaksPerWave               [][]string
	BuildPerWave               [][]string
	ValuePerWave               []int
	WorkersPerWave             []int
	GameResult                 string
	OverallElo                 int
	PlayerName                 string
}

type Unit struct {
	UnitId        string
	Name          string
	IconPath      string
	Version       string
	TotalValue    string
	MythiumCost   string
	IncomeBonus   string
	UnitClass     string
	CategoryClass string
	UpgradesFrom  []string
}

func New() *LtdApi {
	key := os.Getenv("apikey")
	return &LtdApi{
		Key: key,
	}
}

func (api *LtdApi) GetLatestVersion() (string, error) {
	req, err := http.NewRequest("GET", "https://apiv2.legiontd2.com/units/byId/elite_archer_unit_id", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", api.Key)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		fmt.Println(resp)
		return "", errors.New("failed to retrieve version")
	}

	type u struct {
		Version string
	}
	au := u{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&au); err != nil {
		return "", err
	}
	if au.Version == "" {
		return "", errors.New("unit contained an empty version")
	}
	return au.Version, nil
}

func (api *LtdApi) RequestUnits(version string, output chan<- Unit, errChan chan<- error) {
	defer close(output)
	defer close(errChan)

	// get units from api
	resp, err := api.getUnits(version)
	if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 404) {
		fmt.Println(resp)
		errChan <- errors.New("failed to retrieve units")
		return
	} else if resp.StatusCode == 404 {
		return
	}

	// append to return channel
	units := []Unit{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&units); err != nil {
		errChan <- err
		return
	}
	for _, u := range units {
		output <- u
	}
}

func (api *LtdApi) RequestGames(startDate string, endDate string, output chan<- Game, errChan chan<- error, wg *sync.WaitGroup, worker int, numWorkers int) {
	defer wg.Done()
	offset := worker * 50
	for {
		// get units from api
		games, err := api.getGames(offset, startDate, endDate, worker)
		if err != nil {
			errChan <- errors.New("failed to retrieve games")
			return
		}

		if len(games) == 0 {
			return
		}

		for _, g := range games {
			output <- g
		}

		offset += numWorkers * 50
	}
}

func (api *LtdApi) getUnits(version string) (*http.Response, error) {
	pUrl := fmt.Sprintf(unitsUrl, version)
	req, err := http.NewRequest("GET", pUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", api.Key)

	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}

func (api *LtdApi) getGames(offset int, startDate string, endDate string, w int) ([]Game, error) {
	pUrl := fmt.Sprintf(url, startDate, endDate, offset)
	var resp *http.Response
	var req *http.Request
	var err error
	for i := 0; i < 50; i++ {
		time.Sleep(time.Second * 5)
		req, err = http.NewRequest("GET", pUrl, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", api.Key)
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		client := &http.Client{Timeout: time.Second * 25}

		if i > 0 {
			fmt.Printf("worker %d retrying request #%d, %d\n", w, i, offset)
		}

		resp, err = client.Do(req)
		if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 404) {
			continue
		} else if resp.StatusCode == 404 {
			return []Game{}, nil
		}

		games := []Game{}
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		if err = decoder.Decode(&games); err != nil {
			continue
		}

		return games, nil
	}

	fmt.Printf("worker %d failed all tries %d\n", w, offset)
	return nil, err
}
