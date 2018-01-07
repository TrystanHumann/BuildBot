package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	utils "github.com/buildbot/utils"

	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
)

type Configuration struct {
	Username string
	Token    string
}

var (
	Command    string
	BuildName  string
	Match      string
	BuildType  string
	BuildOrder string
)
var limitPerDay = 3000
var messageMaxLength = 10000
var fileSize = 40000000
var numRequestDaily = 0

var displayFormattedHelper = "\n\nCurrent Commands:  \nHelp: @buildbot [help]\nStatus: @buildbot [status]\nInfo: @buildbot [info]\nGet: @buildbot [get] [matchup] [type] [name] \nGet(any): @buildbot [get] [any]\nSave: @buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]\nList: @buildbot [list] [matchup]\nRandom: @buildbot [random] [matchup]\nRock Paper Scissors: @buildbot [rock/paper/scissors]\nMod: @buildbot [mod]\n\nExample: @buildbot save 12-Pool zvz cheese 12 Pool,13 Overlord,Spam Lings and A-Move,???,Collect tears"
var modFormattedHelper = "\n\nCurrent Moderator Commands: \nWhitelist: @buildbot [whitelist] [DiscordUserName#0123]\nGet Build Id: @buildbot [id] [build name]\nDelete: @buildbot [delete] [build id]"
var db storm.DB

