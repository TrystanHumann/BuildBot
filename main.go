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
)

type Configuration struct {
	Email string
	Password string
}

type Build struct {
  ID int `storm:"id,increment"`
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

	// if Command == "get" {
	// 	getBuild(db, Match, BuildType)
	// } else {
	// 	saveBuild(db, BuildName, Match, BuildType, BuildOrder)
	// }


	// build := Build{
	// 	Matchup: Match,
	// 	Type: BuildType,
	// 	Build: "Test",
	// }

	// err = db.Save(&build)
	// if handleErr(err) {
	// 	return
	// }



	// fmt.Println(returnBuild(Match, BuildType))
	// reader := bufio.NewReader(os.Stdin)
    // var input string
	// for input != ":close" {
    //     fmt.Print("Enter message or command: ")
    //     text, err := reader.ReadString('\n')
    //     input = strings.TrimSuffix(strings.Trim(text, ""), "\n") // assigning text to input
    //     if err != nil {
    //         fmt.Println(err)
    //     }
 
    //     if input != ":close" {
    //         _, err := disc.ChannelMessageSend("CHANNEL ID", input)
    //         if err != nil {
    //             fmt.Println(err)
    //         }
    //     }
        // } else {
        //  // closing
        //  fmt.Println("Hehe")
        //  break
        // }
    // }
}

func RecieveMessage(sess *discordgo.Session, mess *discordgo.MessageCreate) {
	// fmt.Println(mess)
	db, err := storm.Open("my.db")
	if err != nil{
		fmt.Println(err)
	}
	defer db.Close()
	slice := strings.Split(mess.Message.Content, " ")
	if slice[0] == "!buildbot" {
		if len(slice) <= 2 {
			sess.ChannelMessageSend(mess.ChannelID, "Error has occured.\n\nUse the following format:  \nGet: !buildbot [get] [matchup] [type]\nSave: !buildbot [save] [Name] [Matchup] [BuildType] [Build,Seperated,By,Commas]")
			return
		}
		if slice[1] == "get" && slice[2] != "" {
			sess.ChannelMessageSend(mess.ChannelID, getBuild(db, slice[2], slice[3]))
		} else if slice[1] == "save" && slice[2] != "" {
			saveBuild(db, slice[2], slice[3], slice[4], slice[5])
			sess.ChannelMessageSend(mess.ChannelID, "Saved Build.")
		}
		fmt.Println("Shitty message author: ", mess.Message.Author)
		fmt.Println("Shitty Message content: ", mess.Message.Content)
		// sess.ChannelMessageSend(mess.ChannelID,  slice[1])
		// saveBuild(db, slice[1], slice[2], slice[3], "hi")
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

func saveBuild(db *storm.DB, buildName string, match string, buildType string, buildOrder string) {
	
	build := Build{
		BuildName: buildName,
		Matchup: match,
		Type: buildType,
		Build: buildOrder,
	}

	err := db.Save(&build)
	if handleErr(err) {
		return
	}
	fmt.Println("Saved build Order")
}

func getBuild(db *storm.DB, match string, buildType string) string {
	var build []Build
	err := db.Find("Matchup", match, &build)
	if handleErr(err) {
		fmt.Println(err)
	}
	randNum := rand.Intn(len(build))
	return "Build Name: " + build[randNum].BuildName + "\n" + "Matchup: " + build[randNum].Matchup+ "\n" + "Build Type:" + build[randNum].Type + "\n" + "Build: " + build[randNum].Build
}