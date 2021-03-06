package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	discord "github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
)

const notFoundText = "Command not found"

const helpText = "**Yeetbot**\n" +
	"This bot yeets inactive users from your server, the following commands allow you to modify this behaviour.\n" +
	"Activity is based on message creation and on voice state events (joining voice channel, moving, leaving, etc.).\n" +
	"The bot will warn you on the halfway mark as well as the final day before you get kicked\n" +
	"\n" +
	"**Syntax**\n" +
	"!yeet <command> <args...>\n" +
	"\n" +
	"**Commands**\n" +
	"```\n" +
	" - help               | Shows this help dialog\n" +
	" - timeout            | Gets the timeout (in days) before a user gets kicked\n" +
	" - timeout (days)     | Sets the timeout (in days) before a user gets kicked\n" +
	" - warntimeout (days) | Sets the timeout (in days) before a user gets warned, set to -1 to show the warning at the halfway mark\n" +
	" - warntimeout        | Gets the timeout (in days) before a user gets warned\n" +
	" - kickmsg            | Gets the message displayed when a user gets kicked\n" +
	" - kickmsg (msg)      | Sets the message displayed when a user gets kicked\n" +
	" - warnmsg            | Gets the message displayed when a user gets warned\n" +
	" - warnmsg (msg)      | Sets the message displayed when a user gets warned\n" +
	" - isimmune (mention) | Gets the user's immunity to being kicked\n" +
	" - immune (mention)   | Toggles the user's immunity to being kicked\n" +
	" - forceadd           | Forces all users (that make sense) to be added to yeetbots internal timing list\n" +
	" - (mention)          | Forcefully yeets that person with a dumb message, you evil tater\n" +
	"```\n" +
	"**Bot written with <3 by Clipsey**\nSource: <https://github.com/Member1221/yeetbot>"

const cmdTag = "!yeet"

func UpdateServerCount(session *discord.Session) {
	serverCount := MongoClient.CountServers()

	// Set game playing, discard any errors
	err := session.UpdateStatus(0, fmt.Sprint("Yeeting on ", serverCount, " servers..."))

	if err != nil {
		log.Println(err)
	}
}

func HandleKickForGuild(session *discord.Session, guild *discord.Guild, guildData GuildData) {

	unixDay := int64(24 * time.Hour.Seconds())
	currentDay := time.Now().Unix() / unixDay
	lastUpdated := guildData.LastUpdated.Unix() / unixDay

	// Don't update the server multiple times a day
	if currentDay == lastUpdated {
		return
	}

	guildData.UpdateLastUpdated(time.Now().UTC())

	// Create a cursor over all the users on a server
	cur, err := MongoClient.UsersCollection().Find(context.Background(), bson.D{{"guildId", guild.ID}})
	if err != nil {
		log.Println(err)
	} else {
		defer cur.Close(context.Background())

		// No errors, iterate over all servers
		for cur.Next(context.Background()) {

			var result UserData
			err := cur.Decode(&result)
			if err != nil {
				log.Println(err)
				break
			}

			// Skip users whom are immune
			if result.Immune {
				continue
			}

			// Skip the owner of the server
			if result.UserId == guild.OwnerID {
				continue
			}

			// The bot really shouldn't be here, we'll delete it
			if result.UserId == SelfId {
				DeleteUser(result.GuildId, result.UserId)
				continue
			}

			lastActivity := result.LastActivity.Unix() / unixDay

			// Calculate and check day offsets
			dayOffset := currentDay - lastActivity
			halfwayMark := guildData.MaxDayInactivity / 2
			lastDay := guildData.MaxDayInactivity - 1

			// If the admin has specified a day offset for the warning use that instead
			if guildData.FirstWarnOffset >= 5 {
				halfwayMark = guildData.FirstWarnOffset
			}

			// Send warning messages at the halfway mark as well as the last day
			if dayOffset == halfwayMark || dayOffset == lastDay {
				channel, err := session.UserChannelCreate(result.UserId)
				if err == nil {
					timeRepl := strings.ReplaceAll(guildData.WarningMessage, "%time%", fmt.Sprint(guildData.MaxDayInactivity-dayOffset))
					serverRepl := strings.ReplaceAll(timeRepl, "%server%", guild.Name)

					session.ChannelMessageSend(channel.ID, serverRepl)
					continue
				}
			}

			// After time's up kick the user
			if dayOffset > guildData.MaxDayInactivity {

				timeRepl := strings.ReplaceAll(guildData.KickMessage, "%time%", strconv.FormatInt(guildData.MaxDayInactivity, 10))
				serverRepl := strings.ReplaceAll(timeRepl, "%server%", guild.Name)

				// Do the yeetin'
				yeet(session, result.GuildId, result.UserId, serverRepl, fmt.Sprintln("Inactivity for over ", guildData.MaxDayInactivity, " days. (Automated)"))
			}

		}
	}
}

