package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	bot "github.com/Member1221/yeetbot/bot"
	discord "github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {

	// Load config json
	cfgstr, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// Marshal config json to config struct
	var config bot.ConfigData
	err = json.Unmarshal(cfgstr, &config)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to database
	err = bot.ConnectToDB(config.ConnectionString)
	defer bot.MongoClient.Disconnect()
	if err != nil {
		log.Fatal(err)
	}

	// Log on to discord with bot token
	session, err := discord.New("Bot " + config.Token)
	session.Identify.Intents = discord.MakeIntent(discord.IntentsAllWithoutPrivileged | discord.IntentsGuildMembers)
	defer session.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Add event handlers
	log.Println("Adding event handlers...")
	session.AddHandler(bot.HandleMessage)
	session.AddHandler(bot.HandleUserJoin)
	session.AddHandler(bot.HandleUserLeave)
	session.AddHandler(bot.HandleUserVoice)
	session.AddHandler(bot.HandleSelfJoin)
	session.AddHandler(bot.HandleSelfLeave)

	// Session that does server updates.
	session.AddHandler(func(s *discord.Session, ready *discord.Ready) {

		// Try to get the bots user instance
		self, err := s.User("@me")
		if err != nil {
			log.Fatal(err)
		}

		// Get its ID
		bot.SelfId = self.ID

		// Scan servers
		log.Println("Scanning for missed servers...")
		scanServers(s)

		// Begin the loop that occasionally kicks inactive people
		log.Println("Bot started...")
		go updateServers(s)
	})

	// Connect to discord
	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func scanServers(session *discord.Session) {
	for _, guild := range session.State.Guilds {

		// Make sure guild exists in database.
		_, err := bot.GetGuild(guild.ID)
		if err != nil {

			log.Println("Missed server", guild.ID, "...")
			log.Println(err)

			err = bot.CreateGuild(guild.ID)
			if err != nil {

				// Something bad happened?
				log.Println(err)
				return
			}
		}
	}
}

func updateServers(session *discord.Session) {
	for {

		// Update the server count
		bot.UpdateServerCount(session)

		// Create a cursor over all the servers
		cur, err := bot.MongoClient.ServersCollection().Find(context.Background(), bson.D{})
		if err != nil {
			log.Println(err)
		} else {

			// No errors, iterate over all servers
			for cur.Next(context.Background()) {

				var result bot.GuildData
				err := cur.Decode(&result)
				if err != nil {
					log.Println(err)
					break
				}

				// Get Discord guild
				guild, err := session.Guild(result.GuildId)
				if err != nil {
					log.Println(err)
					break
				}

				// Handle kicking for the guild
				bot.HandleKickForGuild(session, guild, result)

				// Sleep 1 minute between each server update
				time.Sleep(1 * time.Minute)
			}
		}

		// Update server lisiting every 10 minutes
		cur.Close(context.Background())
		time.Sleep(1 * time.Hour)
	}
}
