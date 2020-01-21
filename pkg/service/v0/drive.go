package svc

import (
	"github.com/go-chi/render"
	"net/http"
	"strings"

	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/cs3org/reva/pkg/token"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"google.golang.org/grpc/metadata"
)

const defaultHeader = "x-access-token"

func getToken(r *http.Request) string {
	// 1. check Authorization header
	hdr := r.Header.Get("Authorization")
	t := strings.TrimPrefix(hdr, "Bearer ")
	if t != "" {
		return t
	}
	// TODO 2. check form encoded body parameter for POST requests, see https://tools.ietf.org/html/rfc6750#section-2.2

	// 3. check uri query parameter, see https://tools.ietf.org/html/rfc6750#section-2.3
	tokens, ok := r.URL.Query()["access_token"]
	if !ok || len(tokens[0]) < 1 {
		return ""
	}

	return tokens[0]
}

// GetRootDriveChildren implements the Service interface.
func (g Graph) GetRootDriveChildren(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msgf("Calling GetRootDriveChildren")
	accessToken := getToken(r)
	ctx := r.Context()
	ctx = token.ContextSetToken(ctx, accessToken)
	ctx = metadata.AppendToOutgoingContext(ctx, defaultHeader, accessToken)
	g.logger.Info().Msgf("provides access token %s", ctx)

	// TODO: read the path from request
	fn := "/"
	listChildren := true

	// TODO: where to get the client from
	client, err := g.GetClient()
	if err != nil {
		g.logger.Err(err).Msg("error getting grpc client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ref := &storageprovider.Reference{
		Spec: &storageprovider.Reference_Path{Path: fn},
	}
	req := &storageprovider.StatRequest{Ref: ref}
	res, err := client.Stat(ctx, req)
	if err != nil {
		g.logger.Error().Err(err).Msg("error sending a grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if res.Status.Code != cs3rpc.Code_CODE_OK {
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			g.logger.Error().Err(err).Msgf("resource not found %s", fn)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	info := res.Info
	infos := []*storageprovider.ResourceInfo{info}
	if info.Type == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER && listChildren {
		req := &storageprovider.ListContainerRequest{
			Ref: ref,
		}
		res, err := client.ListContainer(ctx, req)
		if err != nil {
			g.logger.Error().Err(err).Msgf("error sending list container grpc request %s", fn)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if res.Status.Code != cs3rpc.Code_CODE_OK {
			g.logger.Error().Err(err).Msgf("error calling grpc list container %s", fn)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		infos = append(infos, res.Infos...)
	}

	files, err := formatDriveItems(infos)
	if err != nil {
		g.logger.Error().Err(err).Msgf("error encoding response as json %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &listResponse{Value: files})
}


func cs3ResourceToDriveItem(res *storageprovider.ResourceInfo) (*msgraph.DriveItem, error) {
	/*
		{
			"value": [
			  {"name": "myfile.jpg", "size": 2048, "file": {} },
			  {"name": "Documents", "folder": { "childCount": 4} },
			  {"name": "Photos", "folder": { "childCount": 203} },
			  {"name": "my sheet(1).xlsx", "size": 197 }
			],
			"@odata.nextLink": "https://..."
		  }
	*/
	size := new(int)
	*size = int(res.Size) // uint64 -> int :boom:

	driveItem := &msgraph.DriveItem{
		BaseItem: msgraph.BaseItem{
			Name: &res.Path,
		},
		Size: size,
	}
	return driveItem, nil
}

func formatDriveItems(mds []*storageprovider.ResourceInfo) ([]*msgraph.DriveItem, error) {
	responses := make([]*msgraph.DriveItem, 0, len(mds))
	for i := range mds {
		res, err := cs3ResourceToDriveItem(mds[i])
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}

	return responses, nil
}
