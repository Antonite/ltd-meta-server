package server

import (
	"database/sql"

	"github.com/antonite/ltd-meta-server/db"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/unit"
)

type Server struct {
	db  *sql.DB
	Api *ltdapi.LtdApi
}

func New() (*Server, error) {
	database, err := db.New()
	if err != nil {
		return nil, err
	}

	api := ltdapi.New()

	return &Server{db: database, Api: api}, nil
}

func (s *Server) GetUnits() (map[string]unit.Unit, error) {
	return unit.GetAll(s.db)
}

func (s *Server) SaveUnit(u unit.Unit) error {
	return u.Save(s.db)
}

func (s *Server) GetMercs() (map[string]mercenary.Mercenary, error) {
	return mercenary.GetAll(s.db)
}

func (s *Server) SaveMerc(m mercenary.Mercenary) error {
	return m.Save(s.db)
}

func (s *Server) CreateTable(name string) error {
	return db.CreateTable(s.db, name)
}
