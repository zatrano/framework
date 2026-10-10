package doctor

// RecommendedGoMinimum is the oldest Go release that contains the current
// stdlib security fixes. Review it on every framework release. The release
// workflow compares it with the newest stable patch on the same minor from
// https://go.dev/dl/?mode=json and warns when this constant is more than
// two patches behind.
const RecommendedGoMinimum = "go1.26.9"
