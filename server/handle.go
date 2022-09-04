package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
)

func (s *Server) HandleGetTopHolds(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Headers, Origin,Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Request-Headers")

	type req struct {
		Id   string
		Wave int
	}

	var sr req
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if sr.Id == "" || sr.Wave == 0 {
		http.Error(w, "invalid id or wave", http.StatusBadRequest)
		return
	}

	// check cache
	if _, ok := s.Stats[sr.Wave]; !ok {
		s.Stats[sr.Wave] = make(map[string]CachedStat)
	}
	var stats []dynamicdata.Stats
	cachedStats, ok := s.Stats[sr.Wave][sr.Id]
	if !ok || cachedStats.exp.Before(time.Now()) {
		stats, err = dynamicdata.GetTopHolds(s.db, sr.Id, sr.Wave)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cachedStats = CachedStat{
			stats: stats,
			exp:   time.Now().Add(time.Hour * 24),
		}
		s.Stats[sr.Wave][sr.Id] = cachedStats
		fmt.Println("saved cache")
		fmt.Println(cachedStats)
	} else {
		fmt.Println("hit cache")
		fmt.Println(cachedStats)
		stats = cachedStats.stats
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

	units, err := s.GetUnits()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(units)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(js)
}
