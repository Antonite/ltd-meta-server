package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/server"
	"github.com/antonite/ltd-meta-server/unit"
)

const vers = "9.07.2"

func main() {
	srv, err := server.New()
	if err != nil {
		panic("failed to create server")
	}

	fmt.Println("starting unit generation")
	if err := generateUnits(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished unit generation")

	fmt.Println("starting table generation")
	if err := generateTables(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished table generation")
}

func generateTables(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	waves := 10
	for k := range savedUnits {
		n := strings.TrimSuffix(k, "unit_id")
		for i := 1; i <= waves; i++ {
			tableName := fmt.Sprintf("%swave_%v", n, i)
			if err := srv.CreateTable(tableName); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateUnits(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}
	savedMercs, err := srv.GetMercs()
	if err != nil {
		return err
	}

	// special hardcoded units
	if _, ok := savedUnits["hell_raiser_buffed_unit_id"]; !ok {
		newUnit := unit.Unit{
			ID:         "hell_raiser_buffed_unit_id",
			Name:       "Hell Raiser Buffed",
			IconPath:   "Icons/HellRaiser.png",
			TotalValue: 215,
			Version:    "9.07.2",
		}
		if err := srv.SaveUnit(newUnit); err != nil {
			return err
		}
	}
	if _, ok := savedUnits["pack_rat_nest_unit_id"]; !ok {
		newUnit := unit.Unit{
			ID:         "pack_rat_nest_unit_id",
			Name:       "Pack Rat Hunting",
			IconPath:   "Icons/PackRat(Footprints).png",
			TotalValue: 0,
			Version:    "9.07.2",
		}
		if err := srv.SaveUnit(newUnit); err != nil {
			return err
		}
	}

	off := 0
	for {
		fmt.Printf("offset: %v\n", off)
		resp, err := srv.Api.RequestUnits(off, vers)
		if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 404) {
			fmt.Println(resp)
			return errors.New("failed to retrieve units")
		} else if resp.StatusCode == 404 {
			return nil
		}

		units := []ltdapi.Unit{}
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		if err = decoder.Decode(&units); err != nil {
			return err
		}

		for _, u := range units {
			if u.CategoryClass != "Standard" {
				continue
			}
			switch u.UnitClass {
			case "Mercenary":
				if _, ok := savedMercs[u.UnitId]; !ok {
					newMerc := mercenary.Mercenary{
						ID:          u.UnitId,
						Name:        u.Name,
						IconPath:    u.IconPath,
						MythiumCost: u.MythiumCost,
						Version:     u.Version,
					}
					if err := srv.SaveMerc(newMerc); err != nil {
						return err
					}
				}
			case "Fighter":
				if _, ok := savedUnits[u.UnitId]; !ok {
					if u.TotalValue == "" {
						return errors.New(fmt.Sprintf("got a unit with empty total value: %s", u.UnitId))
					}
					val, err := strconv.Atoi(u.TotalValue)
					if err != nil {
						return errors.New(fmt.Sprintf("failed to convert unit total value: %s", u.UnitId))
					}
					newUnit := unit.Unit{
						ID:         u.UnitId,
						Name:       u.Name,
						IconPath:   u.IconPath,
						TotalValue: val,
						Version:    u.Version,
					}
					if err := srv.SaveUnit(newUnit); err != nil {
						return err
					}
				}
			}
		}

		off += 50
	}
}
