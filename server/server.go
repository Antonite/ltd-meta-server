package server

import (
	"database/sql"

	"github.com/antonite/ltd-meta-server/benchmark"
	"github.com/antonite/ltd-meta-server/db"
	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/unit"
)

type Server struct {
	db      *sql.DB
	Api     *ltdapi.LtdApi
	Version string
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

	return &Server{db: database, Api: api, Version: v}, nil
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

func (s *Server) Getbenchmarks() (map[int]map[string]*benchmark.Benchmark, error) {
	return benchmark.GetAll(s.db)
}

func (s *Server) SaveBenchmark(b *benchmark.Benchmark) error {
	return b.Save(s.db)
}

func (s *Server) CreateTable(name string) error {
	return db.CreateTable(s.db, name)
}

func (s *Server) FindHold(tb, hash string) (*dynamicdata.Hold, error) {
	return dynamicdata.FindHold(s.db, tb, hash)
}

func (s *Server) SaveHold(tb string, h *dynamicdata.Hold) (int, error) {
	return h.SaveHold(s.db, tb)
}

func (s *Server) FindSends(tb string, id int, sends string) (*dynamicdata.Send, error) {
	return dynamicdata.FindSends(s.db, tb, id, sends)
}

func (s *Server) InsertSend(tb string, send *dynamicdata.Send) error {
	return send.InsertSend(s.db, tb)
}

func (s *Server) UpdateSend(tb string, send *dynamicdata.Send) error {
	return send.UpdateSend(s.db, tb)
}
