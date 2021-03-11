package rofitoggl

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/jorgenbele/rofi-toggl/toggl"
)

const (
	DefaultCommandTimeout time.Duration = 5 * time.Second
)

// https://github.com/tkancf/rofi-snippet/blob/master/main.go
func run(command string, args []string, r io.Reader) (error, string) {
	// var cmd *exec.Cmd
	// cmd = exec.Command("sh", "-c", command)
	// out, err := cmd.Output()
	// result := strings.TrimRight(string(out), "\n")
	// return err, result

	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = r
	out, err := cmd.Output()
	result := strings.TrimRight(string(out), "\n")
	return err, result
}

func runWithoutInput(command string, args []string) (error, string) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	result := strings.TrimRight(string(out), "\n")
	return err, result
}

type selectResult struct {
	value     string
	timeEntry *toggl.TimeEntry
}

func SelectFromRecent(projects []toggl.Project, latest []toggl.TimeEntry, token string) (selectResult, error) {
	recentCompare := func(i, j int) bool {
		if latest[i].IsRunning() {
			return true
		}
		if latest[i].Stop != nil && latest[j].Stop != nil {
			return (*latest[i].Stop).After(*latest[j].Stop)
		}
		return latest[i].Start.After(latest[j].Start)
	}
	sort.SliceStable(latest, recentCompare)

	list := make([]string, len(latest), len(latest))
	for i, e := range latest {
		list[i] = e.FullString(projects)
	}

	reader := strings.NewReader(strings.Join(list, "\n"))
	err, out := run("rofi", []string{"-dmenu"}, reader)
	if err != nil {
		log.Fatal(err)
	}

	selected := selectResult{value: out}

	for i, elem := range list {
		if elem == out {
			selected.timeEntry = &latest[i]
			break
		}
	}

	return selected, nil
}

type actionType = string

const (
	start       actionType = "Start"
	stopCurrent            = "Stop current"
	noAction               = "Cancel"
)

func SelectAction(current *toggl.TimeEntry, token string) (actionType, error) {

	actions := map[string]string{
		start:       start,
		stopCurrent: stopCurrent,
	}

	var actionStrings []string
	if current != nil {
		actionStrings = []string{
			fmt.Sprintf("Currently running: %s (%s)", current.Description, current.TimeDuration().Round(time.Second)),
			start,
			stopCurrent,
			noAction,
		}
	} else {
		actionStrings = []string{start, noAction}
	}
	alts := strings.Join(actionStrings, "\n")
	reader := strings.NewReader(alts)

	args := []string{"-dmenu"}
	if current != nil {
		args = append(args, []string{"-selected-row", "1"}...)
	}

	err, out := run("rofi", args, reader)
	if err != nil {
		log.Fatal(err)
	}

	if val, ok := actions[out]; ok {
		return val, nil
	}
	return noAction, fmt.Errorf("invalid action: %s", out)
}

type projectSelectResult struct {
	value   string // create new project ??
	project *toggl.Project
}

func SelectProject(projects []toggl.Project, token string) (projectSelectResult, error) {
	list := make([]string, len(projects), len(projects))
	for i, e := range projects {
		list[i] = e.String()
	}

	reader := strings.NewReader(strings.Join(list, "\n"))
	err, out := run("rofi", []string{"-dmenu"}, reader)
	if err != nil {
		log.Fatal(err)
	}

	selected := projectSelectResult{value: out}

	for i, elem := range list {
		if elem == out {
			selected.project = &projects[i]
			break
		}
	}
	return selected, nil
}

func SelectWorkspace(workspaces []toggl.Workspace) (toggl.Workspace, error) {
	list := make([]string, len(workspaces), len(workspaces))
	for i, e := range workspaces {
		list[i] = e.Name
	}

	reader := strings.NewReader(strings.Join(list, "\n"))
	err, out := run("rofi", []string{"-dmenu"}, reader)

	found := false
	var workspace toggl.Workspace
	if err != nil {
		return workspace, err
	}

	for i, elem := range list {
		if elem == out {
			workspace = workspaces[i]
			found = true
			break
		}
	}

	if !found {
		return workspace, fmt.Errorf("invalid workspace: %s", out)
	}
	return workspace, nil
}

// FIXME: this isn't working properly
func DisplayMessage(msg string) {
	err, _ := runWithoutInput("rofi", []string{"-e", msg})
	if err != nil {
		log.Fatal(err)
	}
}

func Run() {
	var token string
	if value, ok := os.LookupEnv("TOGGL_API_TOKEN"); ok {
		token = value
	} else {
		log.Fatal(fmt.Errorf("no api token provided, env TOGGL_API_TOKEN not set"))
	}

	// Fetch projects while user is selecting action
	projectsChan := make(chan []toggl.Project, 1)
	workspaceChan := make(chan []toggl.Workspace, 1)
	go func(token string, projectsOut chan []toggl.Project, workspacesOut chan []toggl.Workspace) {
		userData, err := toggl.Me(token)
		if err != nil {
			projectsOut <- []toggl.Project{}
			workspacesOut <- []toggl.Workspace{}
			return
		}

		workspacesOut <- userData.Workspaces
		projects := toggl.GetProjects(token, userData.Workspaces)
		projectsOut <- projects
	}(token, projectsChan, workspaceChan)

	// Fetch last entries as well
	latestChan := make(chan []toggl.TimeEntry, 1)
	go func(token string, out chan []toggl.TimeEntry) {
		latest, err := toggl.GetLatestTimeEntries(token)
		if err != nil {
			out <- []toggl.TimeEntry{}
		}
		out <- latest
	}(token, latestChan)

	var currentPtr *toggl.TimeEntry = nil
	if current, err := toggl.CurrentTimeEntry(token); err == nil {
		// Not the best test
		if current.IsRunning() {
			currentPtr = &current
		}
	}

	action, _ := SelectAction(currentPtr, token)

	switch action {
	case start:
		{
			var err error
			var started toggl.TimeEntry

			latest := <-latestChan
			projects := <-projectsChan
			selected, _ := SelectFromRecent(projects, latest, token)

			if selected.timeEntry != nil {
				started, err = selected.timeEntry.StartNow(token)
			} else {
				selectedProject, err := SelectProject(projects, token)

				newTimeEntry := toggl.NewTimeEntry(selected.value)

				if selectedProject.project != nil {
					newTimeEntry.PID = &selectedProject.project.ID
				} else if err == nil && selectedProject.value != "" {
					workspaces := <-workspaceChan
					workspace, err := SelectWorkspace(workspaces)
					if err != nil {
						DisplayMessage(fmt.Sprintf("Failed to create project: '%v'", err))
						break
					}

					project, err := toggl.CreateNewProjectOnWorkspace(token, workspace.ID, selectedProject.value)
					if err != nil {
						DisplayMessage(fmt.Sprintf("Failed to create project: '%v'", err))
						break
					}
					newTimeEntry.PID = &project.ID
				}

				started, err = newTimeEntry.StartNow(token)
			}

			if err != nil {
				DisplayMessage(fmt.Sprintf("Failed to start : %v", err))
			} else {
				DisplayMessage(fmt.Sprintf("Started: %s", started))
			}
		}

	case stopCurrent:
		if currentPtr == nil {
			DisplayMessage(fmt.Sprintf("No currently running time entry"))
			break
		}

		stopped, err := currentPtr.StopNow(token)
		if err != nil {
			DisplayMessage(fmt.Sprintf("Failed to stop : %v", err))
		} else {
			DisplayMessage(fmt.Sprintf("Stopped: %s", stopped))
		}

	case noAction:
	default:
	}

}
