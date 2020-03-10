package youtube

import (
	"fmt"
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

type Response struct {
	Result string
}

type uploadInfo struct {
	LastUpdate time.Time
	Start      time.Time
	Total      int64
	Progress   int64
}

var currentUpload uploadInfo

func (t *YoutubeService) Upload(r *http.Request, args *YoutubeArgs, reply *Response) error {
	err := upload(args)
	if err != nil {
		logrus.Error(errors.FullTrace(err))
		return err
	}
	reply.Result = fmt.Sprintf("Video uploaded to youtube ( %s )", args.Title)
	return nil
}

func upload(args *YoutubeArgs) error {
	if args.FilePath == "" {
		//return errors.Err("You must provide a filename of a video file to upload")
	}

	client, err := getClient(youtube.YoutubeUploadScope)
	if err != nil {
		return errors.Err(err)
	}

	service, err := youtube.New(client)
	if err != nil {
		return errors.Prefix("Error creating YouTube client: %v", err)
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
		return errors.Err("Error opening %v: %v", args.FilePath, err)
	}
	currentUpload.Start = time.Now()
	response, err := call.ProgressUpdater(progressUpdate).Media(file).Do()
	if err != nil {
		return errors.Err(err)
	}
	logrus.Infof("Upload successful! Video ID: %v\n", response.Id)
	return nil
}

func progressUpdate(current, total int64) {
	currentUpload.LastUpdate = time.Now()
	currentUpload.Progress = current
	currentUpload.Total = total
	logrus.Infof("Upload Progress: %d/%d in %.2f minutes", current, total, time.Since(currentUpload.Start).Minutes())
}
