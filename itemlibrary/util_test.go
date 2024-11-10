package itemlibrary

import (
	"path/filepath"
	"testing"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

func TestGetJSONFilePath(t *testing.T) {
	t.Run("Legacy Hash", func(t *testing.T) {
		options := new(Options)
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)

		have := GetJSONFilePath(item, options)
		want := filepath.Join(options.BackupFolder, "fd85", "e62d", "9beb45428771ec688418b271.json")

		if have != want {
			t.Errorf("GetJSONFilePath() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time", func(t *testing.T) {
		options := new(Options)
		options.FolderFormat = filepath.Join("2006", "January")
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := GetJSONFilePath(item, options)
		want := filepath.Join(options.BackupFolder, "2019", "October", "13_34567890.json")

		if have != want {
			t.Errorf("GetJSONFilePath() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name", func(t *testing.T) {
		options := new(Options)
		options.UseFileName = true
		options.FolderFormat = filepath.Join("2006", "January")
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.Id = "12345678901234567890"
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := GetJSONFilePath(item, options)
		want := filepath.Join(options.BackupFolder, "2019", "October", ".12345678901234567890.json")

		if have != want {
			t.Errorf("GetJSONFilePath() = %v; want %v", have, want)
		}
	})
}

func TestGetFolderPath(t *testing.T) {
	t.Run("Missing", func(t *testing.T) {
		options := new(Options)
		options.FolderFormat = filepath.Join("2006", "January")
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)

		have := GetFolderPath(item, options)
		want := filepath.Join(options.BackupFolder, "1970", "January")
		if have != want {
			t.Errorf("GetFolderPath() = %v; want %v", have, want)
		}
	})

	t.Run("Date", func(t *testing.T) {
		options := new(Options)
		options.FolderFormat = filepath.Join("2006", "January")
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := GetFolderPath(item, options)
		want := filepath.Join(options.BackupFolder, "2019", "October")
		if have != want {
			t.Errorf("GetFolderPath() = %v; want %v", have, want)
		}
	})

	t.Run("Format", func(t *testing.T) {
		options := new(Options)
		options.FolderFormat = filepath.Join("2006", "01")
		options.BackupFolder = t.TempDir()

		item := new(photoslibrary.MediaItem)
		item.MediaMetadata = new(photoslibrary.MediaMetadata)
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := GetFolderPath(item, options)
		want := filepath.Join(options.BackupFolder, "2019", "10")
		if have != want {
			t.Errorf("GetFolderPath() = %v; want %v", have, want)
		}
	})
}
