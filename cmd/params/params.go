package params

type RequiredOrientation byte

const (
	_ RequiredOrientation = iota
	RequiredOrientationLandscape
	RequiredOrientationPortrait
	RequiredOrientationAny
)

type RequiredContentType byte

const (
	_ RequiredContentType = iota
	RequiredContentTypeImages
	RequiredContentTypeVideos
	RequiredContentTypeAny
)

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
