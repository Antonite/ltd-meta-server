package server

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antonite/ltd-meta-server/db"
	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/guide"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/unit"
	"github.com/antonite/ltd-meta-server/util"
)

const maxGuides = 102

type Server struct {
	db       *sql.DB
	Api      *ltdapi.LtdApi
	Version  string
	AllUnits CachedUnits
	UnitMap  map[string]*unit.Unit
	Stats    map[string]map[int]map[string]map[string]CachedStat
	Tables   map[string]bool
	Versions []string
	Guides   []guide.Guide
}

type CachedUnits struct {
	Units []*unit.Unit
	Mercs map[string]*mercenary.Mercenary
}

type CachedStat struct {
	stats []*dynamicdata.Stats
	exp   time.Time
}

func New() (*Server, error) {
	database, err := db.New()
	if err != nil {
		return nil, err
	}

	api := ltdapi.New()

	v, err := api.GetLatestVersion()
	if err != nil {
		return nil, err
	}

	tables, err := db.GetTables(database)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]map[int]map[string]map[string]CachedStat)

	s := &Server{db: database, Api: api, Version: v, Stats: stats, Tables: tables}

	units, err := s.GetUnits()
	if err != nil {
		return nil, err
	}
	ulist := []*unit.Unit{}
	for _, u := range units {
		ulist = append(ulist, u)
	}

	mercs, err := s.GetMercs()
	if err != nil {
		return nil, err
	}

	s.AllUnits = CachedUnits{Units: ulist, Mercs: mercs}
	s.UnitMap = units

	return s, nil
}

func (s *Server) GenerateGuides() {
	fmt.Println("starting guide generation")
	versions, err := s.GetVersions()
	if err != nil {
		fmt.Println(err)
		return
	}
	s.Versions = versions

	guides := []guide.Guide{}
	statMap := make(map[int]map[int][]*dynamicdata.Stats)
	specials := []string{}
	upgrades, err := s.GetUpgrades()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, u := range s.AllUnits.Units {
		viable := true
		htnms := []string{}
		stnms := []string{}
		// make sure we have the data
		for i := 1; i <= guide.Waves; i++ {
			tn := util.GenerateUnitTableName(u.UnitID, i)
			htn := tn + "_holds"
			if _, ok := s.Tables[htn]; !ok {
				viable = false
				break
			}
			htnms = append(htnms, htn)
			stnms = append(stnms, tn+"_sends")
		}
		if !viable {
			continue
		}

		// add cat and sakura to specials for stacks
		if u.UnitID == "sakura_unit_id" || u.UnitID == "nekomata_unit_id" {
			specials = append(specials, strconv.Itoa(u.ID))
		}

		// find the stats for each wave
		sMap := make(map[int][]*dynamicdata.Stats)
		for i := 1; i <= guide.Waves; i++ {
			stats, err := dynamicdata.GetTopHolds(s.db, u.UnitID, "Any", s.AllUnits.Mercs, i, s.Versions[0], 500, false, 3)
			if err != nil {
				fmt.Printf("failed to generate stats for wave %d unit %s: %v\n", i, u.UnitID, err)
				break
			}
			sMap[i] = stats
		}
		if err != nil {
			continue
		}
		statMap[u.ID] = sMap
	}

	for uid := range statMap {
		guides = append(guides, guide.GenerateGuides(uid, statMap, upgrades, specials)...)
	}

	sort.Slice(guides, func(i, j int) bool {
		return guides[i].Score < guides[j].Score
	})
	max := maxGuides
	if len(guides) < max {
		max = len(guides)
	}

	idMap := make(map[string]*unit.Unit)
	for _, u := range s.AllUnits.Units {
		idMap[strconv.Itoa(u.ID)] = u
	}

	counter := 0
	dupes := make(map[string]bool)
	for _, g := range guides {
		if counter >= max {
			break
		}

		primary, _, _ := s.getExpensiveUnits(g.Waves[0].PositionHash, idMap)
		primaryw3, secondary, hasCheapUnit := s.getExpensiveUnits(g.Waves[2].PositionHash, idMap)
		if g.Waves[2].Value > 295 && !hasCheapUnit {
			continue
		}
		g.MainUnitID = primary
		g.SecondaryUnitID = secondary
		if primaryw3 != primary {
			g.SecondaryUnitID = primaryw3
		}
		if secondary == 0 {
			for i := 3; i < 7; i++ {
				p, s, _ := s.getExpensiveUnits(g.Waves[i].PositionHash, idMap)
				if s == 0 {
					continue
				}
				if p != primary {
					g.SecondaryUnitID = p
				} else {
					g.SecondaryUnitID = s
				}
				break
			}
		}
		if g.SecondaryUnitID == 0 {
			continue
		}

		// dedupe
		id := fmt.Sprintf("%d_%d", g.MainUnitID, g.SecondaryUnitID)
		if dupes[id] {
			continue
		}

		// figure out mastermind option
		g.Mastermind = "Greed"
		if len(g.Waves[0].Sends) > 0 && g.Waves[0].Sends[0].Held == 0 && len(g.Waves[1].Sends) > 0 && g.Waves[1].Sends[0].Held == 0 {
			g.Mastermind = "Fiesta"
		} else if g.Waves[0].Value > 250 {
			g.Mastermind = "Cashout"
		} else if g.Waves[2].Value >= 285 && !hasCheapUnit {
			g.Mastermind = "Cashout/Yolo"
		} else if g.Waves[2].Value >= 290 && !hasCheapUnit {
			g.Mastermind = "Cashout/Cartel"
		}

		dupes[id] = true
		counter++
		s.Guides = append(s.Guides, g)
	}

	fmt.Println("finished guide generation")
	return
}

