// Code generated by "stringer -type ItemStatus"; DO NOT EDIT.

package parser

import "strconv"

const _ItemStatus_name = "StUnknownStPendingStStartedStDoneStCancelStArchive"

var _ItemStatus_index = [...]uint8{0, 9, 18, 27, 33, 41, 50}

func (i ItemStatus) String() string {
	if i < 0 || i >= ItemStatus(len(_ItemStatus_index)-1) {
		return "ItemStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ItemStatus_name[_ItemStatus_index[i]:_ItemStatus_index[i+1]]
}
