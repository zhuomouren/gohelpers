package goassets

var assetTmpl = `
package {{.PackageName}}

import (
	"time"

	"github.com/zhuomouren/gohelpers/goassets"
)

var _goassets_assets = []goassets.Asset{
	{{range $asset := .Assets}}
	goassets.Asset{
		Path:"{{$asset.Path}}",
		ModTime: time.Unix({{$asset.ModTime.Unix}}, {{$asset.ModTime.UnixNano}}),
		Data: ` + "`{{output $asset.Data}}`" + `,
	},
	{{end}}
}

func init() {
	goassets.Assets.SetAssets(_goassets_assets)
}
`
