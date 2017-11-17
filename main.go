package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/asdine/storm/q"

	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
)

type Configuration struct {
	Email    string
	Password string
}

type Build struct {
	ID          int    `storm:"id,increment"`
	SubmittedBy string `storm:"index"`
	BuildName   string `storm:"index"`
	Matchup     string `storm:"index"`
	Type        string `storm:"index"`
	Build       string
}

type WhiteListUser struct {
	ID            int    `storm:"id,increment"`
	UserName      string `storm:"index"`
	WhiteListedBy string `storm:"index"`
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
var numRequestDaily = 0

var displayFormattedHelper = "\n\nCurrent Commands:  \nHelp: !buildbot [help]\nStatus: !buildbot [status]\nInfo: !buildbot [info]\nGet: !buildbot [get] [matchup] [type] [name] \nGet(any): !buildbot [get] [any]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]\nRandom: !buildbot [random] [matchup]\nMod: !buildbot [mod]\n\nExample: !buildbot save 12-Pool zvz cheese 12 Pool,13 Overlord,Spam Lings and A-Move,???,Collect tears"
var modFormattedHelper="\n\nCurrent Moderator Commands: \nWhitelist: !buildbot [whitelist] [DiscordUserName#0123]\nGet Build Id: !buildbot [id] [build name]\nDelete: !buildbot [delete] [build id]"
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
	fmt.Println(configuration.Email)

	disc, err := discordgo.New(configuration.Email, configuration.Password)
	if err != nil {
		fmt.Println("Error creating discord session", err)
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
	<-sc
}

// RecieveMessage ... receives messages sent to the bot via any channel it is in
func RecieveMessage(sess *discordgo.Session, mess *discordgo.MessageCreate) {
	db, err := storm.Open("my.db")
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()

	fmt.Println(mess.Author.String())
	slice := strings.Split(mess.Message.Content, " ")
	if slice[0] == "!buildbot" {
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
		if slice[1] == "get" {
			if len(slice) == 5 {
				sess.ChannelMessageSend(mess.ChannelID, getBuild(db, slice[2], slice[3], slice[4]))
			} else if len(slice) == 4 {
				sess.ChannelMessageSend(mess.ChannelID, getBuild(db, slice[2], slice[3], ""))
			} else if len(slice) == 3 {
				sess.ChannelMessageSend(mess.ChannelID, getBuild(db, slice[2], "", ""))
			} else {
				sess.ChannelMessageSend(mess.ChannelID, "An error has occured.")
				return
			}
		} else if slice[1] == "save" && len(slice) > 5 {
			if err != nil {
				sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
				return
			}
			saveBuild(db, slice[2], slice[3], slice[4], slice[5:len(slice)], mess.Author.String())
			sess.ChannelMessageSend(mess.ChannelID, "Saved Build.")
		} else if slice[1] == "help" {
			sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
		} else if slice[1] == "status" {
			s, r := getAllBuildCount(db)
			sess.ChannelMessageSend(mess.ChannelID, "Current Status: Online\nBuild Count: "+s+"\nUnique User Count: "+r)
		} else if slice[1] == "info" {
			sess.ChannelMessageSend(mess.ChannelID, "Created by: Enlisted Reb\nCommunity driven and free to use! Please follow my progress at https://github.com/TrystanHumann \nIf you have any questions or ideas you want to share, add me on discord! :D EnlistedReb#8778")
		} else if slice[1] == "random" && len(slice) > 2 {
			sess.ChannelMessageSend(mess.ChannelID, getRand(db, slice[2]))
		} else if slice[1] == "whitelist" && len(slice) > 2 && mess.Author.String() == "EnlistedReb#8778" {
			AddToWhiteList(db, slice[2], mess.Author.String())
			sess.ChannelMessageSend(mess.ChannelID, "User: "+slice[2]+" added to white list by "+mess.Author.String()+". ")
		} else if slice[1] == "backup" {
			if !CheckWhiteList(db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			f, err := os.OpenFile("my.db", os.O_RDONLY, 0755)
			if handleErr(err) {
				return
			}
			sess.ChannelFileSend(mess.ChannelID, "my.db", f)
			f.Close()
		} else if slice[1] == "id" && len(slice) > 2 {
			if !CheckWhiteList(db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, getOneBuildId(db, slice[2]))
		} else if slice[1] == "delete" && len(slice) > 2 {
			if !CheckWhiteList(db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, deleteBuild(db, slice[2]))
		} else if slice[1] == "mod" && len(slice) > 1 {
			if !CheckWhiteList(db, mess.Author.String()) {
				sess.ChannelMessageSend(mess.ChannelID, "You don't have access to this command.")
				return
			}
			sess.ChannelMessageSend(mess.ChannelID, modFormattedHelper)
		} else {
			sess.ChannelMessageSend(mess.ChannelID, displayFormattedHelper)
		}
	}
	db.Close()
}

//handleErr ... error handler, returns a true/false
func handleErr(err error) bool {
	if err != nil {
		fmt.Println(err)
		return true
	}
	return false
}

//saveBuild ... save the build for user - they can copy and paste
func saveBuild(db *storm.DB, buildName string, match string, buildType string, buildOrder []string, submittedBy string) error {
	var i string
	match = strings.ToLower(match)
	buildType = strings.ToLower(buildType)
	for _, element := range buildOrder {
		i += element + " "
	}

	build := Build{
		SubmittedBy: submittedBy,
		BuildName:   buildName,
		Matchup:     match,
		Type:        buildType,
		Build:       i,
	}

	err := db.Save(&build)
	if handleErr(err) {
		return err
	}

	fmt.Println("Saved build Order")
	return err
}

//getRand ... gets a random build in a matchup from the database and returns a formatted string for it.
func getRand(db *storm.DB, match string) string {
	var build []Build
	err := db.Find("Matchup", match, &build)
	if handleErr(err) {
		fmt.Println(err)
		return "Error finding build order. Try again please!"
	}
	randNum := rand.Intn(len(build))
	formattedBuild := strings.Replace(build[randNum].Build, ",", "\n", -1)
	return "Build Name: " + build[randNum].BuildName + "\n" + "Matchup: " + build[randNum].Matchup + "\n" + "Build Type: " + build[randNum].Type + "\nSubmitted By: " + build[randNum].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//getBuild ... get a build that the user requests and return it
func getBuild(db *storm.DB, match string, buildType string, buildName string) string {
	var matches []q.Matcher
	var build []Build
	fmt.Println(match, buildType, buildName)
	if match != "" {
		matches = append(matches, q.Eq("Matchup", match))
	}
	if buildType != "" {
		matches = append(matches, q.Eq("Type", buildType))
	}
	if buildName != "" {
		matches = append(matches, q.Eq("BuildName", buildName))
	}
	query := db.Select(matches...)
	err := query.Find(&build)
	if handleErr(err) {
		fmt.Println(err)
		output, errRet := getBuildOneItemSearch(db, match)
		if handleErr(errRet) {
			return "Error finding build order. Try again please!"
		}
		return output
	}
	//format builds that are seperated by commas into something that is readable at a glance (just first ones for now)
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	return "Build Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//getBuildOneItemSearch ... search for a build based on the first keyword
func getBuildOneItemSearch(db *storm.DB, search string) (string, error) {
	var build []Build
	err := db.Find("Matchup", search, &build)
	if !handleErr(err) {
		return formatBuild(build), err
	}
	err = db.Find("Type", search, &build)
	if !handleErr(err) {
		return formatBuild(build), err
	}
	err = db.Find("BuildName", search, &build)
	if !handleErr(err) {
		return formatBuild(build), err
	}
	return "", err
}

//getOneBuildId ... search for a build based on the first keyword and return string with id so user can delete the build
func getOneBuildId(db *storm.DB, search string) string {
	var build []Build
	err := db.Find("Matchup", search, &build)
	if !handleErr(err) {
		return formatBuildWithID(build)
	}
	err = db.Find("Type", search, &build)
	if !handleErr(err) {
		return formatBuildWithID(build)
	}
	err = db.Find("BuildName", search, &build)
	if !handleErr(err) {
		return formatBuildWithID(build)
	}
	return ""
}

//getBuildOneItemSearch ... search for a build based on the first keyword
func deleteBuild(db *storm.DB, searchId string) string {
	var build Build
	convNum, err := strconv.Atoi(searchId)
	if handleErr(err) {
		return "An error has occured, put an integer id." + err.Error()
	}
	err = db.One("ID", convNum, &build)
	if handleErr(err) {
		return "An error has occured. " + err.Error()
	}
	buildName := build.BuildName
	fmt.Println("ima delete it for real" + buildName)
	err = db.DeleteStruct(&build)
	if handleErr(err) {
		return "An error has occured." + err.Error()
	}
	return "Deleted Build Named: " + buildName
}

//formatBuild ... format a build so it doesn't look bad
func formatBuild(build []Build) string {
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	return "Build Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//formatBuild ... format a build so it doesn't look bad
func formatBuildWithID(build []Build) string {
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	idField := strconv.Itoa(build[0].ID)
	return "ID: " + idField + "\nBuild Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//getAllBuildCount ... get a build count and unique player count
func getAllBuildCount(db *storm.DB) (string, string) {
	var build []Build
	err := db.All(&build)
	if handleErr(err) {
		fmt.Println(err)
		return "", "Error finding build count. Try again please!"
	}
	s := strconv.Itoa(len(build))
	var dupeList []string
	for _, el := range build {
		dupeList = append(dupeList, el.SubmittedBy)
	}
	RemoveDuplicates(&dupeList)
	fmt.Println(dupeList)
	r := strconv.Itoa(len(dupeList))
	return s, r
}

//RemoveDuplicates ... get a string slice and remove all duplicates to show unique list only
func RemoveDuplicates(dupeSlice *[]string) {
	foundSlice := make(map[string]bool)
	j := 0
	for i, x := range *dupeSlice {
		if !foundSlice[x] {
			foundSlice[x] = true
			(*dupeSlice)[j] = (*dupeSlice)[i]
			j++
		}
	}
	*dupeSlice = (*dupeSlice)[:j]
}

//AddToWhiteList ... add a user to the white list, currently only able to if you're EnlistedReb#8778
func AddToWhiteList(db *storm.DB, userName string, whiteLister string) error {
	whiteListUser := WhiteListUser{
		UserName:      userName,
		WhiteListedBy: whiteLister,
	}
	err := db.Save(&whiteListUser)
	if handleErr(err) {
		return err
	}
	return err
}

//CheckWhiteList ... whitelist users for now to only delete random stuff, will be used to save eventually (maybe idk)
func CheckWhiteList(db *storm.DB, userName string) bool {
	var whiteList []WhiteListUser
	err := db.All(&whiteList)
	if handleErr(err) {
		return false
	}
	var dupeList []string
	for _, el := range whiteList {
		dupeList = append(dupeList, el.UserName)
	}
	RemoveDuplicates(&dupeList)
	//look for user in the unique list (didn't have to make it unique, but why not)
	for _, user := range dupeList {
		if user == userName {
			return true
		}
	}

	return false
}
