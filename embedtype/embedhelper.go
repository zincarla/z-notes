package embedtype

import (
	"mime"
	"path/filepath"
	"strings"
)

//EmbedType represents an embed method
type EmbedType uint64

const (
	//Unknown no embed recommended
	Unknown EmbedType = 0
	//Image embed as image
	Image EmbedType = 1
	//Video embed as video
	Video EmbedType = 2
	//Direct embed as markdown
	Direct EmbedType = 3
	//Code embed in code fence
	Code EmbedType = 4
	//Audio Embed as video/audio
	Audio EmbedType = 5
)

//GetEmbedType returns an EmbedType representing the recommended embed method
func GetEmbedType(fileName string) EmbedType {
	extension := filepath.Ext(fileName)
	suggestedMime := mime.TypeByExtension(extension)

	//Attempt to base off of common mimes
	if suggestedMime != "" && strings.Split(suggestedMime, "/")[0] == "image" {
		return Image
	}
	if suggestedMime != "" && strings.Split(suggestedMime, "/")[0] == "video" {
		return Video
	}
	if suggestedMime != "" && strings.Split(suggestedMime, "/")[0] == "audio" {
		return Audio
	}
	if suggestedMime != "" && suggestedMime == "text/plain" {
		return Direct
	}
	if suggestedMime != "" && strings.Split(suggestedMime, "/")[0] == "text" {
		return Code
	}
	//Then fallback to extensions
	if embedType, ok := getEmbedMap()[extension]; ok == true {
		return embedType
	}

	//Default with unknown
	return Unknown
}

func getEmbedMap() map[string]EmbedType {
	return map[string]EmbedType{
		".md":   Direct,
		".cs":   Code,
		".py":   Code,
		".js":   Code,
		".go":   Code,
		".ps1":  Code,
		".html": Code,
		".css":  Code,
		".php":  Code,
	}
}
