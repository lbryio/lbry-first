package youtube

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	"google.golang.org/api/youtube/v3"
)

type YoutubeArgs struct {
	Title       string
	Description string
	FilePath    string
	Category    string
	Keywords    string
}

type YoutubeService struct{}

type UploadResponse struct {
	Video youtube.Video
}

type uploadInfo struct {
	LastUpdate time.Time
	Start      time.Time
	Total      int64
	Progress   int64
}

var currentUpload uploadInfo

func (t *YoutubeService) Upload(r *http.Request, args *YoutubeArgs, reply *UploadResponse) error {
	ytVideo, err := upload(args)
	if err != nil {
		return err
	}

	reply.Video = *ytVideo
	return nil
}

type AuthResponse struct {
	HasAuth bool
}

type AuthArgs struct{}

func (t *YoutubeService) HasAuth(r *http.Request, args *AuthArgs, reply *AuthResponse) error {
	reply.HasAuth = isTokenOnFile()
	return nil
}

func upload(args *YoutubeArgs) (*youtube.Video, error) {
	if args.FilePath == "" {
		//return errors.Err("You must provide a filename of a video file to upload")
	}

	client, err := getClient(youtube.YoutubeUploadScope)
	if err != nil {
		return nil, errors.Err(err)
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, errors.Prefix("Error creating YouTube client: %v", err)
	}

	upload := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       args.Title,
			Description: args.Description,
			CategoryId:  "",
		},
		Status: &youtube.VideoStatus{PrivacyStatus: "private"},
	}

	// The API returns a 400 Bad Request response if tags is an empty string.
	if strings.Trim(args.Keywords, "") != "" {
		upload.Snippet.Tags = strings.Split(args.Keywords, ",")
	}

	call := service.Videos.Insert("snippet,status", upload)

	file, err := os.Open(args.FilePath)
	defer file.Close()
	if err != nil {
		return nil, errors.Err("Error opening %v: %v", args.FilePath, err)
	}
	currentUpload.Start = time.Now()
	response, err := call.ProgressUpdater(progressUpdate).Media(file).Do()
	if err != nil {
		return nil, errors.Err(err)
	}
	logrus.Infof("Upload successful! Video ID: %v\n", response.Id)
	return response, nil
}

func progressUpdate(current, total int64) {
	currentUpload.LastUpdate = time.Now()
	currentUpload.Progress = current
	currentUpload.Total = total
	logrus.Infof("Upload Progress: %d/%d in %.2f minutes", current, total, time.Since(currentUpload.Start).Minutes())
}
