package server

import (
	"database/sql"
	"time"

	"github.com/antonite/ltd-meta-server/db"
	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/unit"
)

type Server struct {
	db       *sql.DB
	Api      *ltdapi.LtdApi
	Version  string
	AllUnits CachedUnits
	Stats    map[int]map[string]CachedStat
	Tables   map[string]bool
	Versions []string
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

	stats := make(map[int]map[string]CachedStat)

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

	return s, nil
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
