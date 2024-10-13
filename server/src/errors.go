package videoconf

import "errors"

var (
	ErrWrongMarshalSize        = errors.New("RTCP : Wrong Marshal Size")
	ErrInvalidTotalLost        = errors.New("RTCP : Invalid total lost count")
	ErrInvalidSizeorStartIndex = errors.New("RTCP : Invalid Size or Start Index")
)