func (s *Server) RefreshTables() error {
	tables, err := db.GetTables(s.db)
	if err != nil {
		return err
	}

	s.Tables = tables
	return nil
}

func (s *Server) GetUnits() (map[string]*unit.Unit, error) {
	return unit.GetAll(s.db)
}

func (s *Server) SaveUnit(u *unit.Unit) error {
	return u.Save(s.db)
}

func (s *Server) GetUpgrades() (map[string][]string, error) {
	return unit.GetUpgrades(s.db)
}

func (s *Server) SaveUpgrade(up *unit.UnitUpgrade) error {
	return up.Save(s.db)
}

func (s *Server) GetMercs() (map[string]*mercenary.Mercenary, error) {
	return mercenary.GetAll(s.db)
}

func (s *Server) SaveMerc(m *mercenary.Mercenary) error {
	return m.Save(s.db)
}

func (s *Server) CreateTable(name string) error {
	return db.CreateTable(s.db, name)
}

func (s *Server) FindHold(tb, hash string, version string) (*dynamicdata.Hold, error) {
	return dynamicdata.FindHold(s.db, tb, hash, version)
}

func (s *Server) SaveHold(tb string, h *dynamicdata.Hold) (int, error) {
	return h.SaveHold(s.db, tb)
}

func (s *Server) UpdateHold(tb string, h *dynamicdata.Hold) error {
	return h.UpdateHold(s.db, tb)
}

func (s *Server) FindSend(tb string, id int, sends string) (*dynamicdata.Send, error) {
	return dynamicdata.FindSend(s.db, tb, id, sends)
}

func (s *Server) InsertSend(tb string, send *dynamicdata.Send) (int, error) {
	return send.InsertSend(s.db, tb)
}

func (s *Server) UpdateSend(tb string, send *dynamicdata.Send) error {
	return send.UpdateSend(s.db, tb)
}

func (s *Server) GetVersions() ([]string, error) {
	return dynamicdata.GetVersions(s.db)
}

func (s *Server) getExpensiveUnits(hash string, uMap map[string]*unit.Unit) (int, int, bool) {
	dupes := make(map[string]bool)
	units := []*unit.Unit{}
	for _, pos := range strings.Split(hash, ",") {
		u := strings.Split(pos, ":")[0]
		dupes[u] = true
	}
	for k := range dupes {
		units = append(units, uMap[k])
	}
	sort.Slice(units, func(i, j int) bool {
		return units[i].TotalValue > units[j].TotalValue
	})
	if len(units) < 2 {
		return units[0].ID, 0, false
	}
	cheap := units[len(units)-1].TotalValue <= 25
	return units[0].ID, units[1].ID, cheap
}
