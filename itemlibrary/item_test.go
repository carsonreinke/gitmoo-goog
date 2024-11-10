package itemlibrary

import (
	"errors"
	"mime"
	"os"
	"path/filepath"
	"testing"

	photoslibrary "github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"
)

func TestMarshallJSON(t *testing.T) {
	item := new(Item)
	item.UsedFileName = "test.jpg"

	bytes, err := item.marshalJSON()
	if err != nil {
		t.Fatalf("itemlibrary.item.MarshallJSON: %v", err)
	}
	have := string(bytes)
	want := "{\"UsedFileName\":\"test.jpg\"}"

	if have != want {
		t.Errorf("itemlibrary.item.MarshallJSON = %v; want %v", have, want)
	}
}

func TestJSONExists(t *testing.T) {
	t.Run("Existing", func(t *testing.T) {
		options := createOptions(t)
		mediaItem := createMediaItem()

		jsonPath := GetJSONFilePath(mediaItem, options)
		err := os.MkdirAll(filepath.Dir(jsonPath), 0700)
		if err != nil {
			t.Fatalf("%v", err)
		}
		err = os.WriteFile(jsonPath, []byte{}, 0644)
		if err != nil {
			t.Fatalf("%v", err)
		}

		have, err := JSONExists(mediaItem, options)
		want := true
		if err != nil {
			t.Fatalf("itemlibrary.JSONExists: %v", err)
		}
		if have != want {
			t.Errorf("itemlibrary.JSONExists = %v; want %v", have, want)
		}
	})

	t.Run("Not Existing", func(t *testing.T) {
		options := createOptions(t)
		mediaItem := createMediaItem()

		have, err := JSONExists(mediaItem, options)
		want := false
		if err != nil {
			t.Fatalf("itemlibrary.JSONExists: %v", err)
		}
		if have != want {
			t.Errorf("itemlibrary.JSONExists = %v; want %v", have, want)
		}
	})
}

func TestLoadFromJson(t *testing.T) {
	t.Run("Existing", func(t *testing.T) {
		id := "12345678901234567890"
		options := createOptions(t)
		mediaItem := createMediaItem()

		jsonPath := GetJSONFilePath(mediaItem, options)
		os.MkdirAll(filepath.Dir(jsonPath), 0755)
		err := os.WriteFile(jsonPath, []byte("{\"id\": \""+id+"\", \"UsedFileName\":\"test.jpg\"}"), 0644)
		if err != nil {
			t.Fatalf("itemlibrary.LoadFromJSON(): %v", err)
		}

		item, err := LoadFromJSON(mediaItem, options)
		if err != nil {
			t.Fatalf("itemlibrary.LoadFromJSON(): %v", err)
		}
		if item == nil {
			t.Fatalf("itemlibrary.LoadFromJSON() = nil; want %v", item)
		}

		if item.Id != id {
			t.Fatalf("itemlibrary.LoadFromJSON() = %v; want %v", item.Id, "test")
		}
		if item.UsedFileName != "test.jpg" {
			t.Fatalf("itemlibrary.LoadFromJSON() = %v; want %v", item.UsedFileName, "test.jpg")
		}
	})

	t.Run("Not Existing", func(t *testing.T) {
		item, err := LoadFromJSON(createMediaItem(), createOptions(t))
		if err != nil {
			t.Fatalf("itemlibrary.LoadFromJSON(): %v", err)
		}
		if item != nil {
			t.Fatalf("itemlibrary.LoadFromJSON() = %v; want %v", item, nil)
		}
	})
}

func TestCreateBaseFileName(t *testing.T) {
	t.Run("Legacy Hash", func(t *testing.T) {
		options := new(Options)

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Filename = "test.jpg"

		have := item.createBaseFileName(0)
		want := "8f00b204e9800998ecf8427e"

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time", func(t *testing.T) {
		options := new(Options)

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Id = "12345678901234567890"
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := item.createBaseFileName(0)
		want := "13_34567890"

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Legacy Time With Mime Type", func(t *testing.T) {
		options := new(Options)

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Id = "12345678901234567890"
		item.MimeType = "image/jpeg"
		item.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"

		have := item.createBaseFileName(0)
		// Extensions can vary by system and will just defer to that for the correct extension
		ext, _ := mime.ExtensionsByType(item.MimeType)
		want := "13_34567890" + ext[0]

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name", func(t *testing.T) {
		options := new(Options)
		options.UseFileName = true

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Filename = "test.jpg"

		have := item.createBaseFileName(0)
		want := item.Filename

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Use File Name Uses UsedFileName", func(t *testing.T) {
		options := new(Options)
		options.UseFileName = true

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Filename = "test.jpg"
		item.UsedFileName = "test (1).jpg"

		have := item.createBaseFileName(0)
		want := item.UsedFileName

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		options := new(Options)
		options.UseFileName = true

		mediaItem := new(photoslibrary.MediaItem)
		mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
		item := NewItem(mediaItem, options)
		item.Filename = "test.jpg"

		have := item.createBaseFileName(1)
		want := "test (1).jpg"

		if have != want {
			t.Errorf("itemlibrary.item.createBaseFileName() = %v; want %v", have, want)
		}
	})
}