func yeet(session *discord.Session, guildId, userId, message, reason string) {
	log.Println(fmt.Sprint("Yeeting ", userId, " due to inactivity..."))

	// Tell the user that they have been kicked
	channel, err := session.UserChannelCreate(userId)
	if err == nil {
		session.ChannelMessageSend(channel.ID, message)
	}

	// Proceed to kick the user and add a reason for the audit log
	err = session.GuildMemberDeleteWithReason(guildId, userId, reason)
	if err != nil {
		log.Println(err)
	}
}

func HandleUserJoin(session *discord.Session, user *discord.GuildMemberAdd) {

	// User does not exist, create them
	err := CreateUser(user.GuildID, user.User.ID, time.Now().UTC())
	if err != nil {

		// Something bad happened?
		log.Println(err)
		return
	}
}

func HandleUserLeave(session *discord.Session, user *discord.GuildMemberRemove) {

	// Try to delete a user from the db, if it fails it's fine
	_ = DeleteUser(user.GuildID, user.User.ID)
}

func HandleUserVoice(session *discord.Session, state *discord.VoiceStateUpdate) {

	// Dont count the bot's activity
	if state.UserID == SelfId {
		return
	}

	// Get the guild
	guild, err := session.Guild(state.GuildID)
	if err != nil {
		log.Println(err)
		return
	}

	// Don't count the owner's activity, they are automatically immune anyways
	if state.UserID == guild.OwnerID {
		return
	}

	// Get current UTC time
	stamp := time.Now().UTC()

	// Try to get the user
	user, err := GetUser(state.GuildID, state.UserID)
	if err != nil {

		// User does not exist, create them
		err := CreateUser(state.GuildID, state.UserID, stamp)
		if err != nil {

			// Something bad happened?
			log.Println(err)
			return
		}

		return
	}

	// Update the user's activity
	user.UpdateActivity(stamp)
}

func HandleSelfJoin(session *discord.Session, data *discord.GuildCreate) {

	// Make sure to reuse old guilds
	_, err := GetGuild(data.Guild.ID)
	if err != nil {

		// Try adding the guild
		log.Println("Adding server", data.Guild.ID, "...")
		err := CreateGuild(data.Guild.ID)
		if err != nil {

			// Something failed, delete the guild again also leave it
			session.GuildLeave(data.Guild.ID)
			err = DeleteGuild(data.Guild.ID)
			if err != nil {
				log.Println(err)
			}
		}

		// Update the server count
		UpdateServerCount(session)
	}
}

func HandleSelfLeave(session *discord.Session, data *discord.GuildDelete) {

	// Delete the data associated with the guild
	// We don't want to waste database space on it
	DeleteUsersForGuild(data.ID)
	DeleteGuild(data.ID)

	// Update the server count
	UpdateServerCount(session)
}

func HandleMessage(session *discord.Session, data *discord.MessageCreate) {

	const mercyText string = "yeetbot please have mercy"
	const memorialText string = "yeetbot memorial"
	const memorialBody string = "30th of June, 2020\n" +
		"_For the fallen whom got yeeted during The Great Yeetening of 2020_\n" +
		"🇫"

	// We DON'T want to handle the bot's messages.
	// Otherwise the bot would try to kick it self, lol.
	if data.Author.ID == SelfId {
		return
	}

	// Get the guild
	guild, err := session.Guild(data.GuildID)
	if err != nil {
		log.Println(err)
		return
	}

	// If the length of the message is long enough for a command
	// And if a command tag is the first thing, handle that command
	if len(data.Content) >= len(mercyText) &&
		strings.ToLower(data.Content[0:len(mercyText)]) == mercyText {

		// Stupid easter egg
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint(data.Author.Mention(), " no"))
	} else if len(data.Content) >= len(memorialText) &&
		strings.ToLower(data.Content[0:len(memorialText)]) == memorialText {

		// Stupid easter egg
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint(memorialBody))
	} else if len(data.Content) >= len(cmdTag) &&
		data.Content[0:len(cmdTag)] == cmdTag {

		handleCommand(session, data, guild)
	} else {
		// Otherwise update the user data for the message
		// If the user isn't present in db they will be created
		// The owner of the server is immune to this
		if data.Author.ID != guild.OwnerID {
			handleUpdateData(session, data)
		}
	}
}

