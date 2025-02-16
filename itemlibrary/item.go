package itemlibrary

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

// LibraryItem Google Photo item and meta data
type Item struct {
	//Google Photos item
	photoslibrary.MediaItem
	//Actual file name that was used, without a path
	UsedFileName string

	options *Options
}

func NewItem(mediaItem *photoslibrary.MediaItem, options *Options) *Item {
	item := new(Item)
	item.MediaItem = *mediaItem
	item.options = options
	return item
}

// JSONExists Check if the JSON file exists for the item
func JSONExists(mediaItem *photoslibrary.MediaItem, options *Options) (bool, error) {
	filePath := GetJSONFilePath(mediaItem, options)
	_, err := os.Stat(filePath)

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Load TODO
func Load(filePath string, options *Options) (*Item, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	item := new(Item)
	err = json.Unmarshal(bytes, item)
	if err != nil {
		return nil, err
	}
	// After loading, set the current options
	item.options = options
	return item, nil
}

// LoadFromJSON Load the Item from a JSON file
func LoadFromJSON(mediaItem *photoslibrary.MediaItem, options *Options) (*Item, error) {
	exists, err := JSONExists(mediaItem, options)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	filePath := GetJSONFilePath(mediaItem, options)
	return Load(filePath, options)
}

// FindNonConflictingItem Find an item that does not have a conflicting name
func FindNonConflictingItem(mediaItem *photoslibrary.MediaItem, options *Options) (*Item, error) {
	var item *Item
	var conflict uint

	for conflict = 0; true; conflict++ {
		if conflict >= 255 {
			return nil, errors.New("Too many conflict attempts")
		}

		item = NewItem(mediaItem, options)
		item.UsedFileName = item.createBaseFileName(uint(conflict))

		exists, err := item.ImageFileExists()
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
	}

	return item, nil
}

// WalkLibrary TODO
func WalkLibrary(options *Options, walkFunc func(*Item) error) error {
	return filepath.Walk(options.BackupFolder, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(filePath) != ".json" {
			return nil
		}

		item, err := Load(filePath, options)
		if err != nil {
			return err
		}

		return walkFunc(item)
	})
}

// CreateJSON Create a JSON file to represent the Item
func (i *Item) CreateJSON() error {
	filePath := GetJSONFilePath(&i.MediaItem, i.options)
	_, err := os.Stat(filePath)
	if err == nil {
		return errors.New("JSON file already exists")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// TODO: move this log statement to downloader
	log.Printf("Creating JSON for '%v' ", i.UsedFileName)
	bytes, err := i.MarshalJSON()
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(filePath), 0700)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, bytes, 0644)
}

// GetImageFilePath Get the file path for the image
func (i *Item) GetImageFilePath() string {
	var fileName string

	if i.options.UseFileName {
		fileName = i.UsedFileName
		if fileName == "" {
			fileName = i.Filename
		}

		fileName = filepath.Join(GetFolderPath(&i.MediaItem, i.options), fileName)
	} else {
		fileName = GetLegacyPrefixFilePath(&i.MediaItem, i.options)

		//Append the file extension based on the mime type
		ext, _ := mime.ExtensionsByType(i.MimeType)
		if len(ext) > 0 {
			fileName += ext[0]
		}
	}

	return fileName
}

// ImageFileExists Check if the image has been downloaded to a file
func (i *Item) ImageFileExists() (bool, error) {
	folderPath := GetFolderPath(&i.MediaItem, i.options)
	filePath := i.GetImageFilePath()

	// Ensure folder exists prior to checking for file
	_, err := os.Stat(folderPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	_, err = os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Remove Delete the JSON and image file for the item
func (i *Item) Remove() error {
	err1 := os.Remove(GetJSONFilePath(&i.MediaItem, i.options))
	err2 := os.Remove(i.GetImageFilePath())

	if err1 != nil {
		return err1
	}
	return err2
}

// marshalJSON Marshal item as json
func (i *Item) marshalJSON() ([]byte, error) {
	data, err := i.MediaItem.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	m["UsedFileName"] = i.UsedFileName
	return json.Marshal(m)
}

// createBaseFileName Create a file name based on the item's file name and
// conflict number
func (i *Item) createBaseFileName(conflict uint) string {
	fileName := i.GetImageFilePath()
	if conflict > 0 {
		fileExtension := filepath.Ext(fileName)
		index := strings.LastIndex(fileName, fileExtension)
		fileName = fileName[0:index] + " (" + fmt.Sprintf("%d", conflict) + ")" + fileExtension
	}

	return filepath.Base(fileName)
}
