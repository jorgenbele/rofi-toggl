package toggl

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/imroc/req"
)

const baseURL string = "https://api.track.toggl.com/api/v8"

func basicAuthWithToken(token string) string {
	auth := token + ":" + "api_token"
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func authHeader(token string) req.Header {
	return req.Header{"Authorization": fmt.Sprintf("Basic %s", basicAuthWithToken(token))}
}

type Userdata struct {
	ID         int `json:"id"`
	ApiToken   int `json:"api_token"`
	DefaultWID int `json:"default_wid"`
}

type Invitation struct {
	// ??
}

type Workspace struct {
	ID                          int       `json:"id"`
	Name                        string    `json:"name"`
	Profile                     int       `json:"profile"`
	Premium                     bool      `json:"premium"`
	Admin                       bool      `json:"admin"`
	DefaultHourlyRate           int       `json:"default_hourly_rate"`
	DefaultCurRency             string    `json:"default_cur rency"`
	OnlyAdminsMayCreateProjects bool      `json:"only_admins_may_create_projects"`
	OnlyAdminsSeeBillableRates  bool      `json:"only_admins_see_billable_rates"`
	OnlyAdminsSeeTeamDashboard  bool      `json:"only_admins_see_team_dashboard"`
	ProjectsBillableByDefault   bool      `json:"projects_billable_by_default"`
	Rounding                    int       `json:"rounding"`
	RoundingMinutes             int       `json:"rounding_minutes"`
	APIToken                    string    `json:"api_token"`
	At                          time.Time `json:"at"`
	IcalEnabled                 bool      `json:"ical_enabled"`
}

type UserData struct {
	ID                     int         `json:"id"`
	APIToken               string      `json:"api_token"`
	DefaultWid             int         `json:"default_wid"`
	Email                  string      `json:"email"`
	Fullname               string      `json:"fullname"`
	JqueryTimeofdayFormat  string      `json:"jquery_timeofday_format"`
	JqueryDateFormat       string      `json:"jquery_date_format"`
	TimeofdayFormat        string      `json:"timeofday_format"`
	DateFormat             string      `json:"date_format"`
	StoreStartAndStopTime  bool        `json:"store_start_and_stop_time"`
	BeginningOfWeek        int         `json:"beginning_of_week"`
	Language               string      `json:"language"`
	ImageURL               string      `json:"image_url"`
	SidebarPiechart        bool        `json:"sidebar_piechart"`
	At                     time.Time   `json:"at"`
	CreatedAt              time.Time   `json:"created_at"`
	Retention              int         `json:"retention"`
	RecordTimeline         bool        `json:"record_timeline"`
	RenderTimeline         bool        `json:"render_timeline"`
	TimelineEnabled        bool        `json:"timeline_enabled"`
	TimelineExperiment     bool        `json:"timeline_experiment"`
	ShouldUpgrade          bool        `json:"should_upgrade"`
	Timezone               string      `json:"timezone"`
	OpenidEnabled          bool        `json:"openid_enabled"`
	SendProductEmails      bool        `json:"send_product_emails"`
	SendWeeklyReport       bool        `json:"send_weekly_report"`
	SendTimerNotifications bool        `json:"send_timer_notifications"`
	Invitation             Invitation  `json:"invitation"`
	Workspaces             []Workspace `json:"workspaces"`
	DurationFormat         string      `json:"duration_format"`
}

type meResponse struct {
	Since int      `json:"since"`
	Data  UserData `json:"data"`
}

func Me(token string) (UserData, error) {
	log.Println("ME")
	var resp meResponse
	header := authHeader(token)
	r, err := req.Get(fmt.Sprintf("%s/me", baseURL), header)
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	log.Println("ME DONE")
	return resp.Data, nil
}

type ProjectResponse struct {
	Data Project `json:"data"`
}

type Project struct {
	ID        int       `json:"id"`
	Wid       int       `json:"wid"`
	Cid       int       `json:"cid"`
	Name      string    `json:"name"`
	Billable  bool      `json:"billable"`
	IsPrivate bool      `json:"is_private"`
	Active    bool      `json:"active"`
	At        time.Time `json:"at"`
	Template  bool      `json:"template"`
	Color     string    `json:"color"`
}

func (p *Project) String() string {
	return fmt.Sprintf("%s", p.Name)
}

type NewProjectRequestType struct {
	Project NewProjectType `json:"project"`
}
type NewProjectType struct {
	Name       string `json:"name"`
	WID        int    `json:"wid"`
	TemplateID int    `json:"template_id"`
	IsPrivate  bool   `json:"is_private"`
	Cid        int    `json:"cid"`
}

func CreateNewProjectOnWorkspace(token string, wid int, name string) (Project, error) {
	var resp Project
	header := authHeader(token)
	request := NewProjectRequestType{Project: NewProjectType{Name: name, WID: wid, TemplateID: 10237, IsPrivate: true}}
	r, err := req.Post(fmt.Sprintf("%s/projects", baseURL), header, req.BodyJSON(&request))
	if err != nil {
		return resp, err
	}
	r.ToJSON(&resp)
	return resp, nil
}

func GetProjectsForWorkspace(token string, id int) ([]Project, error) {
	var resp []Project
	header := authHeader(token)
	r, err := req.Get(fmt.Sprintf("%s/workspaces/%d/projects", baseURL, id), header)
	if err != nil {
		return resp, err
	}
	r.ToJSON(&resp)
	log.Printf("got projects for workspace: %d -- %v\n", id, resp)
	return resp, nil
}

func GetProjects(token string, workspaces []Workspace) []Project {
	results := make(chan []Project)
	for _, workspace := range workspaces {
		go func(token string, id int, out chan []Project) {
			projects, err := GetProjectsForWorkspace(token, id)
			if err != nil {
				out <- []Project{}
			} else {
				out <- projects
			}

		}(token, workspace.ID, results)
	}

	var out []Project
	for _, _ = range workspaces {
		result := <-results
		out = append(out, result...)
	}
	return out
}

type TimeEntry struct {
	ID          int        `json:"id"`
	Wid         int        `json:"wid"`
	PID         *int       `json:"pid"`
	Billable    bool       `json:"billable"`
	Start       time.Time  `json:"start"`
	Stop        *time.Time `json:"stop,omitempty"`
	Duration    int64      `json:"duration"`
	Description string     `json:"description"`
	Duronly     bool       `json:"duronly"`
	At          time.Time  `json:"at"`
	UID         int        `json:"uid"`
}

type NewTimeEntryType struct {
	Billable    bool   `json:"billable"`
	Description string `json:"description"`
	CreatedWith string `json:"created_with"`
	PID         *int   `json:"pid"`
}

func (t NewTimeEntryType) StartNow(token string) (TimeEntry, error) {
	return StartTimeEntry(token, t)
}

type newTimeEntryRequest struct {
	TimeEntry NewTimeEntryType `json:"time_entry"`
}

func NewTimeEntry(description string) NewTimeEntryType {
	return NewTimeEntryType{Description: description, CreatedWith: "curl"}
}

func NewTimeEntryFromExisting(existing TimeEntry) NewTimeEntryType {
	return NewTimeEntryType{Billable: existing.Billable, Description: existing.Description, CreatedWith: "curl"}
}

func (t TimeEntry) TimeDuration() time.Duration {
	if t.Duration < 0 {
		// -t.duration is time in seconds since epoch of start
		start := time.Unix(-t.Duration, 0)
		duration := time.Since(start)
		return duration
	}
	return time.Duration(t.Duration * 10e8) // seconds to nanoseconds
}

type timeEntryResponse struct {
	Data TimeEntry `json:"data"`
}

func (t TimeEntry) IsRunning() bool {
	return t.Duration < 0
}

func (t TimeEntry) StartNow(token string) (TimeEntry, error) {
	newT := NewTimeEntryFromExisting(t)
	if t.IsRunning() {
		return t, fmt.Errorf("cannot start already started timeentry: %d", t.ID)
	}
	return StartTimeEntry(token, newT)
}

func (t TimeEntry) StopNow(token string) (TimeEntry, error) {
	if !t.IsRunning() {
		return t, fmt.Errorf("cannot stop already stopped timeentry: %d", t.ID)
	}
	return StopTimeEntry(token, t.ID)
}

func (t TimeEntry) String() string {
	var runningStr string
	if t.IsRunning() {
		runningStr = "running"
	} else {
		runningStr = "not running"
	}
	return fmt.Sprintf("%s [%s]", t.Description, runningStr)
}

func (t TimeEntry) FullString(projects []Project) string {
	var runningStr string
	if t.IsRunning() {
		runningStr = "running"
	} else {
		runningStr = "not running"
	}

	if t.PID != nil {
		var projectName *string
		for _, project := range projects {
			if project.ID == *t.PID {
				projectName = &project.Name
			}
		}

		if projectName != nil {
			return fmt.Sprintf("%s [%s] @%s", t.Description, runningStr, *projectName)
		}
	}

	return fmt.Sprintf("%s [%s]", t.Description, runningStr)
}

func GetLatestTimeEntries(token string) ([]TimeEntry, error) {
	var resp []TimeEntry
	header := authHeader(token)
	r, err := req.Get(fmt.Sprintf("%s/time_entries", baseURL), header)
	if err != nil {
		return resp, err
	}
	r.ToJSON(&resp)
	return resp, nil
}

// Will only get the latest in the last 9 days
func GetLastTimeEntry(token string) (TimeEntry, error) {
	var mostRecent TimeEntry

	entries, err := GetLatestTimeEntries(token)
	if err != nil {
		return mostRecent, err
	}

	if len(entries) == 0 {
		return mostRecent, fmt.Errorf("no entry the last 9 days")
	}

	for _, e := range entries {
		if e.IsRunning() {
			return e, nil
		}
		// Comparing on start time might not be the best comparison...
		if time.Since(e.Start) < time.Since(mostRecent.Start) {
			mostRecent = e
		}
	}

	return mostRecent, nil
}

func GetTimeEntry(token string, id int) (TimeEntry, error) {
	var resp timeEntryResponse
	// https://api.track.toggl.com/api/v8/time_entries/current
	header := authHeader(token)
	r, err := req.Get(fmt.Sprintf("%s/time_entries/%d", baseURL, id), header)
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	return resp.Data, nil
}

func CurrentTimeEntry(token string) (TimeEntry, error) {
	var resp timeEntryResponse
	// https://api.track.toggl.com/api/v8/time_entries/current
	header := authHeader(token)
	r, err := req.Get(fmt.Sprintf("%s/time_entries/current", baseURL), header)
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	return resp.Data, nil
}

func StopTimeEntry(token string, id int) (TimeEntry, error) {
	var resp timeEntryResponse
	header := authHeader(token)
	r, err := req.Put(fmt.Sprintf("%s/time_entries/%d/stop", baseURL, id), header)
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	return resp.Data, nil
}

func StartTimeEntry(token string, entry NewTimeEntryType) (TimeEntry, error) {
	request := newTimeEntryRequest{TimeEntry: entry}

	var resp timeEntryResponse
	header := authHeader(token)
	r, err := req.Post(fmt.Sprintf("%s/time_entries/start", baseURL), header, req.BodyJSON(&request))
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	return resp.Data, nil
}

func UpdateTimeEntryDescription(token string, entry TimeEntry) (TimeEntry, error) {
	var resp timeEntryResponse
	header := authHeader(token)
	r, err := req.Put(fmt.Sprintf("%s/time_entries/current", baseURL), header, req.BodyJSON(&entry))
	if err != nil {
		return resp.Data, err
	}
	r.ToJSON(&resp)
	return resp.Data, nil
}
