package main

import (
	"database/sql"
	"errors"
	"fmt"
	// "github.com/crtsn/crtsn/carrotson"
	carrotson "github.com/crtsn/crtsn/traced_carrotson"
	"log"
	"math"
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
	db_path := "test.sqlite"
	shouldRemove := true
	shouldInit := true
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

	/*carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: A")
	carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: B")
	carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: B")
	carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: C")
	carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: D")
	carrotson.FeedMessageToCarrotson(db, "SPAMIUSHA: E")

	rows, err := db.Query("SELECT context, follows, frequency FROM Carrotson_Branches")
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		branch := carrotson.Branch{}
		var fullContext sql.NullString
		var follows sql.NullString
		var frequency sql.NullInt64

		if err := rows.Scan(&fullContext, &follows, &frequency); err != nil {
			log.Printf("%s\n", err)
			return
		}

		if fullContext.Valid {
		    branch.Context = []rune(fullContext.String)
		}
		if follows.Valid {
		    branch.Follows = []rune(follows.String)[0]
		}
		if frequency.Valid {
        	branch.Frequency = frequency.Int64
    	}
		fmt.Printf("%-8s|%1c|%-2d\n", string(branch.Context), branch.Follows, branch.Frequency)
	}

	message, err := carrotson.CarrotsonGenerate(db, "SPAMIUSHA", 256)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))

	message, err = carrotson.CarrotsonGenerate(db, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA SPAMIUSHA", 256)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))

	message, err = carrotson.CarrotsonGenerate(db, "I LIKE SPENDING MOST OF MY DAY PLAYING WITH MY FAVORITE THING IN THE WHOLE WORLD. IT IS CALLED \"CARROT\"!!!!!!! (C) SPAMIUSHA", 256)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))*/

	// links := make([]string, 20)
	// for i := range links {
	// 	links[i] = fmt.Sprintf("http://Y%c.example", 'A'+i)
	// 	carrotson.FeedMessageToCarrotson(db, links[i])
	// }
	// carrotson.FeedMessageToCarrotson(db, links[1])

	context := "http://Y"
	prefixLength := int(math.Ceil(512 / (3 * math.Pi) * math.Acos(2/float64(len(links))-1)))
	prefix := strings.Repeat("A", prefixLength-len([]rune(context))) + context
	message, err := carrotson.CarrotsonGenerate(db, prefix, 256)
 	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))
}
