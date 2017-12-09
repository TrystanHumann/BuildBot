package utils

import (
	"../models"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
)

//HandleErr ... error handler, returns a true/false
func HandleErr(err error) bool {
	if err != nil {
		fmt.Println("here I am")
		fmt.Println(err)
		return true
	}
	return false
}

//saveBuild ... save the build for user - they can copy and paste
func SaveBuild(db *storm.DB, buildName string, match string, buildType string, buildOrder []string, submittedBy string) error {
	var i string
	match = strings.ToLower(match)
	buildType = strings.ToLower(buildType)
	for _, element := range buildOrder {
		i += element + " "
	}

	build := models.Build{
		SubmittedBy: submittedBy,
		BuildName:   buildName,
		Matchup:     match,
		Type:        buildType,
		Build:       i,
	}

	err := db.Save(&build)
	if HandleErr(err) {
		return err
	}

	fmt.Println("Saved build Order")
	return err
}

//getRand ... gets a random build in a matchup from the database and returns a formatted string for it.
func GetRand(db *storm.DB, match string) string {
	var build []models.Build
	err := db.Find("Matchup", match, &build)
	if HandleErr(err) {
		fmt.Println(err)
		return "Error finding build order. Try again please!"
	}
	randNum := rand.Intn(len(build))
	formattedBuild := strings.Replace(build[randNum].Build, ",", "\n", -1)
	return "Build Name: " + build[randNum].BuildName + "\n" + "Matchup: " + build[randNum].Matchup + "\n" + "Build Type: " + build[randNum].Type + "\nSubmitted By: " + build[randNum].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//getBuild ... get a build that the user requests and return it
func GetBuild(db *storm.DB, match string, buildType string, buildName string) string {
	var matches []q.Matcher
	var build []models.Build
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
	if HandleErr(err) {
		fmt.Println(err)
		output, errRet := GetBuildOneItemSearch(db, match)
		if HandleErr(errRet) {
			return "Error finding build order. Try again please!"
		}
		return output
	}
	//format builds that are seperated by commas into something that is readable at a glance (just first ones for now)
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	return "Build Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//getBuildOneItemSearch ... search for a build based on the first keyword
func GetBuildOneItemSearch(db *storm.DB, search string) (string, error) {
	var build []models.Build
	err := db.Find("Matchup", search, &build)
	if !HandleErr(err) {
		return formatBuild(build), err
	}
	err = db.Find("Type", search, &build)
	if !HandleErr(err) {
		return formatBuild(build), err
	}
	err = db.Find("BuildName", search, &build)
	if !HandleErr(err) {
		return formatBuild(build), err
	}
	return "", err
}

//getOneBuildId ... search for a build based on the first keyword and return string with id so user can delete the build
func GetOneBuildId(db *storm.DB, search string) string {
	var build []models.Build
	err := db.Find("Matchup", search, &build)
	if !HandleErr(err) {
		return formatBuildWithID(build)
	}
	err = db.Find("Type", search, &build)
	if !HandleErr(err) {
		return formatBuildWithID(build)
	}
	err = db.Find("BuildName", search, &build)
	if !HandleErr(err) {
		return formatBuildWithID(build)
	}
	return ""
}

//getBuildOneItemSearch ... search for a build based on the first keyword
func DeleteBuild(db *storm.DB, searchId string) string {
	var build models.Build
	convNum, err := strconv.Atoi(searchId)
	if HandleErr(err) {
		return "An error has occured, put an integer id." + err.Error()
	}
	err = db.One("ID", convNum, &build)
	if HandleErr(err) {
		return "An error has occured. " + err.Error()
	}
	buildName := build.BuildName
	err = db.DeleteStruct(&build)
	if HandleErr(err) {
		return "An error has occured." + err.Error()
	}
	return "Deleted Build Named: " + buildName
}

//getAllBuildCount ... get a build count and unique player count
func GetAllBuildCount(db *storm.DB) (string, string) {
	var build []models.Build
	err := db.All(&build)
	if HandleErr(err) {
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

//getAllBuildCount ... get a build count and unique player count
func GetListOfBuilds(db *storm.DB, search string) (string, error) {
	var build []models.Build
	err := db.Find("Matchup", search, &build)
	if !HandleErr(err) {
		return formatBuildList(build), err
	}
	return "No builds found for this matchup", err
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
	whiteListUser := models.WhiteListUser{
		UserName:      userName,
		WhiteListedBy: whiteLister,
	}
	err := db.Save(&whiteListUser)
	if HandleErr(err) {
		return err
	}
	return err
}

//CheckWhiteList ... whitelist users for now to only delete random stuff, will be used to save eventually (maybe idk)
func CheckWhiteList(db *storm.DB, userName string) bool {
	var whiteList []models.WhiteListUser
	err := db.All(&whiteList)
	if HandleErr(err) {
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

//formatBuild ... format a build so it doesn't look bad
func formatBuild(build []models.Build) string {
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	return "Build Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

//formatBuild ... format a build so it doesn't look bad
func formatBuildWithID(build []models.Build) string {
	formattedBuild := strings.Replace(build[0].Build, ",", "\n", -1)
	idField := strconv.Itoa(build[0].ID)
	return "ID: " + idField + "\nBuild Name: " + build[0].BuildName + "\n" + "Matchup: " + build[0].Matchup + "\n" + "Build Type: " + build[0].Type + "\nSubmitted By: " + build[0].SubmittedBy + "\n" + "---Build--- \n" + formattedBuild
}

func formatBuildList(build []models.Build) string {
	var formattedBuild string
	for _, b := range build {
		formattedBuild += b.BuildName + "\n"
	}
	return formattedBuild
}

func DownloadUrl(url string, fileName string) error {
	r, err := http.Get(url)

	out, err := os.Create(fileName)
	if HandleErr(err) {
		return err
	}
	defer out.Close()

	if HandleErr(err) {
		return err
	}

	defer r.Body.Close()
	file, err := io.Copy(out, r.Body)
	if HandleErr(err) {
		return err
	}
	fmt.Println(file)
	return err
}
