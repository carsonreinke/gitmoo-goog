package itemlibrary

import (
	"os"
	"path/filepath"
)

type Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
	//FolderFormat time format used to format folder structure
	FolderFormat string
	//UseFileName use file name when uploaded to Google Photos
	UseFileName bool
}

func NewOptions() *Options {
	options := new(Options)
	options.BackupFolder, _ = os.Getwd()
	options.FolderFormat = filepath.Join("2006", "January")
	options.UseFileName = false

	return options
}
