package models

// Command ... used in future, have to generalize input before this happens
type Command struct {
	User    string
	Command string
	Matchup string
	Type    string
	Name    string
}

// Build ... build object
type Build struct {
	ID          int    `storm:"id,increment"`
	SubmittedBy string `storm:"index"`
	BuildName   string `storm:"index"`
	Matchup     string `storm:"index"`
	Type        string `storm:"index"`
	Build       string
}

// WhiteListUser ... used to be an object of a white list
type WhiteListUser struct {
	ID            int    `storm:"id,increment"`
	UserName      string `storm:"index"`
	WhiteListedBy string `storm:"index"`
}
