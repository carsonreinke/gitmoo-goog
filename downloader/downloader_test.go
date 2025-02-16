package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtylman/gitmoo-goog/itemlibrary"
	"github.com/dtylman/gitmoo-goog/photoslibraryextended"
	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
	"google.golang.org/api/googleapi"
)

// TestMediaItemFileName Test to ensure that JSON with `FileName` will be
// populated under Media Item File Name
func TestMediaItemFileName(t *testing.T) {
	data := `
	{
		"baseUrl": "https://lh3.googleusercontent.com/1234",
		"id": "1234",
		"mediaMetadata": {
			"creationTime": "2019-10-13T17:33:43Z",
			"height": "3024",
			"photo": {
				"apertureFNumber": 1.7,
				"cameraMake": "motorola",
				"cameraModel": "Moto G (5) Plus",
				"focalLength": 4.28,
				"isoEquivalent": 400
			},
			"width": "4032"
		},
		"mimeType": "image/jpeg",
		"productUrl": "https://photos.google.com/1234",
		"filename": "IMG_1234.jpg"
	}
	`
	item := new(photoslibrary.MediaItem)
	json.Unmarshal([]byte(data), item)

	if item.Filename != "IMG_1234.jpg" {
		t.Errorf("photoslibrary.MediaItem.FileName = %v; want \"IMG_1234.jpg\"", item.Filename)
	}
}

func TestDownloadImage(t *testing.T) {
	succesfulTest := func(downloader *Downloader, mediaItem *photoslibrary.MediaItem, imageContents string) {
		item, err := itemlibrary.FindNonConflictingItem(mediaItem, downloader.Options.ToItemLibraryOptions())
		if err != nil {
			t.Fatalf("downloader.downloadImage: %v", err)
		}

		err = downloader.downloadImage(item)
		if err != nil {
			t.Fatalf("downloader.downloadImage: %v", err)
		}

		fileContents, err := readItemContents(item)
		if err != nil {
			t.Fatalf("downloader.downloadImage: %v", err)
		}
		if string(fileContents) != imageContents {
			t.Errorf("downloader.downloadImage() = %v; want %v", string(fileContents), imageContents)
		}
	}

	t.Run("Successful image", func(t *testing.T) {
		imageContents := "[FAKE IMAGE]"
		client := createMockHTTPClient(imageContents, nil)
		downloader := NewDownloaderWithClient(client)
		downloader.Options.BackupFolder = t.TempDir()

		succesfulTest(downloader, createMediaItem(true), imageContents)
	})

	t.Run("Successful video", func(t *testing.T) {
		videoContents := "[FAKE VIDEO]"
		client := createMockHTTPClient(videoContents, nil)
		downloader := NewDownloaderWithClient(client)
		downloader.Options.BackupFolder = t.TempDir()

		succesfulTest(downloader, createMediaItem(false), videoContents)
	})

	t.Run("Throttled succesful", func(t *testing.T) {
		imageContents := "[FAKE IMAGE]"
		client := createMockHTTPClient(imageContents, nil)
		downloader := NewDownloaderWithClient(client)
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.DownloadThrottle = 1.0

		succesfulTest(downloader, createMediaItem(true), imageContents)
	})

	t.Run("Request failed", func(t *testing.T) {
		mockError := errors.New("Mock Request failed")
		client := createMockHTTPClient("", mockError)
		downloader := NewDownloaderWithClient(client)
		downloader.Options.BackupFolder = t.TempDir()

		item, err := createItem(true, downloader)
		if err != nil {
			t.Fatalf("downloader.downloadImage: %v", err)
		}

		have := downloader.downloadImage(item)
		want := mockError

		if have != want {
			t.Errorf("downloader.downloadImage() = %v; want %v", have, want)
		}
	})
}

