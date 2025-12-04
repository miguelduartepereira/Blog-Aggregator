package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"github.com/miguelduartepereira/Blog-Aggregator/internal/config"
	"github.com/miguelduartepereira/Blog-Aggregator/internal/database"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	dbQueries := database.New(db)

	newState := state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmdRegistry := commands{
		cmds: make(map[string]func(*state, command) error),
	}

	cmdRegistry.register("login", handlerLogin)
	cmdRegistry.register("register", handlerRegister)
	cmdRegistry.register("reset", handlerReset)
	cmdRegistry.register("users", handlerUsers)
	cmdRegistry.register("agg", handlerAggregator)
	cmdRegistry.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmdRegistry.register("feeds", handlerFeeds)
	cmdRegistry.register("follow", middlewareLoggedIn(handlerFollow))
	cmdRegistry.register("following", middlewareLoggedIn(handlerFollowing))
	cmdRegistry.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmdRegistry.register("browse", middlewareLoggedIn(handleBrowse))

	var cmd command
	if len(os.Args) < 2 {
		cmd = command{
			name: os.Args[1],
			args: nil,
		}
	} else {
		name := os.Args[1]
		args := os.Args[2:]

		cmd = command{
			name: name,
			args: args,
		}
	}

	err = cmdRegistry.run(&newState, cmd)
	if err != nil {
		fmt.Println(err)
	}
}
