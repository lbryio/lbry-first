package youtube

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
	"google.golang.org/api/youtube/v3"
)

type YoutubeArgs struct {
	Title           string
	Description     string
	FilePath        string
	Keywords        string
	ThumbnailURL    string
	PublishAt       string //premiere date (default now)
	PublishType     string // 'private', 'public', or 'unlisted'
	MonetizationOff bool
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
var uploadStatus = map[string]bool{"private": true, "public": true, "unlisted": true}

func (t *YoutubeService) Upload(_ *http.Request, args *YoutubeArgs, reply *UploadResponse) error {
	if args.FilePath == "" {
		return nil //Don't upload if no file path is provided ( likely an update...)
	}
	if args.PublishType != "" && !uploadStatus[args.PublishType] {
		return errors.Err("%s is not a valid publish type, should be 'private', 'public', or 'unlisted'", args.PublishType)
	}
	if args.PublishAt != "" && args.PublishType != "private" {
		return errors.Err("publish type of 'private' needs to be set if the publish at is set")
	}
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

func (t *YoutubeService) HasAuth(_ *http.Request, _ *AuthArgs, reply *AuthResponse) error {
	reply.HasAuth = isTokenOnFile()
	return nil
}

type SignupResponse struct{}

type SignUpArgs struct{}

func (t *YoutubeService) Signup(_ *http.Request, _ *SignUpArgs, _ *SignupResponse) error {
	_, err := getClient(youtube.YoutubeUploadScope)
	if err != nil {
		return errors.Err(err)
	}
	return nil
}

type RemoveResponse struct{}

type RemoveArgs struct{}

func (t *YoutubeService) Remove(_ *http.Request, _ RemoveArgs, _ *RemoveResponse) error {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return errors.Err(err)
	}
	_, err = tokenFromFile(cacheFile)
	if err == nil {
		err := os.Remove(cacheFile)
		if err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func upload(args *YoutubeArgs) (*youtube.Video, error) {
	if args.PublishType == "" {
		args.PublishType = "private"
	}

	client, err := getClient(youtube.YoutubeUploadScope)
	if err != nil {
		return nil, errors.Err(err)
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, errors.Prefix("Error creating YouTube client: %v", err)
	}

	var thumbNail *youtube.ThumbnailDetails
	if args.ThumbnailURL != "" {
		thumbNail = &youtube.ThumbnailDetails{
			Default: &youtube.Thumbnail{
				Url: args.ThumbnailURL,
			},
		}
	}

	upload := &youtube.Video{
		FileDetails: &youtube.VideoFileDetails{
			FileName: filepath.SplitList(args.FilePath)[1],
		},
		Snippet: &youtube.VideoSnippet{
			Title:       args.Title,
			Description: args.Description,
			Tags:        strings.Split(args.Keywords, ","),
			Thumbnails:  thumbNail,
		},
		MonetizationDetails: &youtube.VideoMonetizationDetails{
			Access: &youtube.AccessPolicy{
				Allowed: !args.MonetizationOff,
			},
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: args.PublishType,
			PublishAt:     args.PublishAt,
		},
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
