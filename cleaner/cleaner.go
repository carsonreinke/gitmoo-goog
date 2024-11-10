package cleaner

// gather all the items (either provided by downloader or separately)

// search through downloaded library to remove items

// 1. Gather any known items (from downloader), this could be an empty set
// 2. Iterate through all items in directory
// 3. Hit API and check if item does not exist
// 4. If item does not exist, delete it by adding it to a queue
// 5. Maybe prompt user to confirm deletion?

type Cleaner struct {
	Options *Options
}

func NewCleaner() *Cleaner {
	cleaner := new(Cleaner)
	cleaner.Options.DryRun = true

	return cleaner
}
