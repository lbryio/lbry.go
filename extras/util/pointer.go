// mostly copied from github.com/AlekSi/pointer
// Provides helpers to get pointers to values of build-in types.

package util

import (
	"time"

	"github.com/lbryio/lbry.go/v2/extras/null"
)

func PtrToBool(b bool) *bool                   { return &b }
func PtrToByte(b byte) *byte                   { return &b }
func PtrToComplex128(c complex128) *complex128 { return &c }
func PtrToComplex64(c complex64) *complex64    { return &c }
func PtrToError(e error) *error                { return &e }
func PtrToFloat32(f float32) *float32          { return &f }
func PtrToFloat64(f float64) *float64          { return &f }
func PtrToInt(i int) *int                      { return &i }
func PtrToInt16(i int16) *int16                { return &i }
func PtrToInt32(i int32) *int32                { return &i }
func PtrToInt64(i int64) *int64                { return &i }
func PtrToInt8(i int8) *int8                   { return &i }
func PtrToRune(r rune) *rune                   { return &r }
func PtrToString(s string) *string             { return &s }
func PtrToTime(t time.Time) *time.Time         { return &t }
func PtrToUint(u uint) *uint                   { return &u }
func PtrToUint16(u uint16) *uint16             { return &u }
func PtrToUint32(u uint32) *uint32             { return &u }
func PtrToUint64(u uint64) *uint64             { return &u }
func PtrToUint8(u uint8) *uint8                { return &u }
func PtrToUintptr(u uintptr) *uintptr          { return &u }

func PtrToNullString(s string) *null.String { n := null.StringFrom(s); return &n }
func PtrToNullUint64(u uint64) *null.Uint64 { n := null.Uint64From(u); return &n }

func StrFromPtr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func StrFromNull(str null.String) string {
	if !str.Valid {
		return ""
	}
	return str.String
}

func NullStringFrom(s string) null.String {
	if s == "" {
		return null.String{}
	}
	return null.StringFrom(s)
}
