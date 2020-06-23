package server

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/lbryio/lbry-first/commands/server/services/status"
	"github.com/lbryio/lbry-first/commands/server/services/youtube"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
)

func Start() {
	logrus.SetOutput(os.Stdout)
	rpcServer := rpc.NewServer()

	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	ytService := new(youtube.YoutubeService)
	statusService := new(status.ServerService)

	rpcServer.RegisterService(ytService, "youtube")
	rpcServer.RegisterService(statusService, "server")
	rpcServer.RegisterBeforeFunc(beforeHook)
	rpcServer.RegisterAfterFunc(afterHook)

	router := mux.NewRouter()
	router.Handle("/rpc", rpcServer)
	logrus.Info("Running RPC Server @ http://localhost:1337/rpc")
	logrus.Fatal(http.ListenAndServe(":1337", router))
}

func beforeHook(info *rpc.RequestInfo) {

}

func afterHook(info *rpc.RequestInfo) {
	if info.Error != nil {
		info.StatusCode = http.StatusInternalServerError
		logrus.Error(errors.FullTrace(info.Error))
	}
}
