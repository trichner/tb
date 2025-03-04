package jiracli

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/trichner/tb/pkg/jira"
	"github.com/trichner/tb/pkg/jira/credentials"
)

var cli struct {
	CreateUser struct {
		Email  string `help:"Email of the new user." required:""`
		Groups string `help:"Groups, comma separated." required:""`
	} `cmd:"" help:"Create a new user."`
	Groups struct {
		User   string `help:"accountId of the user to assign the groups to" required:""`
		Add    string `help:"Groups to add, comma separated." required:""`
		Remove string `help:"Groups to remove, comma separated." required:""`
	} `cmd:"" help:"Add groups to an existing user."`
	Issues struct {
		Query string `help:"search by query" required:""`
	} `cmd:"" help:"Find or update issues"`
}

func Exec(ctx context.Context, args []string) {
	k, err := kong.New(&cli)
	if err != nil {
		log.Fatal(err)
	}
	kctx, err := k.Parse(args[1:])
	if err != nil {
		log.Fatal(err)
	}

	switch kctx.Command() {
	case "create-user":

		email := cli.CreateUser.Email
		groups := strings.Split(cli.CreateUser.Groups, ",")
		name := deriveNameFromEmail(email)
		createUser(name, email, groups)
	case "issues":
		queryIssues(cli.Issues.Query)
	default:
		panic(kctx.Command())
	}
}

func queryIssues(query string) {
	clientCredentials, err := credentials.FindCredentials()
	if err != nil {
		log.Fatal(err)
	}

	service, err := jira.NewJiraService(clientCredentials.Baseurl, clientCredentials.Username, clientCredentials.Token)
	if err != nil {
		log.Fatal(err)
	}

	issues, err := service.SearchByQuery(query)
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(issues)
	if err != nil {
		log.Fatal(err)
	}
}

func createUser(name, email string, groups []string) {
	clientCredentials, err := credentials.FindCredentials()
	if err != nil {
		log.Fatalf("failed to read credentials: %s", err)
	}

	service, err := jira.NewJiraService(clientCredentials.Baseurl, clientCredentials.Username, clientCredentials.Token)
	if err != nil {
		log.Fatalf("failed to create service: %s", err)
	}

	s, err := service.CreateUser(&jira.CreateUser{
		Name:   name,
		Email:  email,
		Groups: groups,
	})
	if err != nil {
		log.Fatalf("failed to create user: %s", err)
	}
	fmt.Println(s)
}

func deriveNameFromEmail(email string) string {
	splits := strings.SplitN(email, "@", 2)
	if len(splits) != 2 {
		return email
	}

	nameParts := strings.SplitN(splits[0], ".", 2)
	if len(nameParts) != 2 {
		return email
	}
	return strings.ToTitle(nameParts[0]) + " " + strings.ToTitle(nameParts[1])
}
