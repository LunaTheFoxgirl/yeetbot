package main

import (
	"context"
	"encoding/json"
	"fmt"
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

var session *discord.Session

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
	session, err = discord.New("Bot " + config.Token)
	defer session.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Add event handlers
	log.Println("Adding event handlers")
	session.AddHandler(bot.HandleMessage)
	session.AddHandler(bot.HandleUserJoin)
	session.AddHandler(bot.HandleUserLeave)
	session.AddHandler(bot.HandleSelfJoin)

	// Scan servers
	log.Println("Scanning for missed servers...")
	scanServers()

	// Begin the loop that occasionally kicks inactive people
	log.Println("Bot started...")
	go updateServers()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func scanServers() {
	for _, guild := range session.State.Guilds {

		// Make sure guild exists in database.
		_, err := bot.GetGuild(guild.ID)
		if err != nil {

			err = bot.CreateGuild(guild.ID)
			if err != nil {

				// Something bad happened?
				log.Println(err)
				return
			}
		}
	}
}

func updateServers() {
	for {

		serverCount := bot.MongoClient.CountServers()

		// Set game playing, discard any errors
		_ = session.UpdateStatusComplex(discord.UpdateStatusData{
			IdleSince: nil,
			Game:      nil,
			AFK:       false,
			Status:    fmt.Sprint("Yeeting on ", serverCount, "servers... !yeet help for help"),
		})

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
		time.Sleep(10 * time.Minute)
	}
}
