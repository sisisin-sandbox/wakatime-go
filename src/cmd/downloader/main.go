package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

type contextKey string

const loggerKey contextKey = "logger"

const defaultUserID = "52f058ec-e04e-436b-906d-eff6c461abf5"

func WithLogger(ctx context.Context) context.Context {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return context.WithValue(ctx, loggerKey, logger)
}

func main() {
	ctx := WithLogger(context.Background())
	logger := LoggerFromCtx(ctx)

	yesterday := time.Now().AddDate(0, 0, -1)
	targetDateStr := flag.String("target-date", yesterday.Format("2006-01-02"), "Target date to process. format: yyyy-mm-dd")
	targetDate, err := time.Parse("2006-01-02", *targetDateStr)
	if err != nil {
		logger.Info("failed to parse target date", slog.Any("error", err))
		os.Exit(1)
	}

	userID := flag.String("user-id", defaultUserID, "User ID to process")
	flag.Parse()

	logger.Info("process start", slog.String("target_date", *targetDateStr))

	client := newWakatimeClient(ctx)
	res, err := client.getProjects(*userID, *targetDateStr)

	if err != nil {
		logger.Info("failed to getProjects", slog.Any("error", err))
		os.Exit(1)
	}

	if len(res.Summary.Data) == 0 {
		logger.Info("no projects found")
		os.Exit(0)
	}

	details := make([]map[string]interface{}, 0, len(res.Summary.Data[0].Projects))
	for _, project := range res.Summary.Data[0].Projects {
		resp, err := client.getProjectDetails(project.Name, *userID, *targetDateStr)
		if err != nil {
			logger.Info("failed to getProjectDetails", slog.Group("jsonPayload",
				slog.Any("error", err),
				slog.String("project_name", project.Name),
				slog.String("user_id", *userID),
				slog.String("target_date", *targetDateStr),
			))
			continue
		}
		var detail map[string]interface{}
		err = json.Unmarshal([]byte(*resp), &detail)
		if err != nil {
			logger.Info("failed to getProjectDetails", slog.Group("jsonPayload",
				slog.Any("error", err),
				slog.String("project_name", project.Name),
				slog.String("user_id", *userID),
				slog.String("target_date", *targetDateStr),
			))
			continue
		}

		details = append(details, detail)
	}

	now := time.Now()
	var summariesMap map[string]interface{}
	err = json.Unmarshal([]byte(res.RawResponse), &summariesMap)

	if err != nil {
		logger.Info("failed to unmarshal summaries", slog.Any("error", err))
		os.Exit(1)
	}

	o := Output{
		Meta: meta{
			DownloadedAt: now.Format(time.RFC3339),
		},
		Parameters: outParams{
			TargetDate: *targetDateStr,
		},
		Summaries: summariesMap,
		ByDetails: details,
	}

	err = os.Mkdir(".tmp", 0777)
	if err != nil && !os.IsExist(err) {
		logger.Info("failed to create tmp directory", slog.Any("error", err))
		os.Exit(1)
	}
	fileName := fmt.Sprintf(".tmp/output_%v.json", now.Format("2006-01-02 15:04:05"))
	file, err := os.Create(fileName)
	if err != nil {
		logger.Info("failed to create output file", slog.Any("error", err))
		os.Exit(1)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	// enc.SetIndent("", "  ")
	err = enc.Encode(o)
	if err != nil {
		logger.Info("failed to encode output", slog.Any("error", err))
		os.Exit(1)
	}

	err = uploadToGCS(ctx, "wakatime", &targetDate, fileName)
	if err != nil {
		logger.Info("failed to upload to GCS", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("process end")
}

type meta struct {
	DownloadedAt string `json:"downloaded_at"`
}
type outParams struct {
	TargetDate string `json:"target_date"`
}
type Output struct {
	Meta       meta                     `json:"meta"`
	Parameters outParams                `json:"parameters"`
	Summaries  any                      `json:"summaries"`
	ByDetails  []map[string]interface{} `json:"by_details"`
}

func LoggerFromCtx(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey).(*slog.Logger)
}

type WakatimeClient struct {
	ctx     context.Context
	baseUrl string
	apiKey  string
}

func newWakatimeClient(ctx context.Context) WakatimeClient {
	apiKey := os.Getenv("WAKATIME_KEY")
	if apiKey == "" {
		panic("WAKATIME_KEY must be set")
	}

	return WakatimeClient{
		ctx:     ctx,
		baseUrl: "https://wakatime.com/api/v1",
		apiKey:  apiKey,
	}
}

type Project struct {
	Name string `json:"name"`
}
type ProjectData struct {
	Projects []Project `json:"projects"`
}
type Summary struct {
	Data []ProjectData `json:"data"`
}
type GetProjectsResult struct {
	RawResponse string
	Summary     Summary
}

func buildSummariesUrl(baseUrl string, userID string, apiKey string, targetDate string, projectName *string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%v/users/%v/summaries", baseUrl, userID))
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("api_key", apiKey)
	params.Add("start", targetDate)
	params.Add("end", targetDate)
	if projectName != nil {
		params.Add("project", *projectName)
	}
	u.RawQuery = params.Encode()

	return u.String(), nil
}

func (c *WakatimeClient) getProjects(userID string, targetDate string) (*GetProjectsResult, error) {
	u, err := buildSummariesUrl(c.baseUrl, userID, c.apiKey, targetDate, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rawResponse := string(body)

	var s Summary
	err = json.Unmarshal(body, &s)
	if err != nil {
		return nil, err
	}

	return &GetProjectsResult{
		RawResponse: rawResponse,
		Summary:     s,
	}, nil
}

func (c *WakatimeClient) getProjectDetails(projectName string, userID string, targetDate string) (*string, error) {
	u, err := buildSummariesUrl(c.baseUrl, userID, c.apiKey, targetDate, &projectName)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(u)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := string(body)
	return &result, nil
}

func uploadToGCS(ctx context.Context, bucketName string, targetDate *time.Time, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer file.Close()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	objectName := fmt.Sprintf("raw/%v_summary.json", targetDate.Format("2006_01_02"))
	wc := bucket.Object(objectName).NewWriter(ctx)
	if _, err = io.Copy(wc, file); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	return nil
}