func handleCommand(session *discord.Session, data *discord.MessageCreate, guild *discord.Guild) {

	// Delete commands sent by unaothorized users
	if data.Author.ID != guild.OwnerID {
		log.Println(data.Author.ID, guild.OwnerID)
		session.ChannelMessageDelete(data.ChannelID, data.ID)
		return
	}

	// Help text needed (for "!yeet")
	if len(data.Content) < len(cmdTag)+1 {
		session.ChannelMessageSend(data.ChannelID, helpText)
		return
	}

	// Split up command by spaces
	command := strings.Split(data.Content[len(cmdTag)+1:], " ")

	// Help text
	if len(command) == 0 || command[0] == "help" {
		session.ChannelMessageSend(data.ChannelID, helpText)
		return
	}

	// Gets the guild data
	guildData, err := GetGuild(data.GuildID)
	if err != nil {

		// Server *probably* doesn't exist in the DB for some reason
		log.Println(err)
		return
	}

	// Get which command is being run
	switch strings.ToLower(command[0]) {
	case "timeout":
		if len(command) == 1 {
			session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Kick timeout for this server is ", guildData.MaxDayInactivity, " days**"))
			return
		}

		value, err := strconv.ParseInt(command[1], 0, 64)
		if err != nil {
			session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**", err.Error(), "**"))
			break
		}

		err = guildData.UpdateMaxInactivity(value)
		if err != nil {
			log.Println(err)
			return
		}
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Kick timeout for this server set to ", guildData.MaxDayInactivity, " days**"))
		break

	case "warntimeout":

		if len(command) == 1 {

			// Get the current offset
			wOffset := fmt.Sprint(guildData.FirstWarnOffset)
			if guildData.FirstWarnOffset == -1 {
				wOffset = fmt.Sprint(guildData.MaxDayInactivity/2, " (auto)")
			}

			session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Warning timeout for this server is ", wOffset, " days**"))
			return
		}

		value, err := strconv.ParseInt(command[1], 0, 64)
		if err != nil {
			session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**", err.Error(), "**"))
			break
		}

		err = guildData.UpdateWarnOffset(value)
		if err != nil {
			session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**", err.Error(), "**"))
			break
		}

		// Get the current offset
		wOffset := fmt.Sprint(guildData.FirstWarnOffset)
		if guildData.FirstWarnOffset == -1 {
			wOffset = fmt.Sprint(guildData.MaxDayInactivity/2, " (auto)")
		}

		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Warning timeout for this server set to ", wOffset, " days**"))
		break

	case "kickmsg":

		if len(command) == 1 {
			session.ChannelMessageSend(data.ChannelID, guildData.KickMessage)
			return
		}

		msg := strings.Join(command[1:], " ")

		err = guildData.SetKickMsg(msg)
		if err != nil {
			log.Println(err)
			return
		}

		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Kick messaged updated**"))
		break

	case "warnmsg":

		if len(command) == 1 {
			session.ChannelMessageSend(data.ChannelID, guildData.KickMessage)
			return
		}

		msg := strings.Join(command[1:], " ")

		err = guildData.SetWarnMsg(msg)
		if err != nil {
			log.Println(err)
			return
		}

		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Warning messaged updated**"))
		break

	case "isimmune":
		if len(command) != 2 {
			session.ChannelMessageDelete(data.ChannelID, data.ID)
		}

		member := mentionToMember(session, guild.ID, command[1])
		if member == nil {

			// User was not found
			session.ChannelMessageSend(data.ChannelID, "User not found")
			return
		}

		// Get the guild user
		guildUser, err := guildData.GetUser(member.User.ID)
		if err != nil {
			log.Println(err)
			return
		}

		session.ChannelMessageDelete(data.ChannelID, data.ID)
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("User immunity is set to: ", guildUser.Immune))
		break

	case "immune":
		if len(command) != 2 {
			session.ChannelMessageDelete(data.ChannelID, data.ID)
		}

		member := mentionToMember(session, guild.ID, command[1])
		if member == nil {

			// User was not found
			session.ChannelMessageSend(data.ChannelID, "User not found")
			return
		}

		// Get the guild user
		guildUser, err := guildData.GetUser(member.User.ID)
		if err != nil {
			log.Println(err)
			return
		}

		err = guildUser.UpdateImmunity(!guildUser.Immune)
		if err != nil {
			log.Println(err)
			return
		}

		session.ChannelMessageDelete(data.ChannelID, data.ID)
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint(member.Mention(), " had their immunity is set to: ", guildUser.Immune))
		break

	case "forceadd":
		guild, err := session.Guild(data.GuildID)
		if err != nil {
			log.Println(err)
			return
		}

		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Force adding everyone...**"))
		amount := handleForceAdd(session, guild)
		session.ChannelMessageSend(data.ChannelID, fmt.Sprint("**Done, added ", amount, " users...**"))
		break

	default:

		member := mentionToMember(session, guild.ID, command[0])
		if member == nil {

			// Alert the user that the command was not found
			session.ChannelMessageSend(data.ChannelID, fmt.Sprint(command[0], ": ", notFoundText))
			session.ChannelMessageDelete(data.ChannelID, data.ID)

			// User was not found
			return
		} else {
			yeet(session, data.GuildID, member.User.ID, "**Thou hath been yeeteth by the server owner**", "Yeeted by owner")
		}

		break
	}
}

