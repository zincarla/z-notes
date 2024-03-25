package interfaces

import (
	"strconv"
	"time"
)

//APITokenInformation contains information for a APIToken
type APITokenInformation struct {
	OwnerID        uint64
	ID             uint64
	FriendlyID     string
	CreationTime   time.Time
	ExpirationTime time.Time
	Expires        bool
}

//GetCompositeID This returns a string of identifiers for the token
func (ti APITokenInformation) GetCompositeID() string {
	toReturn := ""
	if ti.ID != 0 {
		toReturn += strconv.FormatUint(ti.ID, 10) + " "
	} else {
		toReturn += "- "
	}
	if ti.OwnerID != 0 {
		toReturn += strconv.FormatUint(ti.OwnerID, 10) + " "
	} else {
		toReturn += "- "
	}
	return toReturn[:len(toReturn)-1]
}
