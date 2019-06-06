package gohelpers

import "github.com/zhuomouren/gohelpers/goasset"

func Asset(root, packageName string) error {

	goAsset := &goasset.GoAsset{
		Root:        root,
		PackageName: packageName,
	}

	return goAsset.Build()
}