func TestCreateImage(t *testing.T) {
	t.Run("Existing", func(t *testing.T) {
		imageContents := "[FAKE IMAGE]"
		downloader := NewDownloaderWithClient(createMockHTTPClient(imageContents, nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		item, err := createItem(true, downloader)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}
		err = touchItem(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}

		err = downloader.createImage(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}

		fileContents, err := readItemContents(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}
		if string(fileContents) == imageContents {
			t.Errorf("downloader.createImage() = %v; want %v", string(fileContents), imageContents)
		}
	})

	t.Run("Not existing", func(t *testing.T) {
		imageContents := "[FAKE IMAGE]"
		downloader := NewDownloaderWithClient(createMockHTTPClient(imageContents, nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		item, err := createItem(true, downloader)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}

		err = downloader.createImage(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}

		err = downloader.waitForCompletion()
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}

		fileContents, err := readItemContents(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}
		if string(fileContents) != imageContents {
			t.Errorf("downloader.createImage() = %v; want %v", string(fileContents), imageContents)
		}
	})
}

func TestDownloadItem(t *testing.T) {
	t.Run("Existing", func(t *testing.T) {
		downloader := NewDownloaderWithClient(createMockHTTPClient("[FAKE IMAGE]", nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		item, err := createItem(true, downloader)
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		err = item.CreateJSON()
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}
		err = touchItem(item)
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		err = downloader.downloadItem(&item.MediaItem)
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		err = downloader.waitForCompletion()
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		fileContents, err := readItemContents(item)
		if err != nil {
			t.Fatalf("downloader.createImage: %v", err)
		}
		if string(fileContents) != "" {
			t.Errorf("downloader.createImage() = %v; want %v", string(fileContents), "")
		}
	})

	t.Run("Not existing", func(t *testing.T) {
		downloader := NewDownloaderWithClient(createMockHTTPClient("[FAKE IMAGE]", nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		mediaItem := createMediaItem(true)

		err := downloader.downloadItem(mediaItem)
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		err = downloader.waitForCompletion()
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}

		jsonExists, err := itemlibrary.JSONExists(mediaItem, downloader.Options.ToItemLibraryOptions())
		if err != nil {
			t.Fatalf("downloader.downloadItem: %v", err)
		}
		if !jsonExists {
			t.Errorf("downloader.downloadItem() = %v; want true", jsonExists)
		}
	})
}

func TestDownloadAll(t *testing.T) {
	t.Run("Empty results", func(t *testing.T) {
		downloader := NewDownloaderWithClient(createMockHTTPClient("[FAKE IMAGE]", nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		searchCall := createMockMediaItemsSearchCall([]*photoslibrary.MediaItem{})
		service := createMockMediaItemsService([]*MockMediaItemsSearchCall{searchCall})

		err := downloader.DownloadAll(func(smir *photoslibrary.SearchMediaItemsRequest) photoslibraryextended.MediaItemsSearchCall {
			return service.Search(smir)
		})
		if err != nil {
			t.Fatalf("downloader.DownloadAll: %v", err)
		}
	})

	t.Run("Single page", func(t *testing.T) {
		imageContents := "[FAKE IMAGE]"
		downloader := NewDownloaderWithClient(createMockHTTPClient(imageContents, nil))
		downloader.Options.BackupFolder = t.TempDir()
		downloader.Options.FolderFormat = ""

		item, err := createItem(true, downloader)
		if err != nil {
			t.Fatalf("downloader.DownloadAll: %v", err)
		}

		searchCall := createMockMediaItemsSearchCall([]*photoslibrary.MediaItem{
			&item.MediaItem,
		})
		service := createMockMediaItemsService([]*MockMediaItemsSearchCall{searchCall})

		err = downloader.DownloadAll(func(smir *photoslibrary.SearchMediaItemsRequest) photoslibraryextended.MediaItemsSearchCall {
			return service.Search(smir)
		})
		if err != nil {
			t.Fatalf("downloader.DownloadAll: %v", err)
		}

		fileContents, err := readItemContents(item)
		if err != nil {
			t.Fatalf("downloader.DownloadAll: %v", err)
		}
		if string(fileContents) != imageContents {
			t.Errorf("downloader.DownloadAll() = %v; want %v", string(fileContents), imageContents)
		}
	})
}

type MockHTTPClient struct {
	Response *http.Response
	Err      error
}

func (c *MockHTTPClient) Get(url string) (resp *http.Response, err error) {
	return c.Response, c.Err
}

func createMockHTTPClient(contents string, err error) *MockHTTPClient {
	body := io.NopCloser(strings.NewReader(contents))
	if err != nil {
		body = nil
	}

	return &MockHTTPClient{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
		},
		Err: err,
	}
}

type MockMediaItemsService struct {
	Calls []*MockMediaItemsSearchCall
}

type MockMediaItemsSearchCall struct {
	Response *photoslibrary.SearchMediaItemsResponse
	Err      error
}

func (s *MockMediaItemsService) Search(*photoslibrary.SearchMediaItemsRequest) *MockMediaItemsSearchCall {
	call := s.Calls[0]
	s.Calls = s.Calls[1:]

	token := ""
	if len(s.Calls) > 1 {
		token = fmt.Sprintf("%v", s.Calls[1])
	}
	call.Response.NextPageToken = token

	return call
}

func (c *MockMediaItemsSearchCall) Do(opts ...googleapi.CallOption) (*photoslibrary.SearchMediaItemsResponse, error) {
	return c.Response, c.Err
}

func createMockMediaItemsService(calls []*MockMediaItemsSearchCall) *MockMediaItemsService {
	return &MockMediaItemsService{
		Calls: calls,
	}
}

func createMockMediaItemsSearchCall(items []*photoslibrary.MediaItem) *MockMediaItemsSearchCall {
	return &MockMediaItemsSearchCall{
		Response: &photoslibrary.SearchMediaItemsResponse{
			MediaItems: items,
		},
	}
}

func createMediaItem(image bool) *photoslibrary.MediaItem {
	var extension string
	var mimeType string
	if image {
		extension = "jpg"
		mimeType = "image/jpeg"
	} else {
		extension = "mp4"
		mimeType = "video/mp4"
	}

	mediaItem := new(photoslibrary.MediaItem)
	mediaItem.Id = "12345678901234567890"
	mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
	mediaItem.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"
	mediaItem.Filename = "test." + extension
	mediaItem.MimeType = mimeType
	mediaItem.BaseUrl = "http://localhost/12345678901234567890"
	mediaItem.ProductUrl = "http://localhost/12345678901234567890"

	return mediaItem
}

func createItem(image bool, downloader *Downloader) (*itemlibrary.Item, error) {
	mediaItem := createMediaItem(image)
	item, err := itemlibrary.FindNonConflictingItem(mediaItem, downloader.Options.ToItemLibraryOptions())
	if err != nil {
		return nil, err
	}

	return item, nil
}

func readItemContents(item *itemlibrary.Item) (string, error) {
	filePath := item.GetImageFilePath()
	_, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	var fileContents []byte
	fileContents, err = os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(fileContents), nil
}

func touchItem(item *itemlibrary.Item) error {
	path := item.GetImageFilePath()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte{}, 0644)
	if err != nil {
		return err
	}

	return nil
}
