package lbry

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lbryio/lbry.go/v2/extras/errors"
)

type UserMeResponse struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	Data    struct {
		ID                  int         `json:"id"`
		Language            string      `json:"language"`
		GivenName           string      `json:"given_name"`
		FamilyName          interface{} `json:"family_name"`
		CreatedAt           time.Time   `json:"created_at"`
		UpdatedAt           time.Time   `json:"updated_at"`
		InvitedByID         int         `json:"invited_by_id"`
		InvitedAt           time.Time   `json:"invited_at"`
		InvitesRemaining    int         `json:"invites_remaining"`
		InviteRewardClaimed bool        `json:"invite_reward_claimed"`
		IsEmailEnabled      bool        `json:"is_email_enabled"`
		PublishID           int         `json:"publish_id"`
		Country             interface{} `json:"country"`
		YoutubeChannels     []struct {
			YtChannelName    string      `json:"yt_channel_name"`
			LbryChannelName  string      `json:"lbry_channel_name"`
			ChannelClaimID   string      `json:"channel_claim_id"`
			SyncStatus       string      `json:"sync_status"`
			StatusToken      string      `json:"status_token"`
			Transferable     bool        `json:"transferable"`
			TransferState    string      `json:"transfer_state"`
			PublishToAddress interface{} `json:"publish_to_address"`
			PublicKey        string      `json:"public_key"`
		} `json:"youtube_channels"`
		PrimaryEmail       string        `json:"primary_email"`
		PasswordSet        bool          `json:"password_set"`
		LatestClaimedEmail interface{}   `json:"latest_claimed_email"`
		HasVerifiedEmail   bool          `json:"has_verified_email"`
		IsIdentityVerified bool          `json:"is_identity_verified"`
		IsRewardApproved   bool          `json:"is_reward_approved"`
		Groups             []interface{} `json:"groups"`
		DeviceTypes        []string      `json:"device_types"`
	} `json:"data"`
}

func GetUserInfo() UserMeResponse {
	c := http.Client{}
	form := make(url.Values)
	form.Set("auth_token", AuthToken)
	response, err := c.PostForm("https://api.lbry.com/user/me", form)
	if err != nil {
		return UserMeResponse{Error: errors.FullTrace(err)}
	}
	defer response.Body.Close()
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return UserMeResponse{Error: errors.FullTrace(err)}
	}
	var me UserMeResponse
	err = json.Unmarshal(b, &me)
	if err != nil {
		return UserMeResponse{Error: errors.FullTrace(err)}
	}
	return me
}

func VideoStatusUpdate(ytchannelID, ytVideoID, lbryClaimName, lbryClaimID, status string, publishedAt time.Time, size int) error {
	c := http.Client{}
	if len(AuthToken) > 7 {
		logrus.Info("Making API Call with token ", AuthToken[0:7], "...")
	} else {
		logrus.Warning("No auth token?")
	}
	form := make(url.Values)
	form.Set("auth_token", AuthToken)
	form.Set("youtube_channel_id", ytchannelID)
	form.Set("video_id", ytVideoID)
	form.Set("claim_name", lbryClaimName)
	form.Set("claim_id", lbryClaimID)
	form.Set("status", status)
	form.Set("metadata_version", strconv.Itoa(2))
	form.Set("transferred", strconv.FormatBool(true))
	form.Set("size", strconv.Itoa(size))
	form.Set("is_lbry_first", "true")
	form.Set("published_at", strconv.FormatInt(publishedAt.Unix(), 10))
	r, err := c.PostForm("https://api.lbry.com/yt/video_status", form)
	if r != nil {
		logrus.Info("APIs Call:", r.Status, r.StatusCode)
	}
	if err != nil {
		logrus.Info("APIs Call Error:", err)
		if strings.Contains(err.Error(), "No matching channel found for") {
			return nil
		}
		return errors.Err(err)
	}

	return nil
}
