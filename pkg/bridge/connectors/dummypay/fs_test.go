package dummypay

import "github.com/spf13/afero"

func newTestFS() fs {
	fs := newFS()
	fs = afero.NewMemMapFs()

	return fs
}
