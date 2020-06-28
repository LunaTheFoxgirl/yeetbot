package bot

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type ConfigData struct {
	Token            string `json:"token"`
	ConnectionString string `json:"connectionString"`
}

func createGuild(guildId string) GuildData {
	var guildData GuildData
	guildData.KickMessage = "**You have been yeeted from %time% due to inactivity.**"
	guildData.GuildId = guildId
	guildData.MaxDayInactivity = 30
	return guildData
}

func CreateGuild(guildId string) error {
	data := createGuild(guildId)
	_, err := MongoClient.ServersCollection().InsertOne(context.Background(), data)
	if err != nil {
		return err
	}
	return nil
}

func DeleteGuild(guildId string) error {
	_, err := MongoClient.ServersCollection().DeleteOne(context.Background(), bson.D{{"guildId", guildId}})
	if err != nil {
		return err
	}
	return nil
}

func GetGuild(guildId string) (*GuildData, error) {
	var guildData *GuildData = new(GuildData)

	// Try finding the user
	result := MongoClient.ServersCollection().FindOne(context.Background(), bson.D{{"guildId", guildId}})
	err := result.Err()
	if err != nil {
		return nil, err
	}

	// Decode and return
	err = result.Decode(guildData)
	if err != nil {
		return nil, err
	}

	// Return decoded data
	return guildData, nil
}

type GuildData struct {
	KickMessage      string `bson:"kickmsg"`
	GuildId          string `bson:"guildId"`
	MaxDayInactivity int64  `bson:"dayInactivity"`
}

func (self *GuildData) UpdateMaxInactivity(days int64) error {
	filter := bson.D{{"guildId", self.GuildId}}

	// We don't want to just instakick everybody
	// Minimum is 5 days
	if days < 5 {
		days = 5
	}

	// Over a year is a bit long
	if days > 365 {
		days = 365
	}

	self.MaxDayInactivity = days

	// Update database
	_, err := MongoClient.ServersCollection().UpdateOne(context.Background(), filter, *self)
	if err != nil {
		return err
	}
	return nil
}

func (self *GuildData) DeleteUser(userId string) error {
	return DeleteUser(self.GuildId, userId)
}

func (self *GuildData) GetUser(userId string) (*UserData, error) {
	return GetUser(self.GuildId, userId)
}

func createUser(userId, guildId string, lastAcitivity time.Time) UserData {
	var userData UserData
	userData.GuildId = guildId
	userData.UserId = userId
	userData.LastActivity = lastAcitivity
	userData.Immune = false
	return userData
}

func CreateUser(userId, guildId string, lastMessage time.Time) error {
	data := createUser(userId, guildId, lastMessage)
	_, err := MongoClient.UsersCollection().InsertOne(context.Background(), data)
	if err != nil {
		return err
	}
	return nil
}

func DeleteUser(guildId, userId string) error {
	_, err := MongoClient.UsersCollection().DeleteOne(context.Background(), bson.D{{"guildId", guildId}, {"userId", userId}})
	if err != nil {
		return err
	}
	return nil
}

func GetUser(guildId, userId string) (*UserData, error) {
	var userData *UserData = new(UserData)

	// Try finding the user
	result := MongoClient.UsersCollection().FindOne(context.Background(), bson.D{{"guildId", guildId}, {"userId", userId}})
	err := result.Err()
	if err != nil {
		return nil, err
	}

	// Decode and return
	err = result.Decode(userData)
	if err != nil {
		return nil, err
	}

	// Return decoded data
	return userData, nil
}

type UserData struct {
	GuildId      string    `bson:"guildId"`
	UserId       string    `bson:"userId"`
	LastActivity time.Time `bson:"lastActivity`
	Immune       bool      `bson:"immune"`
}

func (self *UserData) UpdateActivity(time time.Time) error {
	filter := bson.D{{"guildId", self.GuildId}, {"userId", self.UserId}}

	self.LastActivity = time

	// Update database
	_, err := MongoClient.UsersCollection().UpdateOne(context.Background(), filter, *self)
	if err != nil {
		return err
	}
	return nil
}

func (self *UserData) UpdateImmunity(immunity bool) error {
	filter := bson.D{{"guildId", self.GuildId}, {"userId", self.UserId}}

	self.Immune = immunity

	// Update database
	_, err := MongoClient.UsersCollection().UpdateOne(context.Background(), filter, *self)
	if err != nil {
		return err
	}
	return nil
}
