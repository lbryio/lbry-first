package youtube

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/asaskevich/govalidator"
	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
var clientSecret string

func getClient(scope string) (*http.Client, error) {
	if clientSecret == "" {
		clientSecret = os.Getenv("CLIENTSECRET")

	}

	if govalidator.IsHexadecimal(clientSecret) {
		secretbytes, err := hex.DecodeString(clientSecret)
		if err != nil {
			return nil, errors.Err(err)
		}

		clientSecret = string(secretbytes)
	}

	ctx := context.Background()
	// If modifying the scope, delete your previously saved credentials
	// at ~/.credentials/youtube-go.json
	config, err := google.ConfigFromJSON([]byte(clientSecret), scope)
	if err != nil {
		return nil, errors.Err("Unable to parse client secret file to config: %v", err)
	}

	config.RedirectURL = "http://localhost:8090"

	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, errors.Err("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		fmt.Println("Trying to get token from web")
		tok, err = getTokenFromWeb(config, authURL)
		if err != nil {
			return nil, errors.Err(err)
		}
		err = saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok), err
}

// startWebServer starts a web server that listens on http://localhost:8090.
// The webserver waits for an oauth code in the three-legged auth flow.
func startWebServer() (codeCh chan string, err error) {
	listener, err := net.Listen("tcp", "localhost:8090")
	if err != nil {
		return nil, err
	}
	codeCh = make(chan string)

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		codeCh <- code // send code to OAuth flow
		listener.Close()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Received code: %v\r\nYou can now safely close this browser window.", code)
	}))

	return codeCh, nil
}

// openURL opens a browser window to the specified location.
// This code originally appeared at:
//   http://stackoverflow.com/questions/10377243/how-can-i-launch-a-process-that-is-not-a-file-in-go
func openURL(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("Cannot open URL %s on this platform", url)
	}
	return err
}

// Exchange the authorization code for an access token
func exchangeToken(config *oauth2.Config, code string) (*oauth2.Token, error) {
	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, errors.Err("Unable to retrieve token %v", err)
	}
	return tok, nil
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config, authURL string) (*oauth2.Token, error) {
	codeCh, err := startWebServer()
	if err != nil {
		return nil, errors.Prefix("Unable to start a web server.", err)
	}

	err = openURL(authURL)
	if err != nil {
		return nil, errors.Err("Unable to open authorization URL in web server: %v", err)
	} else {
		logrus.Info("Your browser has been opened to an authorization URL.",
			" This program will resume once authorization has been provided.")
		logrus.Info(authURL)
	}

	// Wait for the web server to get the code.
	code := <-codeCh
	return exchangeToken(config, code)
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("youtube-go.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func isTokenOnFile() bool {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return false
	}
	_, err = tokenFromFile(cacheFile)
	return err == nil
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) error {
	logrus.Info("trying to save token")
	logrus.Infof("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Err("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
