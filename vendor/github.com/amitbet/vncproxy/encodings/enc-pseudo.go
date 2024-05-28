package encodings

import (
	"io"
	"github.com/amitbet/vncproxy/common"
)

type PseudoEncoding struct {
	Typ int32
}

func (pe *PseudoEncoding) Type() int32 {
	return pe.Typ
}
func (z *PseudoEncoding) WriteTo(w io.Writer) (n int, err error) {
	return 0, nil
}
func (pe *PseudoEncoding) Read(*common.PixelFormat, *common.Rectangle, *common.RfbReadHelper) (common.IEncoding, error) {
	return pe, nil
}
