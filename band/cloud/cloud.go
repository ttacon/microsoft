package cloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type Summaries struct {
	Summaries []Summary `json:"summaries"`
}

type Summary struct {
	ActiveHours           int                   `json:"activeHours"`
	UVExposure            string                `json:"uvExposure"`
	CaloriesBurnedSummary CaloriesBurnedSummary `json:"CaloriesBurnedSummary"`
	HeartRateSummary      HeartRateSummary      `json:"HeartRateSummary"`
	DistanceSummary       DistanceSummary       `json:"distanceSummary"`
	FloorsClimbed         int                   `json:"floorsClimbed"`
	StepsTaken            int                   `json:"stepsTaken"`
	Duration              string                `json:"duration"`
	Period                string                `json:"period"`
	IsTransitDay          bool                  `json:"isTransitDay"`
	ParentDay             *time.Time            `json:"parentDay"`
	EndTime               *time.Time            `json:"endTime"`
	StartTime             *time.Time            `json:"startTime"`
	UserID                string                `json:"userId"`
}

type CaloriesBurnedSummary struct {
	Period        string `json:"period"`
	TotalCalories int    `json:"totalCalories"`
}

type HeartRateSummary struct {
	Period           string `json:"period"`
	AverageHeartRate int    `json:"averageHeartRate"`
	PeakHeartRate    int    `json:"peakHeartRate"`
	LowestHeartRate  int    `json:"lowestHeartRate"`
}

type DistanceSummary struct {
	Period              string `json:"period"`
	TotalDistance       int    `json:"totalDistance"`
	TotalDistanceOnFoot int    `json:"totalDistanceOnFoot"`
	ActualDistance      int    `json:"actualDistance"`
	ElevationGain       int    `json:"elevationGain"`
	ElevationLoss       int    `json:"elevationLoss"`
	MaxElevation        int    `json:"maxElevation"`
	MinElevation        int    `json:"minElevation"`
	WaypointDistance    int    `json:"waypointDistance"`
	Speed               int    `json:"speed"`
	Pace                int    `json:"pace"`
	OverallPace         int    `json:"overallPace"`
}

const (
	BASE_URL   = "https://api.microsofthealth.net/v1/me"
	USER_AGENT = "go-microsoft-band-cloud-api:v0.0.1"
)

var (
	baseURL, _ = url.Parse(BASE_URL)
)

type Client struct {
	Client  *http.Client
	BaseUrl *url.URL
}

type tokenSource oauth2.Token

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return (*oauth2.Token)(t), nil
}

type ConfigSource struct {
	cfg *oauth2.Config
}

func NewConfigSource(cfg *oauth2.Config) *ConfigSource {
	return &ConfigSource{
		cfg: cfg,
	}
}

func (c *ConfigSource) NewClient(tok *oauth2.Token) *Client {
	// TODO(ttacon): allow the config to have deadlines/timeouts
	// (for the context)?
	return &Client{
		Client:  c.cfg.Client(context.Background(), tok),
		BaseUrl: baseURL,
	}
}

// NewRequest creates an *http.Request with the given method, url and
// request body (if one is passed).
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	// this method is based off
	// https://github.com/google/go-github/blob/master/github/github.go:
	// NewRequest as it's a very nice way of doing this
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// This is useful as this functionality works the same for the actual
	// BASE_URL and the download url (TODO(ttacon): insert download url)
	// this seems to be failing to work not RFC3986 (url resolution)
	//	resolvedUrl := c.BaseUrl.ResolveReference(parsedUrl)
	resolvedUrl, err := url.Parse(c.BaseUrl.String() + urlStr)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if body != nil {
		if err = json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, resolvedUrl.String(), buf)
	if err != nil {
		return nil, err
	}

	// TODO(ttacon): identify which headers we should add
	// e.g. "Accept", "Content-Type", "User-Agent", etc.
	req.Header.Add("User-Agent", USER_AGENT)
	return req, nil
}

// Do "makes" the request, and if there are no errors and resp is not nil,
// it attempts to unmarshal the  (json) response body into resp.
func (c *Client) Do(req *http.Request, respStr interface{}) (*http.Response, error) {
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, errors.New(fmt.Sprintf("http request failed, resp: %#v", resp))
	}

	// TODO(ttacon): maybe support passing in io.Writer as resp (downloads)?
	if respStr != nil {
		err = json.NewDecoder(resp.Body).Decode(respStr)
	}
	return resp, err
}

type Period string

const (
	HOURLY Period = "hourly"
	DAILY  Period = "daily"
)

func (c *Client) PeriodSummaries(period Period) (error, Summaries) {
	var summaries Summaries
	req, err := c.NewRequest("GET", fmt.Sprintf("/Summaries/%s", period), nil)
	if err != nil {
		return err, summaries
	}

	resp, err := c.Do(req, &summaries)
	if err != nil {
		return err, summaries
	}
	resp.Body.Close()

	return nil, summaries
}

