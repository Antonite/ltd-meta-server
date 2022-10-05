package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/util"
)

const cacheTimeout = 24

func (s *Server) HandleGetTopHolds(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	type req struct {
		Id   string
		Wave string
	}

	var sr req
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wave, err := strconv.Atoi(sr.Wave)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tn := util.GenerateUnitTableName(sr.Id, wave) + "_holds"
	if _, ok := s.Tables[tn]; !ok {
		http.Error(w, "not even possible", http.StatusNotFound)
		return
	}

	// check cache
	if _, ok := s.Stats[wave]; !ok {
		s.Stats[wave] = make(map[string]CachedStat)
	}
	var stats []*dynamicdata.Stats
	cachedStats, ok := s.Stats[wave][sr.Id]
	if !ok || cachedStats.exp.Before(time.Now()) {
		stats, err = dynamicdata.GetTopHolds(s.db, sr.Id, s.AllUnits.Mercs, wave)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cachedStats = CachedStat{
			stats: stats,
			exp:   time.Now().Add(time.Hour * cacheTimeout),
		}
		s.Stats[wave][sr.Id] = cachedStats
	} else {
		stats = cachedStats.stats
	}

	if len(stats) == 0 {
		http.Error(w, "no good builds found", http.StatusNotFound)
		return
	}

	js, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(js)
}

func (s *Server) HandleGetUnits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	js, err := json.Marshal(s.AllUnits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(js)
}
