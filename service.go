package textkernel

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Output struct {
	message string `json:"message"`
	status  int    `json:"status"`
}

// DB is a wrapper around SQLite driver
type DB struct {
	driver *sql.DB
}

// NewDB creates a new instance of DB type
func NewDB(driver *sql.DB) DB {
	return DB{driver}
}

// CreateTable creates the initial table in the DB
func (db *DB) CreateTable() error {
	sqlStmt := `
	create table geonames (
		name string not null primary key,
		population int not null,
		latitude float64,
		longitude float64);
	`
	_, err := db.driver.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err, sqlStmt)
	}

	// create an index on lat/lon to speed up location/distance queries
	sqlStmt = "create index geonames_idx on geonames (latitude, longitude)"
	_, err = db.driver.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err, sqlStmt)
	}

	return nil
}

// LoadGeoData loads geo data into the database
func (db *DB) LoadGeoData(geoFile string) error {
	dataFile, err := os.Open(geoFile)
	if err != nil {
		return fmt.Errorf(
			"cannot open geonames dump file %v: %v",
			geoFile, err,
		)
	}
	defer dataFile.Close()

	tx, err := db.driver.Begin()
	if err != nil {
		log.Fatal(err)
	}

	reader := csv.NewReader(dataFile)
	reader.Comma = '\t' // geoname dumps are tab-separated

	for {
		// read file line by line
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf(
				"unable to process line of geonames dump file: %v", err,
			)
			continue
		}
		stmt, err := tx.Prepare(
			"insert into geonames(name, population, latitude, longitude) values(?, ?, ?, ?)",
		)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		// FIXME record should be serialized properly, this is gross 🤮
		stmt.Exec(record[1], record[14], record[4], record[5])
	}
	tx.Commit()

	return nil
}

// GetPopulation is an http handler which accepts a place name and a radius, and
// returns estimated population size
func (db *DB) GetPopulation(w http.ResponseWriter, req *http.Request) {
	place := req.URL.Query().Get("place")
	radiusString := req.URL.Query().Get("radius")

	// check if the url contains the right parameters
	if place == "" || radiusString == "" {
		log.Errorf("incorrect query: %v", req.URL.Query().Encode())
		renderResponse(w,
			"incorrect query, you need to submit place and radius as URL parameters\n",
			http.StatusBadRequest,
		)
		return
	}

	radius, err := strconv.ParseFloat(radiusString, 64)
	if err != nil {
		log.Error("radius is not a valid float")
		renderResponse(w, "radius is not a valid float", http.StatusBadRequest)
		return
	}

	// what's the lat/lon of the place we're starting from?
	var lat, lon float64
	err = db.driver.QueryRow(
		"select latitude, longitude from geonames where name=?", place,
	).Scan(&lat, &lon)

	// 1 degree of latitude is about 111111 meters and 1 degree of longitude is
	// 111111*cos(latitude) meters
	// https://stackoverflow.com/a/22024404/609972
	latDist := 1.0 / 111.1 * radius
	lonDist := 1.0 / math.Abs(111.1*math.Cos(lat)) * radius

	stmt, err := db.driver.Prepare(
		`select population from geonames where
		latitude between ? and ?
		and longitude between ? and ?`,
	)
	if err != nil {
		log.Errorf("cannot prepare query: %v", err)
		renderResponse(w, fmt.Sprintf(
			"cannot prepare query: %v\n",
			err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// execute!
	rows, err := stmt.Query(lat-latDist, lat+latDist, lon-lonDist, lon+lonDist)
	defer rows.Close()

	var population int
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			// Check for a scan error.
			// Query rows will be closed with defer.
			log.Fatal(err)
		}
		population += p
	}
	if err := rows.Err(); err != nil {
		log.Errorf("error reading row: %v", err)
	}

	log.Infof(
		"population in radius of %vkm from %v estimate is: %v",
		radius, place, population,
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(population)
}

func renderResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(message)
}
