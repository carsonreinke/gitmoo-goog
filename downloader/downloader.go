package downloader

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dtylman/gitmoo-goog/itemlibrary"
	"github.com/dustin/go-humanize"
	"github.com/fujiwara/shapeio"
	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
	errgroup "golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
)

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
}
type MediaItemsServiceSearch func(*photoslibrary.SearchMediaItemsRequest) MediaItemsSearchCall
type MediaItemsService interface {
	Search(*photoslibrary.SearchMediaItemsRequest) MediaItemsSearchCall
}
type MediaItemsSearchCall interface {
	Do(opts ...googleapi.CallOption) (*photoslibrary.SearchMediaItemsResponse, error)
}

// Downloader Struct for downloading photos into managed folders, use factory
// method `NewDownloader` to create
type Downloader struct {
	waitGroup                  *errgroup.Group
	concurrentDownloadRoutines chan struct{}
	stats                      *Stats
	client                     HTTPClient
	Options                    *Options
}

// NewDownloader factory to create a Downloader instance with defaults
func NewDownloader() *Downloader {
	return NewDownloaderWithClient(http.DefaultClient)
}

// NewDownloaderWithClient factory to create a Downloader instance using the provided HTTP client
func NewDownloaderWithClient(client HTTPClient) *Downloader {
	downloader := new(Downloader)
	downloader.waitGroup = new(errgroup.Group)
	downloader.stats = new(Stats)
	downloader.client = client
	downloader.concurrentDownloadRoutines = make(chan struct{}, 1)

	downloader.Options = new(Options)
	downloader.Options.BackupFolder, _ = os.Getwd()
	downloader.Options.FolderFormat = filepath.Join("2006", "January")
	downloader.Options.ConcurrentDownloads = 1

	return downloader
}

// downloadImage Download the image file for the library item
func (d *Downloader) downloadImage(item *itemlibrary.Item) error {
	var url string

	filePath := item.GetImageFilePath()

	//Ensure directories exist
	os.MkdirAll(filepath.Dir(filePath), 0755)

	if strings.HasPrefix(strings.ToLower(item.MediaItem.MimeType), "video") {
		url = item.MediaItem.BaseUrl + "=dv"
	} else {
		if d.Options.IncludeEXIF {
			url = fmt.Sprintf("%v=d", item.MediaItem.BaseUrl)
		} else {
			url = fmt.Sprintf("%v=w%v-h%v", item.MediaItem.BaseUrl, item.MediaItem.MediaMetadata.Width, item.MediaItem.MediaMetadata.Height)
		}
	}
	output, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer output.Close()

	response, err := d.client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	//Limit download rate
	rateLimitedReader := shapeio.NewReader(response.Body)
	if d.Options.DownloadThrottle > 0.0 {
		rateLimitedReader.SetRateLimit((d.Options.DownloadThrottle * 1024) / float64(d.Options.ConcurrentDownloads))
	}

	n, err := io.Copy(output, rateLimitedReader)
	if err != nil {
		return err
	}

	// close file to prevent conflicts with writing new timestamp in next step
	output.Close()

	//If timestamp is available, set access time to current timestamp and set modified time to the time the item was first created (not when it was uploaded to Google Photos)
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err == nil {
		err = os.Chtimes(filePath, time.Now(), t)
		if err != nil {
			return errors.New("failed writing timestamp to file: " + err.Error())
		}
	}

	log.Printf("Downloaded '%v' [saved as '%v'] (%v)", item.Filename, item.UsedFileName, humanize.Bytes(uint64(n)))

	d.stats.UpdateStatsDownloaded(uint64(n), 1)

	return nil
}

// createImage Download the image file if it does not already exist
func (d *Downloader) createImage(item *itemlibrary.Item) error {
	exists, err := item.ImageFileExists()
	if err != nil {
		return err
	}

	if !exists {
		filePath := item.GetImageFilePath()

		//Touch file before downloading (to avoid file name conflicts)
		err := ioutil.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			return err
		}

		//Wait till room on channel to start download
		d.concurrentDownloadRoutines <- struct{}{}
		d.waitGroup.Go(func() error {
			err := d.downloadImage(item)

			//Inform channel download is complete (no matter if there was an error or not)
			<-d.concurrentDownloadRoutines

			return err
		})
	} else {
		log.Printf("Skipping '%v' [saved as '%v']", item.Filename, item.UsedFileName)
		d.stats.UpdateStatsSkipped(1)
	}
	return nil
}

// downloadItem Download the Google Photos library item by downloading image
// and creating supporting JSON file
func (d *Downloader) downloadItem(item *photoslibrary.MediaItem) error {
	var libraryItem *itemlibrary.Item
	itemLibraryOptions := d.Options.ToItemLibraryOptions()

	exists, err := itemlibrary.JSONExists(item, itemLibraryOptions)
	if err != nil {
		return err
	}

	if exists {
		libraryItem, err = itemlibrary.LoadFromJSON(item, itemLibraryOptions)
		if err != nil {
			return err
		}
	} else {
		libraryItem, err = itemlibrary.FindNonConflictingItem(item, itemLibraryOptions)
		if err != nil {
			return err
		}

		err = libraryItem.CreateJSON()
		if err != nil {
			return err
		}
	}

	return d.createImage(libraryItem)
}

// waitForCompletion Wait for all downloads to complete
func (d *Downloader) waitForCompletion() error {
	err := d.waitGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}

// DownloadAll Downloads all files
func (d *Downloader) DownloadAll(svc MediaItemsServiceSearch) error {
	hasMore := true
	sleepTime := time.Duration(time.Second * time.Duration(d.Options.Throttle))

	//Setup channel buffer to limit downloads
	d.concurrentDownloadRoutines = make(chan struct{}, d.Options.ConcurrentDownloads)

	req := &photoslibrary.SearchMediaItemsRequest{PageSize: int64(d.Options.PageSize), AlbumId: d.Options.AlbumID}
	for hasMore {
		items, err := svc(req).Do()
		if err != nil {
			return err
		}
		for _, m := range items.MediaItems {
			d.stats.UpdateStatsTotal(1)

			err = d.downloadItem(m)
			if err != nil {
				log.Printf("Failed to download '%v' [id %v]: %v", m.Filename, m.Id, err)
				d.stats.UpdateStatsError(1)
			}

			if d.stats.Total >= d.Options.MaxItems {
				hasMore = false
				break
			}
		}
		req.PageToken = items.NextPageToken
		if req.PageToken == "" {
			hasMore = false
		}

		err = d.waitForCompletion()
		if err != nil {
			return err
		}

		if hasMore {
			log.Printf("Processed: %v, Downloaded: %v, Skipped: %v, Errors: %v, Total Size: %v", d.stats.Total, d.stats.Downloaded, d.stats.Skipped, d.stats.Errors, humanize.Bytes(d.stats.TotalSize))
			time.Sleep(sleepTime)
		}
	}

	log.Printf("Finished: %v, Downloaded: %v, Skipped: %v, Errors: %v, Total Size: %v", d.stats.Total, d.stats.Downloaded, d.stats.Skipped, d.stats.Errors, humanize.Bytes(d.stats.TotalSize))
	return nil
}