func (c *Client) Profile() (error, Profile) {
	var profile Profile
	req, err := c.NewRequest("GET", "/Profile", nil)
	if err != nil {
		return err, profile
	}

	resp, err := c.Do(req, &profile)
	if err != nil {
		return err, profile
	}
	resp.Body.Close()

	return nil, profile
}

type Profile struct {
	FirstName       string     `json:"firstString"`
	MiddleName      string     `json:"middleName"`
	LastName        string     `json:"lastName"`
	Birthdate       *time.Time `json:"birthdate"`
	PostalCode      string     `json:"postalCode"`
	Gender          string     `json:"gender"`
	Height          int        `json:"height"`
	Weight          int        `json:"weight"`
	PreferredLocale string     `json:"preferredLocale"`
	LastUpdateTime  *time.Time `json:"lastUpdateTime"`
}

// Devices
type Device struct {
	ID                 string     `json:"id"`
	DisplayName        string     `json:"displayName"`
	LastSuccessfulSync *time.Time `json:"lastSuccessfulSync"`
	DeviceFamily       string     `json:"deviceFamily"`
	HardwareVersion    string     `json:"hardwareVersion"`
	SoftwareVersion    string     `json:"softwareVersion"`
	ModelName          string     `json:"modelName"`
	Manufacturer       string     `json:"manufacturer"`
	DeviceStatus       string     `json:"deviceStatus"`
	CreatedDate        *time.Time `json:"createdDate"`
}

type DeviceProfiles struct {
	Devices   []Device `json:"deviceProfiles"`
	ItemCount int      `json:"itemCount"`
}

func (c *Client) Devices() (DeviceProfiles, error) {
	var devices DeviceProfiles
	req, err := c.NewRequest("GET", "/Devices", nil)
	if err != nil {
		return devices, err
	}

	resp, err := c.Do(req, &devices)
	if err != nil {
		return devices, err
	}
	resp.Body.Close()

	return devices, nil
}

func (c *Client) Device(id string) (Device, error) {
	var device Device
	req, err := c.NewRequest("GET", "/Devices/"+id, nil)
	if err != nil {
		return device, err
	}

	resp, err := c.Do(req, &device)
	if err != nil {
		return device, err
	}
	resp.Body.Close()

	return device, nil
}

func (c *Client) Activities() (Activities, error) {
	var activities Activities
	req, err := c.NewRequest("GET", "/Activities", nil)
	if err != nil {
		return activities, err
	}

	resp, err := c.Do(req, &activities)
	if err != nil {
		return activities, err
	}
	resp.Body.Close()

	return activities, nil
}

func (c *Client) Activity(id string) (Activity, error) {
	var activity Activity
	req, err := c.NewRequest("GET", "/Activities/"+id, nil)
	if err != nil {
		return activity, err
	}

	resp, err := c.Do(req, &activity)
	if err != nil {
		return activity, err
	}
	resp.Body.Close()

	return activity, nil
}

type Activities struct {
	SleepActivities        []Activity `json:"sleepActivities"`
	RunActivities          []Activity `json:"runActivities"`
	GuidedWorkoutActivites []Activity `json:"guidedWorkoutActivities"`
	GolfActivities         []Activity `json:"golfActivities"`
	FreePlayActivities     []Activity `json:"freePlayActivities"`
	BikeActivities         []Activity `json:"bikeActivities"`

	ItemCount int    `json:"itemCount"`
	NextPage  string `json:"nextPage"`
}

type BikeActivity struct {
	Activity
}

type FreePlayActivity struct {
	Activity
}

type GolfActivity struct {
	Activity
}

type GuidedWorkoutActivity struct {
	Activity
}

type RunActivity struct {
	Activity
}

type PerformanceSummary struct {
	FinishHeartRate             int            `json:"finishHeartRate"`
	RecoveryHeartRateAt1Minute  int            `json:"recoveryHeartRateAt1Minute"`
	RecoveryHeartRateAt2Minutes int            `json:"recoveryHeartRateAt2Minutes"`
	HeartRateZones              HeartRateZones `json:"heartRateZones"`
}

type SleepActivity struct {
	Activity
}

