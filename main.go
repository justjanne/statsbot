package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/lrstanley/girc"
	"golang.org/x/crypto/scrypt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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

type IrcChannel struct {
	Id   int
	Name string
	Salt string
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

func hashName(salt string, name string) string {
	hash, err := scrypt.Key([]byte(strings.ToLower(name)), []byte(salt), 32768, 8, 1, 32)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash)
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
	}
	if config.Irc.SaslEnabled {
		ircConfig.SASL = &girc.SASLPlain{
			User: config.Irc.SaslAccount,
			Pass: config.Irc.SaslPassword,
		}
	}
	client := girc.New(ircConfig)

	channels := map[string]IrcChannel{}
	client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		result, err := db.Query("SELECT id, channel, salt FROM channels")
		if err != nil {
			panic(err)
		}
		for result.Next() {
			var id int
			var name string
			var salt string
			err := result.Scan(&id, &name, &salt)
			if err != nil {
				panic(err)
			}
			channels[name] = IrcChannel{
				Id:   id,
				Name: name,
				Salt: salt,
			}
		}
		for name := range channels {
			fmt.Printf("Joining %s\n", name)
			c.Cmd.Join(name)
		}
	})

	client.Handlers.Add(girc.PRIVMSG, func(client *girc.Client, event girc.Event) {
		if len(event.Params) == 1 {
			channelName := event.Params[0]
			if channelData, ok := channels[channelName]; ok {
				logMessage(channelData, event, client, channelName, db)
			} else if channelName == client.GetNick() {
				handlePrivateMessage(channels, event, client, db)
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

func handlePrivateMessage(channels map[string]IrcChannel, event girc.Event, client *girc.Client, db *sql.DB) {
	split := strings.Split(event.Trailing, " ")
	if len(split) >= 1 {
		command := split[0]
		parameters := split[1:]
		if strings.EqualFold(command, "OPT-IN") {
			if len(parameters) == 1 {
				channelName := parameters[0]
				if channelData, ok := channels[channelName]; ok {
					nick := event.Source.Name
					hash := hashName(channelData.Salt, nick)
					_, err := db.Exec("INSERT INTO users (hash, nick) VALUES ($1, $2)", hash, nick)
					if err != nil {
						client.Cmd.Reply(event, "An error has occured, please try later again")
						println(err.Error())
					} else {
						client.Cmd.Reply(event, "Opt-In successful")
					}
					return
				}
				client.Cmd.Reply(event, "Channel not found")
			}
			printUsageOptIn(client, event)
			return
		} else if strings.EqualFold(command, "OPT-OUT") {
			if len(parameters) == 1 {
				channelName := parameters[0]
				if channelData, ok := channels[channelName]; ok {
					hash := hashName(channelData.Salt, event.Source.Name)
					_, err := db.Exec("DELETE FROM users WHERE hash = $1", hash)
					if err != nil {
						client.Cmd.Reply(event, "An error has occured, please try later again")
						println(err.Error())
					} else {
						client.Cmd.Reply(event, "Opt-Out successful")
					}
					return
				}
				client.Cmd.Reply(event, "Channel not found")
			}
			printUsageOptOut(client, event)
			return
		}
	}
	printUsage(client, event)
}

func printUsage(client *girc.Client, event girc.Event) {
	client.Cmd.Reply(event, "Usage:")
	client.Cmd.Reply(event, "OPT-IN [channel]")
	client.Cmd.Reply(event, "OPT-OUT [channel]")
}

func printUsageOptIn(client *girc.Client, event girc.Event) {
	client.Cmd.Reply(event, "Usage: OPT-IN [channel]")
}

func printUsageOptOut(client *girc.Client, event girc.Event) {
	client.Cmd.Reply(event, "Usage: OPT-OUT [channel]")
}

func logMessage(channelData IrcChannel, event girc.Event, client *girc.Client, channelName string, db *sql.DB) {
	now := time.Now().UTC()
	name := hashName(channelData.Salt, event.Source.Name)
	content := strings.TrimSpace(event.Trailing)
	channel := client.LookupChannel(channelName)
	var users []string
	if channel != nil {
		for _, user := range channel.UserList {
			if strings.Contains(content, user) {
				users = append(users, hashName(channelData.Salt, user))
			}
		}
	}
	for _, user := range users {
		_, err := db.Exec("INSERT INTO \"references\" (channel, time, source, target) VALUES ($1, $2, $3, $4)", channelData.Id, now, name, user)
		if err != nil {
			println(err.Error())
		}
	}
	message := IrcMessage{
		Time:        now,
		Channel:     channelData.Id,
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
