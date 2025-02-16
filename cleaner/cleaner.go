package cleaner

import (
	"log"

	"github.com/dtylman/gitmoo-goog/itemlibrary"
	"github.com/dtylman/gitmoo-goog/photoslibraryextended"
)

// gather all the items (either provided by downloader or separately)

// search through downloaded library to remove items

// 1. Gather any known items (from downloader), this could be an empty set
// 2. Iterate through all items in directory
// 3. Hit API and check if item does not exist
// 4. If item does not exist, delete it by adding it to a queue
// 5. Maybe prompt user to confirm deletion?

type Cleaner struct {
	Options *Options
	ids     *map[string]*itemlibrary.Item
}

func NewCleaner() *Cleaner {
	cleaner := new(Cleaner)
	cleaner.ids = nil
	cleaner.Options = new(Options)
	cleaner.Options.WritesEnabled = false

	return cleaner
}

func (c *Cleaner) Initialize() error {
	ids := make(map[string]*itemlibrary.Item)
	c.ids = &ids

	err := itemlibrary.WalkLibrary(c.Options.ToItemLibraryOptions(), func(item *itemlibrary.Item) error {
		log.Printf("Adding item %v", item.Id)
		c.Add(item)
		return nil
	})

	return err
}

func (c *Cleaner) Add(item *itemlibrary.Item) {
	(*c.ids)[item.Id] = item
}

func (c *Cleaner) Remove(item *itemlibrary.Item) {
	delete((*c.ids), item.Id)
}

func (c *Cleaner) CleanAll(svc photoslibraryextended.MediaItemsServiceGet) error {
	// Iterate and confirm each entry is missing before deleting

	for id, item := range *c.ids {
		mediaItem, err := svc(id).Do()

		if mediaItem != nil {
			log.Printf("Skipping item %v", mediaItem.Id)
			continue
		}

		// TODO check the type of error

		log.Printf("Error was %v", err)

		log.Printf("Removing item %v", item.Id)
		err = item.Remove()
		if err != nil {
			return err
		}
	}

	return nil
}
