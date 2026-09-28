package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/crtsn/crtsn/carrotson"
	"log"
	"os"
	"regexp"
	"strings"

	_ "github.com/crtsn/crtsn/sql/libsqlite3"
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
	// assuming running by `go run cmd/filldb/main.go` in _ABSOLUTE_CARROT directory
	// db_path := "../test.sqlite"
	db_path := "../bee.sqlite"
	shouldRemove := true
	shouldInit := false
	if _, err := os.Stat(db_path); err == nil {
		if shouldRemove {
			if err = os.Remove(db_path); err != nil {
				log.Fatal(err)
			}
			shouldInit = true
		}
	} else if errors.Is(err, os.ErrNotExist) {
		shouldInit = true
	} else {
		log.Println("Errors while checking file existence:", err)
		return
	}
	db, err := sql.Open("libsqlite3", db_path)
	if err != nil {
		log.Println("Could not open sqlite3:", err)
		return
	}
	defer db.Close()

	if shouldInit {
		_, err = db.Exec(carrotson.InitSql)
		if err != nil {
			log.Println("ERROR: couldn't init db:", err)
			return
		}
	}

	// carrotson.FeedMessageToCarrotson(db, "HELLO")
	// carrotson.FeedMessageToCarrotson(db, "HELP")
	// carrotson.FeedMessageToCarrotson(db, "HELL")
	// carrotson.FeedMessageToCarrotson(db, "HELLO KITTY")
	// carrotson.FeedMessageToCarrotson(db, "HELLO WORLD")
	// carrotson.FeedMessageToCarrotson(db, "MOM'S SPAGETI")
	
	b, err := os.ReadFile("../../bee.txt")
	temp := strings.Split(string(b),"\n")
    if err != nil {
        fmt.Println(err)
		return
    }

	for _, line := range temp {
		carrotson.FeedMessageToCarrotson(db, line)
	}
}
