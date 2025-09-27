package main

// GOOS=js GOARCH=wasm go build -o main.wasm main.go

import (
	_ "embed"
	"fmt"
	"log"
	"regexp"
	"syscall/js"
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

//go:embed 02-carrotson.sql
var init_sql string

func main() {
	db := js.Global().Get("db")
	db.Call("run", init_sql)

	db.Call("run", "BEGIN;")
	db.Call("run", "INSERT INTO Carrotson_Branches (context, follows, frequency) VALUES (?, ?, 1) ON CONFLICT (context, follows) DO UPDATE SET frequency = Carrotson_Branches.frequency + 1;", []any{"WOAH", "ASS AHOY"})
	db.Call("run", "COMMIT;")
	FeedMessageToCarrotson(db, "HELLO")
	FeedMessageToCarrotson(db, "HELP")
	FeedMessageToCarrotson(db, "HELL")
	FeedMessageToCarrotson(db, "HELLO KITTY")
	FeedMessageToCarrotson(db, "HELLO WORLD")

	JSON := js.Global().Get("JSON")
	stmt := db.Call("prepare", "SELECT * FROM Carrotson_Branches")
	for stmt.Call("step").Bool() {
		row := stmt.Call("getAsObject");
		fmt.Println("Here is a row: " + JSON.Call("stringify", row).String())
	}

	message, err := CarrotsonGenerate(db, "HEL", 256)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	fmt.Println(maskDiscordPings(message))
}
