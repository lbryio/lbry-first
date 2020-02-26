package status

import "net/http"

type ServerService struct{}

type ServerArgs struct {
}

type Response struct {
	Version string
	Message string
	Running bool
	Commit  string
}

func (t *ServerService) Status(r *http.Request, args *ServerArgs, reply *Response) error {
	reply.Running = true
	reply.Message = "All good to go!"
	return nil
}
