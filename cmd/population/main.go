package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/markbates/pkger"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	"github.com/pkazmierczak/textkernel"
)

var (
	httpListenAddress = flag.String("http-listen-address", ":8000", "The address to listen for http")
	loglvl            = flag.String("log-level", "info", "The log level")

	createDBCmd = flag.NewFlagSet("create-db", flag.ExitOnError)
	dumpDBCmd   = flag.NewFlagSet("dump-db", flag.ExitOnError)

	loadGeoDataCmd = flag.NewFlagSet("load", flag.ExitOnError)
)

func main() {
	flag.Parse()

	// setup logging
	logLevel, err := log.ParseLevel(*loglvl)
	if err != nil {
		logLevel = log.InfoLevel
		log.Warnf("invalid log-level %s, set to %v", *loglvl, log.InfoLevel)
	}
	log.SetLevel(logLevel)

	// open up the sqlite file bundled with the application
	dbFile, err := pkger.Open("/cmd/population/db.sqlite3")
	if err != nil {
		log.Fatalf(
			"unable to open the bundled geo database file. perhaps you need to create one? %v",
			err,
		)
	}
	defer dbFile.Close()

	b, err := ioutil.ReadAll(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	// Create temporary file for database.
	tmpDB, err := ioutil.TempFile("", "db*.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	// Remove this file after on exit.
	defer func() {
		err := os.Remove(tmpDB.Name())
		if err != nil {
			log.Print(err)
		}
	}()

	// Write database to file.
	_, err = tmpDB.Write(b)
	if err != nil {
		log.Print(err)
	}
	err = tmpDB.Close()
	if err != nil {
		log.Print(err)
	}

	// open up the sqlite file
	sqlite, err := sql.Open("sqlite3", tmpDB.Name()+"?mode=ro")
	if err != nil {
		log.Fatalf("cannot open db file: %v", err)
	}
	defer sqlite.Close()

	// create a new DB instance
	db := textkernel.NewDB(sqlite)

	// check if we're asked to put some geonames data into the DB
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create-db":
			createDBCmd.Parse(os.Args[1:])
			err := db.CreateTable()
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		case "dump-db":
			dumpDBCmd.Parse(os.Args[1:])
			err := ioutil.WriteFile("db.sqlite3", b, 0644)
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		case "load":
			loadGeoDataCmd.Parse(os.Args[1:])
			err := db.LoadGeoData(os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	}

	// healthcheck
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hi I am reasonably healthy")
	})

	// API endpoint
	http.HandleFunc("/population", db.GetPopulation)

	log.WithFields(log.Fields{
		"address": *httpListenAddress,
	}).Println("started http server")

	log.Fatal(http.ListenAndServe(*httpListenAddress, nil))
}
