package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/miguelduartepereira/Blog-Aggregator/internal"
	"github.com/miguelduartepereira/Blog-Aggregator/internal/config"
	"github.com/miguelduartepereira/Blog-Aggregator/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	_, ok := c.cmds[name]
	if !ok {
		c.cmds[name] = f
	}
}

func (c *commands) run(s *state, cmd command) error {
	executeCommand, ok := c.cmds[cmd.name]
	if ok {
		return executeCommand(s, cmd)
	}
	return fmt.Errorf("no command found")
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no arguments")
	}

	if len(cmd.args) != 1 {
		return fmt.Errorf("login must have just 1 argument")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err == sql.ErrNoRows {
		fmt.Println("no user found")
		os.Exit(1)
	}

	username := cmd.args[0]

	err = s.cfg.SetUser(username)

	if err != nil {
		return err
	}

	fmt.Println("Login successfull !")
	return nil

}

func handlerRegister(s *state, cmd command) error {
	if cmd.name == "" {
		return fmt.Errorf("no name passed")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])

	if err == nil {
		fmt.Println("user already exists")
		os.Exit(1)
	}

	if err != sql.ErrNoRows {
		fmt.Println(err)
		return err
	}

	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
	}

	createdUser, err := s.db.CreateUser(context.Background(), userParams)

	if err != nil {
		return err
	}

	if err := s.cfg.SetUser(createdUser.Name); err != nil {
		return err
	}

	fmt.Println("User created")
	fmt.Printf("%d %s\n", createdUser.ID, createdUser.Name)

	return nil

}

func handlerReset(s *state, cmd command) error {
	if cmd.name != "reset" {
		fmt.Println("wrong command")
		os.Exit(1)
	}

	err := s.db.EmptyUsers(context.Background())

	if err != nil {
		return fmt.Errorf("couldn't delete users: %w", err)
	}

	fmt.Println("Reset successfull")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetALlUsers(context.Background())
	if err != nil {
		return fmt.Errorf("couldn't retrieve users: %w", err)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func handlerAggregator(s *state, cmd command) error {
	interval, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", interval)
	ticker := time.NewTicker(interval)
	for ; ; <-ticker.C {
		err = scrapeFeeds(s)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
	}

}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		os.Exit(1)
		return fmt.Errorf("addfeed <name> <url>")
	}

	feedToCreate := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	}

	newFeed, err := s.db.CreateFeed(context.Background(), feedToCreate)
	if err != nil {
		return fmt.Errorf("error %w", err)
	}
	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    newFeed.ID,
		},
	)

	if err != nil {
		return fmt.Errorf("error %w", err)
	}

	fmt.Println(newFeed.ID)
	fmt.Println(newFeed.Name)
	fmt.Println(newFeed.CreatedAt)
	fmt.Println(newFeed.UpdatedAt)
	fmt.Println(newFeed.Url)
	fmt.Println(newFeed.UserID)

	return nil

}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error %w", err)
	}

	for _, feed := range feeds {
		fmt.Println(feed.Name)
		fmt.Println(feed.Url)
		fmt.Println(feed.UserName.String)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	feed, err := s.db.GetFeed(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error %w", err)
	}

	s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)

	if err != nil {
		return fmt.Errorf("error %w", err)
	}

	fmt.Println(feed.Name)
	fmt.Println(user.Name)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	userFeeds, err := s.db.GetFeedFollowForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	for _, feedNames := range userFeeds {
		fmt.Println(feedNames)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}

		return handler(s, cmd, user)

	}
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	feedToRemove, err := s.db.GetFeed(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	err = s.db.RemoveFeedFollow(
		context.Background(),
		database.RemoveFeedFollowParams{
			UserID: user.ID,
			FeedID: feedToRemove.ID,
		},
	)

	if err != nil {
		return fmt.Errorf("error: %w", err)

	}
	return nil
}

func scrapeFeeds(s *state) error {
	nextFeedToFetch, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("erro: %w", err)
	}
	s.db.MarkFeedFetched(context.Background(), nextFeedToFetch.ID)

	RSSFeed, err := internal.FetchFeed(
		context.Background(),
		nextFeedToFetch.Url,
	)

	if err != nil {
		return fmt.Errorf("erro: %w", err)
	}

	for _, post := range RSSFeed.Channel.Item {
		err := s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       post.Title,
				Description: post.Description,
				PublishedAt: post.PubDate,
				FeedID:      nextFeedToFetch.ID,
			},
		)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
	}
	return nil

}

func handleBrowse(s *state, cmd command, user database.User) error {
	var limit int32
	if len(cmd.args) == 0 {
		limit = 2
	} else {
		l, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("error %w", err)
		}
		limit = int32(l)
	}

	posts, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			UserID: user.ID,
			Limit:  limit,
		},
	)
	if err != nil {
		return fmt.Errorf("error %w", err)
	}

	for _, value := range posts {
		fmt.Println(value.Title)
		fmt.Println(value.Description)
		fmt.Println(value.PublishedAt)
	}
	return nil
}
