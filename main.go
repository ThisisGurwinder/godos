package main

//TODO: test
import (
	"flag"
	"fmt"
	"log"
	"regexp"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/skratchdot/open-golang/open"

	"github.com/izqui/functional"
)

var (
	token = flag.String("token", "", "Github setup token")
	reset = flag.Bool("reset", false, "Reset Github token")
)

func init() {

	flag.Parse()
}

func main() {

	root, err := GitDirectoryRoot()

	if err != nil {

		fmt.Println("You must use todos inside a git repository")

	} else {

		if len(flag.Args()) < 1 {
			showHelp()
		} else {

			mode := flag.Args()[0]
			switch mode {
			case "setup":

				f := OpenConfiguration(HOME_DIRECTORY_CONFIG)
				defer f.File.Close()

				// Check config file for github access token
				if *token != "" {

					f.Config.GithubToken = *token

				} else if f.Config.GithubToken == "" || *reset {

					fmt.Println("Paste Github access token:")
					open.Run(TOKEN_URL)
					var scanToken string
					fmt.Scanln(&scanToken)
					f.Config.GithubToken = scanToken //TODO: Check if token is valid [Issue: https://github.com/izqui/todos/issues/5]
				}

				f.WriteConfiguration()

				setupHook(root + "/.git/hooks/pre-commit")

			case "work":

				f := OpenConfiguration(HOME_DIRECTORY_CONFIG)
				defer f.File.Close()

				if f.Config.GithubToken == "" {

					fmt.Println("Missing Github token. Set it in ~/.todos/conf.json")

				} else {

					o := &oauth.Transport{
						Token: &oauth.Token{AccessToken: f.Config.GithubToken},
					}

					owner, repo, err := GetOwnerRepoFromRepository()
					logOnError(err)

					fmt.Println("Scanning changed files on", owner, repo)

					client := github.NewClient(o.Client())

					diff, _ := GitDiffFiles()
					diff = functional.Map(func(s string) string { return root + "/" + s }, diff).([]string)

					existingRegex, err := regexp.Compile(ISSUE_URL_REGEX)
					logOnError(err)
					todoRegex, err := regexp.Compile(TODO_REGEX)
					logOnError(err)

					for _, file := range diff {
						lines, err := ReadFileLines(file)
						logOnError(err)

						changes := false

						for i, line := range lines {

							//TODO: Make async [Issue: https://github.com/izqui/todos/issues/6]

							ex := existingRegex.FindString(line)
							todo := todoRegex.FindString(line)

							if ex == "" && todo != "" {

								issue, _, err := client.Issues.Create(owner, repo, &github.IssueRequest{Title: &todo})
								logOnError(err)

								lines[i] = fmt.Sprintf("%s [Issue: %s]", line, *issue.HTMLURL)
								changes = true
							}
						}

						if changes {
							logOnError(WriteFileLines(file, lines, false))
						}
					}
				}

			default:
				showHelp()
			}
		}
	}
}

func setupHook(path string) {

	bash := "#!/bin/bash"
	script := "todos work"

	lns, err := ReadFileLines(path)
	logOnError(err)

	if len(lns) == 0 {
		lns = append(lns, bash)
	}

	//Filter existing script line
	lns = functional.Filter(func(a string) bool { return a != script }, lns).([]string)
	lns = append(lns, script)

	logOnError(WriteFileLines(path, lns, true))

}

func showHelp() {

	fmt.Println("Unknown command") //TODO: Write help [Issue: https://github.com/izqui/todos/issues/7]
}
func logOnError(err error) {

	if err != nil {
		log.Fatal(err)
	}
}
