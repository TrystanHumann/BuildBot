package main

import (
	"os/signal"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"encoding/json"
	"github.com/asdine/storm"
	"syscall"
	"strings"
	"math/rand"
	"strconv"
)

type Configuration struct {
	Email string
	Password string
}

type Build struct {
  ID int `storm:"id,increment"`
  SubmittedBy string `storm:"index"`
  BuildName string `storm:"index"`
  Matchup string `storm:"index"`
  Type string `storm:"index"`
  Build string
}

var (
	Command string
	BuildName string
	Match string
	BuildType string
	BuildOrder string
)

func main() {

	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(configuration.Email)

	disc, err := discordgo.New(configuration.Email, configuration.Password)
	if err != nil {
		fmt.Println("error creating discord session", err)
		return
	}
	fmt.Printf("Your Authentication Token is:\n\n%s\n", disc.Token)
	err = disc.Open()
	defer disc.Close()
	if handleErr(err) {
		return
	}

	disc.AddHandler(RecieveMessage)

	fmt.Println("Click Ctrl+C to close program")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<- sc
}

func RecieveMessage(sess *discordgo.Session, mess *discordgo.MessageCreate) {
	db, err := storm.Open("my.db")
	if err != nil{
		fmt.Println(err)
	}
	defer db.Close()
	slice := strings.Split(mess.Message.Content, " ")
	fmt.Println(mess.Author)
	fmt.Println(len(slice))
	if slice[0] == "!buildbot" {
		if len(slice) <= 1 {
			sess.ChannelMessageSend(mess.ChannelID, "\n\nUse the following format:  \nHelp: !buildbot [help]\nStatus: !buildbot [status]\nGet: !buildbot [get] [matchup] [type]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]")
			return
		}
		if slice[1] == "get" && len(slice) > 3 {
			sess.ChannelMessageSend(mess.ChannelID, getBuild(db, slice[2], slice[3]))
		} else if slice[1] == "save" && len(slice) > 5 {
			if err != nil {
				sess.ChannelMessageSend(mess.ChannelID, "\n\nUse the following format:  \nHelp: !buildbot [help]\nStatus: !buildbot [status]\nGet: !buildbot [get] [matchup] [type]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]")
				return
			}
			saveBuild(db, slice[2], slice[3], slice[4], slice[5:len(slice)], mess.Author.String())
			sess.ChannelMessageSend(mess.ChannelID, "Saved Build.")
		} else if slice[1] == "help" {
			sess.ChannelMessageSend(mess.ChannelID, "Current Commands:\n\nHelp: !buildbot [help]\nStatus: !buildbot [status]\nGet: !buildbot [get] [matchup] [type]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]")
		} else if slice[1] == "status" {
			sess.ChannelMessageSend(mess.ChannelID, "Current Status: Online\n\nBuild count: " + getAllBuildCount(db) )
		} else if slice[1] == "info" {
			sess.ChannelMessageSend(mess.ChannelID, "Created by: Enlisted Reb\nCommunity driven and free to use! Please follow my progress at https://github.com/TrystanHumann \nIf you have any questions or ideas you want to share, add me on discord! :D EnlistedReb#8778")
		} else {	
			sess.ChannelMessageSend(mess.ChannelID, "Current Commands:\n\nHelp: !buildbot [help]\nStatus: !buildbot [status]\nGet: !buildbot [get] [matchup] [type]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]")
		}
	}
	db.Close()
}

func handleErr(err error) bool {
	if err != nil {
		fmt.Println(err)
		return true
	}
	return false
}

func saveBuild(db *storm.DB, buildName string, match string, buildType string, buildOrder[] string, submittedBy string) error {
	var i string
	fmt.Println(buildOrder)
	for _, element := range buildOrder {
		i += element + " "
	}

	build := Build{
		SubmittedBy: submittedBy,
		BuildName: buildName,
		Matchup: match,
		Type: buildType,
		Build: i,
	}

	err := db.Save(&build)
	if handleErr(err) {
		return err
	}
			

	fmt.Println("Saved build Order")
	return err
}

func getBuild(db *storm.DB, match string, buildType string) string {
	var build []Build
	err := db.Find("Matchup", match, &build)
	if handleErr(err) {
		fmt.Println(err)
		return "Error finding build order. Try again please!"
	}
	randNum := rand.Intn(len(build))
	formattedBuild := strings.Replace(build[randNum].Build, ",", "\n", -1)
	return "Build Name: " + build[randNum].BuildName + "\n" + "Matchup: " + build[randNum].Matchup+ "\n" + "Build Type: " + build[randNum].Type + "\nSubmitted By: "+ build[randNum].SubmittedBy+ "\n" + "---Build--- \n" + formattedBuild
}

func getAllBuildCount(db *storm.DB) string {
	var build []Build
	err := db.All(&build)
	if handleErr(err) {
		fmt.Println(err)
		return "Error finding build count. Try again please!"
	}
	s := strconv.Itoa(len(build))
	return s
}