func main() {

	//Set up auto restart for daily limit requests (Don't hit server limit because I'm broke.)
	go (func() {
		c := time.Tick(24 * time.Hour)
		for now := range c {
			fmt.Println("Requests so far: ", numRequestDaily)
			numRequestDaily = 0
			fmt.Println("Daily limit reset", now)
		}
	})()

	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(configuration.Username)
	disc, err := discordgo.New(configuration.Token)
	if err != nil {
		fmt.Println("Error creating discord session", err)
		return
	}
	fmt.Printf("Your Authentication Token is:\n\n%s\n", disc.Token)

	// disc.Login(disc.Token)
	err = disc.Open()
	defer disc.Close()
	if utils.HandleErr(err) {
		return
	}

	db, err := storm.Open("my.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	disc.AddHandler(RecieveMessage)

	fmt.Println("Click Ctrl+C to close program")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

// RecieveMessage ... receives messages sent to the bot via any channel it is in
func RecieveMessage(sess *discordgo.Session, mess *discordgo.MessageCreate) {

	slice := strings.Split(mess.Message.Content, " ")
	if len(mess.Mentions) <= 0 {
		return
	}

	if strings.ToLower(mess.Mentions[0].String()) == strings.ToLower(sess.State.User.String()) {
		numRequestDaily++ //increment request num to limit requests to server
		if numRequestDaily > limitPerDay {
			sess.ChannelMessageSend(mess.ChannelID, "Too many request for the server today. Make more tommorow.")
			return
		}
		if len(mess.Message.Content) > messageMaxLength {
			sess.ChannelMessageSend(mess.ChannelID, "Message too large."+displayFormattedHelper)
			return
		}
		if len(slice) <= 1 {
			sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
			return
		}
		if utils.IgnoreCase(slice[1], "get") {
			if len(slice) == 5 {
				sess.ChannelMessageSend(mess.ChannelID, utils.GetBuild(&db, slice[2], slice[3], slice[4]))
			} else if len(slice) == 4 {
				sess.ChannelMessageSend(mess.ChannelID, utils.GetBuild(&db, slice[2], slice[3], ""))
			} else if len(slice) == 3 {
				sess.ChannelMessageSend(mess.ChannelID, utils.GetBuild(&db, slice[2], "", ""))
			} else {
				sess.ChannelMessageSend(mess.ChannelID, "An error has occured.")
				return
			}
		} else if utils.IgnoreCase(slice[1], "save") && len(slice) > 5 {
			// if err != nil {
			// 	sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
			// 	return
			// }
			utils.SaveBuild(&db, slice[2], slice[3], slice[4], slice[5:len(slice)], mess.Author.String())
			sess.ChannelMessageSend(mess.ChannelID, "Saved Build.")
		} else if utils.IgnoreCase(slice[1], "help") {
			sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
		} else if utils.IgnoreCase(slice[1], "status") {
			s, r := utils.GetAllBuildCount(&db)
			sess.ChannelMessageSend(mess.ChannelID, "Current Status: Online\nBuild Count: "+s+"\nUnique User Count: "+r)
		} else if utils.IgnoreCase(slice[1], "info") {
			sess.ChannelMessageSend(mess.ChannelID, "Created by: Enlisted Reb\nCommunity driven and free to use! Please follow my progress at https://github.com/TrystanHumann \nIf you have any questions or ideas you want to share, add me on discord! :D EnlistedReb#8778")
		} else if utils.IgnoreCase(slice[1], "random") && len(slice) > 2 {
			sess.ChannelMessageSend(mess.ChannelID, utils.GetRand(&db, slice[2]))
		} else if utils.IgnoreCase(slice[1], "whitelist") && len(slice) > 2 && mess.Author.String() == "EnlistedReb#8778" {
			utils.AddToWhiteList(&db, slice[2], mess.Author.String())
			sess.ChannelMessageSend(mess.ChannelID, "User: "+slice[2]+" added to white list by "+mess.Author.String()+". ")
		} else if utils.IgnoreCase(slice[1], "backup") {
			if !utils.CheckWhiteList(&db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			f, err := os.OpenFile("my.db", os.O_RDONLY, 0755)
			if utils.HandleErr(err) {
				return
			}
			sess.ChannelFileSend(mess.ChannelID, "my.db", f)
			f.Close()
		} else if utils.IgnoreCase(slice[1], "id") && len(slice) > 2 {
			if !utils.CheckWhiteList(&db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, utils.GetOneBuildId(&db, slice[2]))
		} else if utils.IgnoreCase(slice[1], "delete") && len(slice) > 2 {
			if !utils.CheckWhiteList(&db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, utils.DeleteBuild(&db, slice[2]))
		} else if utils.IgnoreCase(slice[1], "mod") && len(slice) > 1 {
			if !utils.CheckWhiteList(&db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, modFormattedHelper)
		} else if utils.IgnoreCase(slice[1], "list") {
			s, err := utils.GetListOfBuilds(&db, slice[2])
			if err != nil {
				sess.ChannelMessageSend(mess.ChannelID, "Couldn't find any builds for that matchup. Add some and try again!")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, s)
		} else if utils.IgnoreCase(slice[1], "asciiDEVELOPMENT") {
			if len(mess.Attachments) <= 0 {
				sess.ChannelMessageSend(mess.ChannelID, "Try adding an attachment of your favorite picture.")
				return
			}

			if mess.Attachments[0].Size > fileSize {
				sess.ChannelMessageSend(mess.ChannelID, "File size too large. Limited at 5MB.")
				return
			}

			if err := utils.DownloadUrl(mess.Attachments[0].URL, mess.Attachments[0].Filename); utils.HandleErr(err) {
				sess.ChannelMessageSend(mess.ChannelID, "File size too large. Limited at 5MB.")
				fmt.Println(err)
				return
			}
			fmt.Println(mess.Attachments[0].Size)
			fmt.Println("Heres your stuff", mess.Attachments[0].Filename)
			// sess.ChannelMessageSend(mess.ChannelID, s)
		} else if utils.IgnoreCase(slice[1], "rock") || utils.IgnoreCase(slice[1], "paper") || utils.IgnoreCase(slice[1], "scissors") {
			botRockPaperScissorChoice := utils.RockPaperScissorsGenerator()
			botChoiceOutput, resultOutput := utils.DecideRockPaperScissorWinner(botRockPaperScissorChoice, slice[1], mess.Author.Username)
			sess.ChannelMessageSend(mess.ChannelID, botChoiceOutput+" "+resultOutput)
		} else {
			sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
		}
	}
	// db.Close()
}
