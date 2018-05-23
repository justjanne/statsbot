package main

import (
	"github.com/lrstanley/girc"
	"os"
	"fmt"
	"strings"
	"log"
	"time"
	"golang.org/x/crypto/sha3"
	_ "github.com/lib/pq"
	"encoding/hex"
	"database/sql"
	"strconv"
)

type Config struct {
	Database DatabaseConfig
	Irc      IrcConfig
}

type IrcConfig struct {
	Server       string
	Port         int
	Secure       bool
	Nick         string
	Ident        string
	Realname     string
	SaslAccount  string
	SaslPassword string
	SaslEnabled  bool
}

type DatabaseConfig struct {
	Format string
	Url    string
}

func NewConfigFromEnv() Config {
	var err error
	config := Config{}

	config.Irc.Server = os.Getenv("KSTATS_IRC_SERVER")
	config.Irc.Port, err = strconv.Atoi(os.Getenv("KSTATS_IRC_PORT"))
	if err != nil {
		panic(err)
	}
	config.Irc.Secure = os.Getenv("KSTATS_IRC_SECURE") == "true"
	config.Irc.Nick = os.Getenv("KSTATS_IRC_NICK")
	config.Irc.Ident = os.Getenv("KSTATS_IRC_IDENT")
	config.Irc.Realname = os.Getenv("KSTATS_IRC_REALNAME")
	config.Irc.SaslEnabled = os.Getenv("KSTATS_IRC_SASL_ENABLED") == "true"
	config.Irc.SaslAccount = os.Getenv("KSTATS_IRC_SASL_ACCOUNT")
	config.Irc.SaslPassword = os.Getenv("KSTATS_IRC_SASL_PASSWORD")

	config.Database.Format = os.Getenv("KSTATS_DATABASE_TYPE")
	config.Database.Url = os.Getenv("KSTATS_DATABASE_URL")

	return config
}

type IrcMessage struct {
	Time        time.Time
	Channel     int
	Sender      string
	Words       int
	Characters  int
	Question    bool
	Exclamation bool
	Caps        bool
	Aggression  bool
	EmojiHappy  bool
	EmojiSad    bool
}

func (m *IrcMessage) ToString() string {
	var flags []string
	if m.Question {
		flags = append(flags, "Question")
	}
	if m.Exclamation {
		flags = append(flags, "Exclamation")
	}
	if m.Caps {
		flags = append(flags, "Caps")
	}
	if m.Aggression {
		flags = append(flags, "Aggression")
	}
	if m.EmojiHappy {
		flags = append(flags, "EmojiHappy")
	}
	if m.EmojiSad {
		flags = append(flags, "EmojiSad")
	}

	return fmt.Sprintf("IrcMessage{time=%s,channel=%d,sender=%s,words=%d,characters=%d,flags=[%s]}", m.Time.Format(time.RFC3339), m.Channel, m.Sender, m.Words, m.Characters, strings.Join(flags, ","))
}

func main() {
	config := NewConfigFromEnv()

	db, err := sql.Open(config.Database.Format, config.Database.Url)
	if err != nil {
		panic(err)
	}

	ircConfig := girc.Config{
		Server: config.Irc.Server,
		Port:   config.Irc.Port,
		SSL:    config.Irc.Secure,
		Nick:   config.Irc.Nick,
		User:   config.Irc.Ident,
		Name:   config.Irc.Realname,
		Debug:  os.Stdout,
	}
	if config.Irc.SaslEnabled {
		ircConfig.SASL = &girc.SASLPlain{
			User: config.Irc.SaslAccount,
			Pass: config.Irc.SaslPassword,
		}
	}
	client := girc.New(ircConfig)

	channels := map[string]int{}
	client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		result, err := db.Query("SELECT id, channel FROM channels")
		if err != nil {
			panic(err)
		}
		for result.Next() {
			var id int
			var name string
			err := result.Scan(&id, &name)
			if err != nil {
				panic(err)
			}
			channels[name] = id
		}
		for name := range channels {
			c.Cmd.Join(name)
		}
	})

	client.Handlers.Add(girc.PRIVMSG, func(c *girc.Client, e girc.Event) {
		if len(e.Params) == 1 {
			channel := e.Params[0]
			if id, ok := channels[channel]; ok {
				name := hex.EncodeToString(sha3.New256().Sum([]byte(e.Source.Name)))
				content := strings.TrimSpace(e.Trailing)
				// Add referenced nick part here
				// c.LookupChannel(channel).UserList
				message := IrcMessage{
					Time:        time.Now().UTC(),
					Channel:     id,
					Sender:      name,
					Words:       len(strings.Split(content, " ")),
					Characters:  len(content),
					Question:    strings.HasSuffix(content, "?"),
					Exclamation: strings.HasSuffix(content, "!"),
					Caps:        content == strings.ToUpper(content),
					Aggression:  false,
					EmojiHappy:  strings.Contains(content, ":)"),
					EmojiSad:    strings.Contains(content, ":("),
				}
				_, err := db.Exec("INSERT INTO messages (time, channel, sender, words, characters, question, exclamation, caps, aggression, emoji_happy, emoji_sad) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)", message.Time, message.Channel, message.Sender, message.Words, message.Characters, message.Question, message.Exclamation, message.Caps, message.Aggression, message.EmojiHappy, message.EmojiSad)
				if err != nil {
					println(err.Error())
				}
			}
		}
	})

	for {
		if err := client.Connect(); err != nil {
			log.Printf("error: %s", err)

			log.Println("reconnecting in 30 seconds...")
			time.Sleep(30 * time.Second)
		} else {
			return
		}
	}
}