func handleUpdateData(session *discord.Session, data *discord.MessageCreate) {

	// Parse time stamp
	stamp, err := data.Timestamp.Parse()
	if err != nil {
		log.Println(err)
		return
	}

	// Try to get the user
	user, err := GetUser(data.GuildID, data.Author.ID)
	if err != nil {

		// User does not exist, create them
		err := CreateUser(data.GuildID, data.Author.ID, stamp)
		if err != nil {

			// Something bad happened?
			log.Println(err)
			return
		}

		return
	}

	// Update the user's activity
	user.UpdateActivity(stamp)
}

func getMemberList(session *discord.Session, guild *discord.Guild) []*discord.Member {

	// Scuffed list that has to reallocate a shitton of times
	// Discord provides no mechanism to get the amount of users easily.
	var memberList []*discordgo.Member = make([]*discord.Member, 0)

	// Value which tells discord which user precedes the ones we're getting next
	afterUser := ""

	for {
		// Gets a chunk of 1000 members
		members, err := session.GuildMembers(guild.ID, afterUser, 1000)

		// If it errors out we probably hit the end, break
		if err != nil {
			break
		}

		// Append members to member list
		memberList = append(memberList, members...)

		// If less than 1000 members were added then we're at the end of the list
		if len(members) < 1000 {
			break
		}

		// Make sure to set the "after-user" user
		afterUser = memberList[len(memberList)-1].User.ID
	}
	return memberList
}

func handleForceAdd(session *discord.Session, guild *discord.Guild) int {
	amount := 0

	memberList := getMemberList(session, guild)

	for _, member := range memberList {
		// Skip owner
		if member.User.ID == guild.OwnerID {
			continue
		}

		// Skip self
		if member.User.ID == SelfId {
			continue
		}

		currentTime := time.Now().UTC()

		// See if the user exists
		_, err := GetUser(guild.ID, member.User.ID)
		if err != nil {

			// User does not exist, create them
			err := CreateUser(guild.ID, member.User.ID, currentTime)
			if err != nil {

				// Something bad happened?
				log.Println(err)
				continue
			}

			amount++
		}
	}
	return amount
}

func mentionToMember(session *discordgo.Session, guildId, mention string) *discordgo.Member {

	// It wasn't a mention after all
	if mention[:2] != "<@" {
		return nil
	}

	mention = mention[2:]

	// It's a nickname mention
	if mention[0:1] == "!" {
		mention = mention[1:]
	}

	member, err := session.GuildMember(guildId, mention[:len(mention)-1])
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return member
}
