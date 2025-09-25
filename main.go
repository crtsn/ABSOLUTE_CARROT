package main

// export GATEKEEPER_PGSQL_CONNECTION="postgres://gatekeeper@localhost:5432/gatekeeper?sslmode=disable" # PostgreSQL connection URL https://www.postgresql.org/docs/current/libpq-connect.html#id-1.7.3.8.3.6
// GOOS=js GOARCH=wasm go build -o main.wasm main.go

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/tsoding/gatekeeper/internal"
	"github.com/tsoding/smig"
	"log"
	"os"
	"regexp"
)

var DiscordPingRegexp = regexp.MustCompile("<@[0-9]+>")

func maskDiscordPings(message string) string {
	return DiscordPingRegexp.ReplaceAllString(message, "@[DISCORD PING REDACTED]")
}

var (
	// TODO: make the CommandPrefix configurable from the database, so we can set it per instance
	CommandPrefix = "[\\$\\!]"
	CommandDef    = "([a-zA-Z0-9\\-_]+)( +(.*))?"
	CommandRegexp = regexp.MustCompile("^ *(" + CommandPrefix + ") *" + CommandDef + "$")
)

type Command struct {
	Prefix string
	Name   string
	Args   string
}

func parseCommand(source string) (Command, bool) {
	matches := CommandRegexp.FindStringSubmatch(source)
	if len(matches) == 0 {
		return Command{}, false
	}
	return Command{
		Prefix: matches[1],
		Name:   matches[2],
		Args:   matches[4],
	}, true
}

func main() {
	fmt.Println("Hello, WebAssembly!")
	db := StartPostgreSQL()
	internal.FeedMessageToCarrotson(db, "HELLO")
	internal.FeedMessageToCarrotson(db, "HELP")
	internal.FeedMessageToCarrotson(db, "HELL")
	internal.FeedMessageToCarrotson(db, "HELLO KITTY")
	internal.FeedMessageToCarrotson(db, "HELLO WORLD")
	message, err := internal.CarrotsonGenerate(db, "HEL", 256)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))
}

func migratePostgres(db *sql.DB) bool {
	log.Println("Checking if there are any migrations to apply")
	tx, err := db.Begin()
	if err != nil {
		log.Println("Error starting the migration transaction:", err)
		return false
	}

	err = smig.MigratePG(tx, "./sql/")
	if err != nil {
		log.Println("Error during the migration:", err)

		err = tx.Rollback()
		if err != nil {
			log.Println("Error rolling back the migration transaction:", err)
		}

		return false
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error during committing the transaction:", err)
		return false
	}

	log.Println("All the migrations are applied")
	return true
}

func StartPostgreSQL() *sql.DB {
	pgsqlConnection, found := os.LookupEnv("GATEKEEPER_PGSQL_CONNECTION")
	if !found {
		log.Println("Could not find GATEKEEPER_PGSQL_CONNECTION variable")
		return nil
	}

	db, err := sql.Open("postgres", pgsqlConnection)
	if err != nil {
		log.Println("Could not open PostgreSQL connection:", err)
		return nil
	}

	ok := migratePostgres(db)
	if !ok {
		err := db.Close()
		if err != nil {
			log.Println("Error while closing PostgreSQL connection due to failed migration:", err)
		}
		return nil
	}

	return db
}
