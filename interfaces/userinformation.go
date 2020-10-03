package interfaces

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/mr-tron/base58"
)

//UserInformation contains information for a user
type UserInformation struct {
	DBID          uint64
	OIDCSubject   string
	OIDCIssuer    string
	OIDCIssueTime time.Time
	Name          string
	EMail         string
	EMailVerified bool
	CreationTime  time.Time
	Disabled      bool
	IP            string
}

//GetCompositeID This returns a string of identifiers for the user
func (ui UserInformation) GetCompositeID() string {
	toReturn := ""
	if ui.DBID != 0 {
		toReturn += strconv.FormatUint(ui.DBID, 10) + " "
	} else {
		toReturn += "- "
	}
	if ui.OIDCIssuer != "" && ui.OIDCSubject != "" {
		toReturn += ui.OIDCIssuer + "/" + ui.OIDCSubject + " "
	} else {
		toReturn += "- "
	}
	if ui.IP != "" {
		toReturn += ui.IP + " "
	} else {
		toReturn += "- "
	}
	return toReturn[:len(toReturn)-1]
}

//GetDiscriminateName This returns ther user's DiscriminateName
func (ui UserInformation) GetDiscriminateName() string {
	return ui.Name + "#" + ui.GetIDDiscriminate()
}

//GetIDDiscriminate This returns ther user's DBID in base58
func (ui UserInformation) GetIDDiscriminate() string {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, ui.DBID)
	cutStart := 0
	for i := 0; i < len(idBytes); i++ {
		if idBytes[cutStart] != 0 {
			break
		}
		cutStart++
	}
	if cutStart == len(idBytes) {
		return ""
	}
	return string(base58.Encode(idBytes[cutStart:len(idBytes)]))
}

//SetName Set's the user's name, and if a discriminate is provided, the ID
func (ui *UserInformation) SetName(name string) error {
	if len(name) > 267 {
		return errors.New("name is too long to be used")
	}
	if name == "" {
		return errors.New("name not provided")
	}

	splitResults := strings.Split(name, "#")
	if len(splitResults) == 1 {
		//Not a discriminate name
		ui.Name = splitResults[0]
	} else if len(splitResults) > 2 {
		return errors.New("name has too many '#' symbols")
	} else {
		//Nearly there, verify the discriminate
		encodedID := splitResults[1]
		idArray, err := base58.Decode(encodedID)
		if err != nil {
			return err
		}
		if len(idArray) < 8 {
			newArray := make([]byte, (8 - len(idArray)))
			idArray = append(newArray, idArray...)
		}
		ui.DBID = binary.BigEndian.Uint64(idArray)
		ui.Name = splitResults[0]
	}
	return nil
}
