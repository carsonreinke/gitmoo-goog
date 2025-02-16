package cleaner

import "github.com/dtylman/gitmoo-goog/itemlibrary"

type Options struct {
	//BackupFolderis the backup folder
	BackupFolder string
	//FolderFormat time format used to format folder structure
	FolderFormat string
	//UseFileName use file name when uploaded to Google Photos
	UseFileName bool
	//WritesEnabled TODO
	WritesEnabled bool
}

func (o *Options) ToItemLibraryOptions() *itemlibrary.Options {
	options := new(itemlibrary.Options)
	options.BackupFolder = o.BackupFolder
	options.FolderFormat = o.FolderFormat
	options.UseFileName = o.UseFileName
	return options
}
