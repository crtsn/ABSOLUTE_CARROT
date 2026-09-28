package main

import (
	"database/sql"
	_ "fmt"
	"github.com/crtsn/crtsn/carrotson"
	"log"
	"regexp"
	"syscall/js"

	_ "github.com/crtsn/crtsn/sql/sqljs"
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

func feed_carrot(this js.Value, args []js.Value) any {
	food := args[0].String()
	carrotson.FeedMessageToCarrotson(db, food)

	// JSON := js.Global().Get("JSON")
	// stmt := db.Call("prepare", "SELECT * FROM Carrotson_Branches")
	// for stmt.Call("step").Bool() {
	// 	row := stmt.Call("getAsObject");
	// 	fmt.Println("Here is a row: " + JSON.Call("stringify", row).String())
	// }
	return nil
}

func carrot_generate(this js.Value, args []js.Value) any {
	arg := ""
	if len(args) >= 1 {
		arg = args[0].String()
	}
	message, err := carrotson.CarrotsonGenerate(db, arg, 70)
	if err != nil {
		log.Printf("%s\n", err)
		return nil
	}
	return message
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqljs", "db")
	if err != nil {
		log.Println("Could not open sqljs:", err)
		return
	}
	defer db.Close()

	window := js.Global().Get("window")

	window.Set("carrot_generate", js.FuncOf(carrot_generate))
	window.Set("feed_carrot", js.FuncOf(feed_carrot))

	select {}
}
