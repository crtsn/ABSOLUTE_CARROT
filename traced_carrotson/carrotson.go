package traced_carrotson

import (
	"database/sql"
	"errors"
	"log"
	"math"
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

func TraceDecision(format string, args ...interface{}) {
	log.Printf("TRACE carrotson: "+format, args...)
}

func QueryRandomBranchFromUnfinishedContext(db *sql.DB, context []rune) (*Branch, error) {
	TraceDecision("QueryRandomBranchFromUnfinishedContext: querying starts_with(context, %q)", string(context))
	row := db.QueryRow("SELECT context, follows, frequency FROM Carrotson_Branches WHERE starts_with(context, $1) AND frequency > 0 ORDER BY random() LIMIT 1", string(context))
	var fullContext string
	var follows string
	var frequency int64
	err := row.Scan(&fullContext, &follows, &frequency)
	if err == sql.ErrNoRows {
		TraceDecision("QueryRandomBranchFromUnfinishedContext: no rows for context starting with %q", string(context))
		return nil, nil
	}
	if err != nil {
		TraceDecision("QueryRandomBranchFromUnfinishedContext: error for context starting with %q: %v", string(context), err)
		return nil, err
	}
	if len(follows) == 0 {
		TraceDecision("QueryRandomBranchFromUnfinishedContext: empty follows for context %q", fullContext)
		return nil, EmptyFollowsError
	}
	TraceDecision("QueryRandomBranchFromUnfinishedContext: picked context=%q follows=%c frequency=%d", fullContext, []rune(follows)[0], frequency)
	return &Branch{
		Context:   []rune(fullContext),
		Follows:   []rune(follows)[0],
		Frequency: frequency,
	}, nil
}

func QueryRandomBranchFromContext(db *sql.DB, context []rune, t float64) (*Branch, error) {
	TraceDecision("QueryRandomBranchFromContext: querying context=%q t=%f", string(context), t)
	row := db.QueryRow("select follows, frequency from (select * from carrotson_branches where context = $1 AND frequency > 0 order by frequency desc limit CEIL((select count(*) from carrotson_branches where context = $1 AND frequency > 0)*1.0*$2)) as c order by random() limit 1", string(context), t)
	var follows string
	var frequency int64
	err := row.Scan(&follows, &frequency)
	if err == sql.ErrNoRows {
		TraceDecision("QueryRandomBranchFromContext: no rows for context=%q t=%f", string(context), t)
		return nil, nil
	}
	if err != nil {
		TraceDecision("QueryRandomBranchFromContext: error for context=%q t=%f: %v", string(context), t, err)
		return nil, err
	}
	if len(follows) == 0 {
		TraceDecision("QueryRandomBranchFromContext: empty follows for context=%q", string(context))
		return nil, EmptyFollowsError
	}
	TraceDecision("QueryRandomBranchFromContext: picked context=%q follows=%c frequency=%d", string(context), []rune(follows)[0], frequency)
	return &Branch{
		Context:   context,
		Follows:   []rune(follows)[0],
		Frequency: frequency,
	}, nil
}

func QueryBranchesFromContext(db *sql.DB, context []rune) ([]Branch, error) {
	TraceDecision("QueryBranchesFromContext: querying context=%q", string(context))
	rows, err := db.Query("SELECT follows, frequency FROM Carrotson_Branches WHERE context = $1 AND frequency > 0", string(context))
	if err != nil {
		TraceDecision("QueryBranchesFromContext: error for context=%q: %v", string(context), err)
		return nil, err
	}
	branches := []Branch{}
	for rows.Next() {
		branch := Branch{}
		var follows string
		err = rows.Scan(&follows, &branch.Frequency)
		if err != nil {
			TraceDecision("QueryBranchesFromContext: scan error for context=%q: %v", string(context), err)
			return nil, err
		}
		if len(follows) == 0 {
			TraceDecision("QueryBranchesFromContext: empty follows for context=%q", string(context))
			return nil, EmptyFollowsError
		}
		branch.Follows = []rune(follows)[0]
		TraceDecision("QueryBranchesFromContext: got branch context=%q follows=%c frequency=%d", string(context), branch.Follows, branch.Frequency)
		branches = append(branches, branch)
	}
	TraceDecision("QueryBranchesFromContext: returning %d branches for context=%q", len(branches), string(context))
	return branches, nil
}

func ContextOfMessage(message []rune) []rune {
	i := len(message) - ContextSize
	if i < 0 {
		i = 0
	}
	context := message[i:len(message)]
	TraceDecision("ContextOfMessage: message=%q -> context=%q", string(message), string(context))
	return context
}

func CarrotsonGenerate(db *sql.DB, prefix string, limit int) (string, error) {
	TraceDecision("CarrotsonGenerate: start prefix=%q limit=%d", prefix, limit)
	var err error = nil
	var branch *Branch
	message := []rune(prefix)
	t := float64(len(message)) / float64(limit)
	if len(message) >= ContextSize || len(message) == 0 {
		TraceDecision("CarrotsonGenerate: message len=%d >= ContextSize=%d or empty, using full context query, t=%f", len(message), ContextSize, t)
		branch, err = QueryRandomBranchFromContext(db, ContextOfMessage(message), (math.Cos(t*math.Pi*1.5)+1.0)/2.0)
	} else {
		TraceDecision("CarrotsonGenerate: message len=%d < ContextSize=%d, using unfinished context query", len(message), ContextSize)
		branch, err = QueryRandomBranchFromUnfinishedContext(db, ContextOfMessage(message))
		if err == nil && branch != nil {
			TraceDecision("CarrotsonGenerate: expanding message %q to full context %q", string(message), string(branch.Context))
			message = branch.Context
		}
	}
	for err == nil && branch != nil && len(message) < limit {
		TraceDecision("CarrotsonGenerate: appending follows=%c message_len=%d limit=%d", branch.Follows, len(message), limit)
		message = append(message, branch.Follows)
		t = float64(len(message)) / float64(limit)
		branch, err = QueryRandomBranchFromContext(db, ContextOfMessage(message), (math.Cos(t*math.Pi*1.5)+1.0)/2.0)
	}
	if err != nil {
		TraceDecision("CarrotsonGenerate: stopping with error: %v", err)
	} else if branch == nil {
		TraceDecision("CarrotsonGenerate: stopping, no branch found, message_len=%d limit=%d", len(message), limit)
	} else {
		TraceDecision("CarrotsonGenerate: stopping, limit=%d reached, message_len=%d", limit, len(message))
	}
	TraceDecision("CarrotsonGenerate: done, generated %q", string(message))
	return string(message), err
}

func FeedMessageToCarrotson(db *sql.DB, message string) {
	TraceDecision("FeedMessageToCarrotson: feeding message=%q", message)
	tx, err := db.Begin()
	if err != nil {
		TraceDecision("FeedMessageToCarrotson: could not start transaction: %v", err)
		log.Println("ERROR: feedMessageToCarrotson: could not start transaction:", err)
		return
	}
	TraceDecision("FeedMessageToCarrotson: transaction started")
	for _, path := range splitMessageIntoPaths([]rune(message)) {
		_, err := tx.Exec("INSERT INTO Carrotson_Branches (context, follows, frequency) VALUES ($1, $2, 1) ON CONFLICT (context, follows) DO UPDATE SET frequency = Carrotson_Branches.frequency + 1;", string(path.context), string([]rune{path.follows}))
		if err != nil {
			TraceDecision("FeedMessageToCarrotson: could not insert path context=%q follows=%c: %v", string(path.context), path.follows, err)
			log.Println("ERROR: feedMessageToCarrotson: could not insert element", string(path.context), string([]rune{path.follows}), ":", err)
			err := tx.Rollback()
			if err != nil {
				TraceDecision("FeedMessageToCarrotson: could not rollback transaction after failure: %v", err)
				log.Println("ERROR: feedMessageToCarrotson: could not rollback transaction after failure:", err)
			} else {
				TraceDecision("FeedMessageToCarrotson: transaction rolled back")
			}
			return
		}
		TraceDecision("FeedMessageToCarrotson: fed path context=%q follows=%c", string(path.context), path.follows)
	}
	err = tx.Commit()
	if err != nil {
		TraceDecision("FeedMessageToCarrotson: could not commit transaction: %v", err)
		log.Println("ERROR: feedMessageToCarrotson: could not commit transaction:", err)
	} else {
		TraceDecision("FeedMessageToCarrotson: transaction committed")
	}
}
