package interfaces

import "time"

//APITokenInformation contains information for a APIToken
type APITokenInformation struct {
	OwnerID        uint64
	ID             uint64
	FriendlyID     string
	CreationTime   time.Time
	ExpirationTime time.Time
	Expires        bool
}
