package interfaces

//PageAccessControl Is a uint64 representing access to a page
type PageAccessControl uint64

//All permissions must be a power of two and are checked using bitwise operators, if highest bit is set, then this is a deny permission
const (
	//Basic Bits
	//Read grants access to view a page, and see the titles of other pages above it
	Read PageAccessControl = 1
	//Write grants access to edit a page, or move it
	Write PageAccessControl = 1 << 1
	//Delete grants access to delete a page
	Delete PageAccessControl = 1 << 2
	//Audit grants access to view a page's change history
	Audit PageAccessControl = 1 << 3
	//Moderate grants access to edit permissions on a page
	Moderate PageAccessControl = 1 << 4

	//Control Bits
	//Inherits explicitly sets the access to inherit down
	Inherits PageAccessControl = 1 << 62
	//Deny explicitly denies the access set by the other bits
	Deny PageAccessControl = 1 << 63

	//Composites
	//Full grants all access to a page
	Full PageAccessControl = Read | Write | Delete | Audit | Moderate
)

//HasAccess checks the current access set to see if it matches the provided access
func (Access PageAccessControl) HasAccess(CheckAccess PageAccessControl) bool {
	return ((Access & CheckAccess) == CheckAccess) && (CheckAccess&Deny != Deny)
}

//String returns an easy to read string describing the permission
func (Access PageAccessControl) String() string {
	toReturn := ""
	if Access&Deny == Deny {
		toReturn += "Explicitly Denied "
	} else {
		toReturn += "Allowed "
	}

	somethingSet := false
	if Access&Read == Read {
		toReturn += "Read, "
		somethingSet = true
	}
	if Access&Write == Write {
		toReturn += "Write, "
		somethingSet = true
	}
	if Access&Delete == Delete {
		toReturn += "Delete "
		somethingSet = true
	}
	if Access&Audit == Audit {
		toReturn += "Audit, "
		somethingSet = true
	}
	if Access&Moderate == Moderate {
		toReturn += "Moderate, "
		somethingSet = true
	}
	if !somethingSet {
		toReturn += "nothing "
	}

	toReturn += "to this note"
	if Access&Inherits == Inherits {
		toReturn += " and all subnotes"
	}

	return toReturn
}

//UserPageAccess represents a single access entry for a user on a page
type UserPageAccess struct {
	ID     uint64
	User   UserInformation
	PageID uint64
	Access PageAccessControl
}
