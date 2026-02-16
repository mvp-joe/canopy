package scripts

import "embed"

//go:embed extract/*.risor resolve/*.risor lib
var FS embed.FS
