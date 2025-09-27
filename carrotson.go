package main

import (
	"errors"
	"math"
	"syscall/js"
)

const ContextSize = 8

type Path struct {
	context []rune
	follows rune
}

func splitMessageIntoPaths(message []rune) (branches []Path) {
	for i := -ContextSize; i+ContextSize < len(message); i += 1 {
		j := i
		if j < 0 {
			j = 0
		}
		branches = append(branches, Path{
			context: message[j : i+ContextSize],
			follows: message[i+ContextSize],
		})
	}
	return
}

type Branch struct {
	Context   []rune
	Follows   rune
	Frequency int64
}

var (
	EmptyFollowsError = errors.New("Empty follows of a Carrotson branch")
)

func QueryRandomBranchFromUnfinishedContext(db js.Value, context []rune) (*Branch, error) {
	rows := db.Call("exec", "SELECT context, follows, frequency FROM Carrotson_Branches WHERE context LIKE ? AND frequency > 0 ORDER BY random() LIMIT 1", []any{string(context)+"%"})
	row := rows.Index(0)
	if row.IsNull() || row.IsUndefined() {
		return nil, nil
	}
	values := row.Get("values").Index(0)
	var fullContext string = values.Index(0).String()
	var follows string = values.Index(1).String()
	var frequency int64 = int64(values.Index(2).Int())

	if len(follows) == 0 {
		return nil, EmptyFollowsError
	}
	return &Branch{
		Context:   []rune(fullContext),
		Follows:   []rune(follows)[0],
		Frequency: frequency,
	}, nil
}

func QueryRandomBranchFromContext(db js.Value, context []rune, t float64) (*Branch, error) {
	rows := db.Call("exec", "select follows, frequency from (select * from carrotson_branches where context = ? AND frequency > 0 order by frequency desc limit CEIL((select count(*) from carrotson_branches where context = ? AND frequency > 0)*1.0*?)) as c order by random() limit 1", []any{string(context), string(context), t})
	row := rows.Index(0)
	if row.IsNull() || row.IsUndefined() {
		return nil, nil
	}
	values := row.Get("values").Index(0)
	var follows string = values.Index(0).String()
	var frequency int64 = int64(values.Index(1).Int())
		
	if len(follows) == 0 {
		return nil, EmptyFollowsError
	}
	return &Branch{
		Context:   context,
		Follows:   []rune(follows)[0],
		Frequency: frequency,
	}, nil
}

func QueryBranchesFromContext(db js.Value, context []rune) ([]Branch, error) {
	stmt := db.Call("prepare", "SELECT follows, frequency FROM Carrotson_Branches WHERE context = $1 AND frequency > 0")
	stmt.Call("bind", map[string]any{"$1": string(context)})

	branches := []Branch{}
	for stmt.Call("step").Bool() {
		branch := Branch{}
		var follows string
		row := stmt.Call("getAsObject")
		follows = row.Get("follows").String()
		branch.Frequency = int64(row.Get("follows").Int())
		if len(follows) == 0 {
			return nil, EmptyFollowsError
		}
		branch.Follows = []rune(follows)[0]
		branches = append(branches, branch)
	}
	return branches, nil
}

func ContextOfMessage(message []rune) []rune {
	i := len(message) - ContextSize
	if i < 0 {
		i = 0
	}
	return message[i:len(message)]
}

func CarrotsonGenerate(db js.Value, prefix string, limit int) (string, error) {
	var err error = nil
	var branch *Branch
	message := []rune(prefix)
	t := float64(len(message)) / float64(limit)
	if len(message) >= ContextSize || len(message) == 0 {
		branch, err = QueryRandomBranchFromContext(db, ContextOfMessage(message), (math.Cos(t*math.Pi*1.5)+1.0)/2.0)
	} else {
		branch, err = QueryRandomBranchFromUnfinishedContext(db, ContextOfMessage(message))
		if err == nil && branch != nil {
			message = branch.Context
		}
	}
	for err == nil && branch != nil && len(message) < limit {
		message = append(message, branch.Follows)
		t = float64(len(message)) / float64(limit)
		branch, err = QueryRandomBranchFromContext(db, ContextOfMessage(message), (math.Cos(t*math.Pi*1.5)+1.0)/2.0)
	}
	return string(message), err
}

func FeedMessageToCarrotson(db js.Value, message string) {
	db.Call("run", "BEGIN;")

	for _, path := range splitMessageIntoPaths([]rune(message)) {
		db.Call("run", "INSERT INTO Carrotson_Branches (context, follows, frequency) VALUES (?, ?, 1) ON CONFLICT (context, follows) DO UPDATE SET frequency = Carrotson_Branches.frequency + 1;", []any{string(path.context), string([]rune{path.follows})})
	}
	db.Call("run", "COMMIT;")
}