func TestFindNonConflictingItem(t *testing.T) {
	t.Run("Not Conflicting", func(t *testing.T) {
		options := createOptions(t)
		mediaItem := createMediaItem()

		item, err := FindNonConflictingItem(mediaItem, options)
		if err != nil {
			t.Fatalf("itemlibrary.FindNonConflictingItem: %v", err)
		}

		have := item.UsedFileName
		want := item.Filename
		if have != want {
			t.Errorf("itemlibrary.FindNonConflictingItem() = %v; want %v", have, want)
		}
	})

	t.Run("Conflicting", func(t *testing.T) {
		options := createOptions(t)

		item := createItem(options)
		item.Id = "09876543210987654321"
		item.UsedFileName = item.createBaseFileName(0)
		touchFile(t, item.GetImageFilePath())

		item, err := FindNonConflictingItem(createMediaItem(), options)
		if err != nil {
			t.Fatalf("itemlibrary.FindNonConflictingItem: %v", err)
		}

		have := item.UsedFileName
		want := "test (1).jpg"
		if have != want {
			t.Errorf("itemlibrary.FindNonConflictingItem() = %v; want %v", have, want)
		}
	})
}

func TestCreateJSON(t *testing.T) {
	t.Run("Existing", func(t *testing.T) {
		options := createOptions(t)
		item := createItem(options)

		filePath := GetJSONFilePath(&item.MediaItem, options)
		touchFile(t, filePath)

		err := item.CreateJSON()
		if err == nil {
			t.Errorf("itemlibrary.item.CreateJSON() = %v; want !nil", err)
		}
	})

	t.Run("Not Existing", func(t *testing.T) {
		options := createOptions(t)
		item := createItem(options)

		filePath := GetJSONFilePath(&item.MediaItem, options)

		err := item.CreateJSON()
		if err != nil {
			t.Fatalf("itemlibrary.item.CreateJSON: %v", err)
		}

		_, err = os.Stat(filePath)
		have := errors.Is(err, os.ErrNotExist)
		want := false
		if have != want {
			t.Errorf("itemlibrary.item.CreateJSON() = %v; want %v", have, want)
		}
	})
}

func TestImageFileExists(t *testing.T) {
	t.Run("Folder Not Existing", func(t *testing.T) {
		options := createOptions(t)
		item := createItem(options)

		have, err := item.ImageFileExists()
		want := false
		if err != nil {
			t.Fatalf("item.ImageFileExists: %v", err)
		}

		if have != want {
			t.Errorf("itemlibrary.item.ImageFileExists() = %v; want %v", have, want)
		}
	})

	t.Run("File Not Existing", func(t *testing.T) {
		options := createOptions(t)
		item := createItem(options)
		err := os.MkdirAll(filepath.Dir(item.GetImageFilePath()), 0755)
		if err != nil {
			t.Fatalf("item.ImageFileExists: %v", err)
		}

		have, err := item.ImageFileExists()
		want := false
		if err != nil {
			t.Fatalf("item.ImageFileExists: %v", err)
		}

		if have != want {
			t.Errorf("itemlibrary.item.ImageFileExists() = %v; want %v", have, want)
		}
	})

	t.Run("File Existing", func(t *testing.T) {
		options := createOptions(t)
		item := createItem(options)
		touchFile(t, item.GetImageFilePath())

		have, err := item.ImageFileExists()
		want := true
		if err != nil {
			t.Fatalf("item.ImageFileExists: %v", err)
		}

		if have != want {
			t.Errorf("itemlibrary.item.ImageFileExists() = %v; want %v", have, want)
		}
	})
}

func createOptions(t *testing.T) *Options {
	options := new(Options)
	options.UseFileName = true
	options.BackupFolder = t.TempDir()
	options.FolderFormat = filepath.Join("2006", "January")

	return options
}

func createMediaItem() *photoslibrary.MediaItem {
	mediaItem := new(photoslibrary.MediaItem)
	mediaItem.Id = "12345678901234567890"
	mediaItem.MediaMetadata = new(photoslibrary.MediaMetadata)
	mediaItem.MediaMetadata.CreationTime = "2019-10-13T17:33:43Z"
	mediaItem.Filename = "test.jpg"

	return mediaItem
}

func createItem(options *Options) *Item {
	mediaItem := createMediaItem()

	item := NewItem(mediaItem, options)

	return item
}

func touchFile(t *testing.T, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("%v", err)
	}
	err = os.WriteFile(path, []byte{}, 0644)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
