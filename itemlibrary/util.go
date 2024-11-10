package itemlibrary

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

// GetFolderPath Path of the to store JSON and image files for the particular MediaItem
func GetFolderPath(item *photoslibrary.MediaItem, options *Options) string {
	//TODO Check that item.MediaMetadata exists
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		//Default to an epoch if cannot parse time
		t, err = time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	}

	return filepath.Join(options.BackupFolder, t.Format(options.FolderFormat))
}

// GetLegacyPrefixFilePathByTime Build a file path based on the image creation
// time, file extension will need to be appened after
func GetLegacyPrefixFilePathByTime(item *photoslibrary.MediaItem, options *Options) (string, error) {
	//TODO check that item.MediaMetadata and item.Id exist
	t, err := time.Parse(time.RFC3339, item.MediaMetadata.CreationTime)
	if err != nil {
		return "", err
	}
	//TODO Assuming item.Id is over a certain length without checking
	name := fmt.Sprintf("%v_%v", t.Day(), item.Id[len(item.Id)-8:])
	return filepath.Join(GetFolderPath(item, options), name), nil
}

// GetLegacyPrefixFilePathByHash Build a file path when missing a image
// creation time based on a MD5 hash of the Media Item ID, file extension will
// need to be appened after
func GetLegacyPrefixFilePathByHash(item *photoslibrary.MediaItem, options *Options) string {
	hasher := md5.New()
	hasher.Write([]byte(item.Id))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(options.BackupFolder, hash[:4], hash[4:8], hash[8:])
}

// GetLegacyPrefixFilePath Build a file path based on legacy naming convention
// of using Media Item ID, file extension will need to be appened after
func GetLegacyPrefixFilePath(item *photoslibrary.MediaItem, options *Options) string {
	//Legacy file names
	fileName, err := GetLegacyPrefixFilePathByTime(item, options)
	if err != nil {
		//Must return since this provides its own folder paths
		fileName = GetLegacyPrefixFilePathByHash(item, options)
	}

	return fileName
}

// GetJSONFilePath Get the full path to the JSON file representing the MediaItem
func GetJSONFilePath(item *photoslibrary.MediaItem, options *Options) string {
	if options.UseFileName {
		//TODO item.Id could be missing
		return filepath.Join(GetFolderPath(item, options), "."+item.Id+".json")
	}

	return GetLegacyPrefixFilePath(item, options) + ".json"
}
