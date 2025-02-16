package photoslibraryextended

import (
	"net/http"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
	"google.golang.org/api/googleapi"
)

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
}
type MediaItemsServiceSearch func(*photoslibrary.SearchMediaItemsRequest) MediaItemsSearchCall
type MediaItemsServiceGet func(mediaItemId string) MediaItemsGetCall

type MediaItemsService interface {
	Search(*photoslibrary.SearchMediaItemsRequest) MediaItemsSearchCall
	Get(mediaItemId string) MediaItemsGetCall
}

type MediaItemsSearchCall interface {
	Do(opts ...googleapi.CallOption) (*photoslibrary.SearchMediaItemsResponse, error)
}

type MediaItemsGetCall interface {
	Do(opts ...googleapi.CallOption) (*photoslibrary.MediaItem, error)
}
