package interfaces

import "time"

//Page represent a page entry in the database
type Page struct {
	//Name page name
	Name string
	//ID Page id in database
	ID uint64
	//PrevID parent page's ID. Root if 0
	PrevID uint64
	//OwnerID owning user's ID
	OwnerID uint64
	//RevisionID if this is a page revision, this will be a non-0 id of the revision entry
	RevisionID uint64
	//RevisionTime time this revision was saved
	RevisionTime time.Time
	//Content of page in markdown
	Content string
	//Children slice of this Page's Children
	Children []Page
}
