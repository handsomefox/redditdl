package params

type RequiredOrientation byte

const (
	_ RequiredOrientation = iota
	RequiredOrientationLandscape
	RequiredOrientationPortrait
	RequiredOrientationAny
)

func OrientationFromString(s string) RequiredOrientation {
	switch s {
	case "l":
		return RequiredOrientationLandscape
	case "p":
		return RequiredOrientationPortrait
	default:
		return RequiredOrientationAny
	}
}

type RequiredContentType byte

const (
	_ RequiredContentType = iota
	RequiredContentTypeImages
	RequiredContentTypeVideos
	RequiredContentTypeAny
)

func RequiredContentTypeFromString(s string) RequiredContentType {
	switch s {
	case "image":
		return RequiredContentTypeImages
	case "video":
		return RequiredContentTypeVideos
	case "any":
		return RequiredContentTypeAny
	default:
		return RequiredContentTypeAny
	}
}

type CLIParameters struct {
	Sort             string
	Timeframe        string
	Directory        string
	Subreddits       []string
	MediaMinWidth    int
	MediaMinHeight   int
	MediaCount       int64
	MediaOrientation RequiredOrientation
	ContentType      RequiredContentType
	ShowProgress     bool
	VerboseLogging   bool
}
