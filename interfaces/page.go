package interfaces

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
	//Content of page in markdown
	Content string
	//Children slice of this Page's Children
	Children []Page
}