type Activity struct {
	AwakeDuration              string                `json:"awakeDuration"`
	SleepDuration              string                `json:"sleepDuration"`
	NumberOfWakeups            int                   `json:"numberOfWakeups"`
	FallAsleepDuration         string                `json:"fallAsleepDuration"`
	SleepEfficiencyPercentage  int                   `json:"sleepEfficiencyPercentage"`
	TotalRestlessSleepDuration string                `json:"totalRestlessSleepDuration"`
	TotalRestfulSleepDuration  string                `json:"totalRestlessSleepDuration"`
	RestingHeartRate           int                   `json:"restingHeartRate"`
	FallAsleepTime             *time.Time            `json:"fallAsleepTime"`
	WakeupTime                 *time.Time            `json:"wakeupTime"`
	RoundsPerformed            int                   `json:"roundsPerformed"`
	RepetitionsPerformed       int                   `json:"repetitionsPerformed"`
	WorkoutPlanID              string                `json:"workoutPlanId"`
	PerformanceSummary         PerformanceSummary    `json:"performanceSummary"`
	ActivityType               string                `json:"activityType"`
	ActivitySegment            []ActivitySegment     `json:"activitySegments"`
	ID                         string                `json:"id"`
	UserID                     string                `json:"userId"`
	DeviceID                   string                `json:"deviceId"`
	StartTime                  *time.Time            `json:"startTime"`
	EndTime                    *time.Time            `json:"endTime"`
	DayID                      *time.Time            `json:"dayId"`
	CreatedTime                *time.Time            `json:"createdTime"`
	CreatedBy                  string                `json:"createdBy"`
	Name                       string                `json:"name"`
	Duration                   string                `json:"duration"`
	MinuteSummaries            []Summary             `json:"minuteSummaries"`
	HeartRateSummary           HeartRateSummary      `json:"heartRateSummary"`
	CaloriesBurnedSummary      CaloriesBurnedSummary `json:"caloriesBurnedSummary"`
	UVExposure                 string                `json:"uvExposure"`
	DistanceSummary            DistanceSummary       `json:"distanceSummary"`
	PausedDuration             string                `json:"pausedDuration"`
	SplitDistance              int                   `json:"splitDistance"`
	MapPoints                  []MapPoint            `json:"mapPoints"`
	TotalStepCount             int                   `json:"totalStepCount"`
	TotalDistanceWalked        int                   `json:"totalDistanceWalked"`
	ParOrBetterCount           int                   `json:"parOrBetterCount"`
	LongestDriveDistance       int                   `json:"longestDriveDistance"`
	LongestStrokeDistance      int                   `json:"longestStrokeDistance"`
	ChildActivities            []Activity            `json:"childActivities"`
}

type ActivitySegment struct {
	SleepTime             int                   `json:"sleepTime"`
	DayID                 *time.Time            `json:"dayId"`
	SleepType             string                `json:"sleepType"`
	SegmentID             int                   `json:"segmentId"`
	StartTime             *time.Time            `json:"startTime"`
	EndTime               *time.Time            `json:"endTime"`
	Duration              string                `json:"duration"`
	HeartRateSummary      HeartRateSummary      `json:"heartRateSummary"`
	CaloriesBurnedSummary CaloriesBurnedSummary `json:"caloriesBurnedSummary"`
	SegmentType           string                `json:"segmentType"`
	DistanceSummary       DistanceSummary       `json:"distanceSummary"`
	PausedDuration        string                `json:"pausedDuration"`
	HeartRateZones        HeartRateZones        `json:"heartRateZones"`
	SplitDistance         int                   `json:"splitDistance"`
	CircuitOrdinal        int                   `json:"circuitOrdinal"`
	CircuitType           int                   `json:"circuitType"`
	HoleNumber            int                   `json:"holeNumber"`
	StepCount             int                   `json:"stepCount"`
	DistanceWalked        int                   `json:"distanceWalked"`
}

type HeartRateZones struct {
	UnderHealthyHeart int `json:"underHealthyHeart"`
	UnderAerobic      int `json:"underAerobic"`
	Aerobic           int `json:"aerobic"`
	Anaerobic         int `json:"anaerobic"`
	FitnessZone       int `json:"fitnessZone"`
	HealthyHeart      int `json:"healthyHeart"`
	Redline           int `json:"redline"`
	OverRedline       int `json:"overRedline"`
}

type MapPoint struct {
	SecondsSinceStart int      `json:"secondsSinceStart"`
	MapPointType      string   `json:"mapPointType"`
	Ordinal           int      `json:"ordinal"`
	ActualDistance    int      `json:"actualDistance"`
	TotalDistance     int      `json:"totalDistance"`
	HeartRate         int      `json:"heartRate"`
	Pace              int      `json:"pace"`
	ScaledPace        int      `json:"scaledPace"`
	Speed             int      `json:"speed"`
	Location          Location `json:"location"`
	IsPaused          bool     `json:"isPaused"`
	IsResume          bool     `json:"isResume"`
}

type Location struct {
	SpeedOverGround           int `json:"speedOverGround"`
	Latitude                  int `json:"latitude"`
	Longitude                 int `json:"longitude"`
	ElevationFromMeanSeaLevel int `json:"elevationFromMeanSeaLevel"`
	EstimatedHorizontalError  int `json:"estimatedHorizontalError"`
	EstimatedVerticalError    int `json:"estimatedVerticalError"`
}
