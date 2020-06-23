package youtube

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lbryio/lbry-first/commands/server/lbry"

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
	AuthToken       string
	ClaimName       string
	URI             string
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
	if ytVideo == nil {
		return errors.Err("failed to return youtube confirmation of upload. Please check the status.")
	}

	reply.Video = *ytVideo
	return notifyIAPIs(ytVideo, args)
}

type AuthResponse struct {
	HasAuth bool
}

type AuthArgs struct {
	AuthToken string
}

func (t *YoutubeService) HasAuth(_ *http.Request, args *AuthArgs, reply *AuthResponse) error {
	reply.HasAuth = isTokenOnFile()
	lbry.AuthToken = args.AuthToken
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

func (t *YoutubeService) Remove(r *http.Request, args *RemoveArgs, reply *RemoveResponse) error {
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
	var fileName *youtube.VideoFileDetails
	if false { //len(args.FilePath) > 0 {
		dir, name := filepath.Split(args.FilePath)
		logrus.Debug("Dir:", dir, "Name:", name)
		fileName = &youtube.VideoFileDetails{
			FileName: name,
		}
	}
	var monetizationDetails *youtube.VideoMonetizationDetails
	if false {
		monetizationDetails = &youtube.VideoMonetizationDetails{
			Access: &youtube.AccessPolicy{
				Allowed: !args.MonetizationOff,
			},
		}
	}

	upload := &youtube.Video{
		FileDetails: fileName,
		Snippet: &youtube.VideoSnippet{
			Title:       args.Title,
			Description: args.Description,
			Thumbnails:  thumbNail,
		},
		MonetizationDetails: monetizationDetails,
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

func notifyIAPIs(video *youtube.Video, args *YoutubeArgs) error {
	me := lbry.GetUserInfo()
	if me.Error != nil {
		return errors.Err(me.Error)
	}
	if len(me.Data.YoutubeChannels) > 0 {
		for _, yt := range me.Data.YoutubeChannels {
			if yt.YtChannelName == video.Snippet.ChannelTitle {
				parts := strings.Split(args.URI, "#")
				var claimID string
				if len(parts) > 1 {
					claimID = parts[1]
				}
				if args.ClaimName != "" && claimID != "" {
					err := lbry.VideoStatusUpdate(video.Snippet.ChannelId, video.Id, args.ClaimName, claimID, "published", time.Now())
					if err != nil {
						return errors.Err(err)
					}
					return nil
				}
			}
		}
	}
	return nil
}
