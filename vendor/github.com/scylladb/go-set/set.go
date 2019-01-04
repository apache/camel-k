// Copyright (C) 2017 ScyllaDB
// Use of this source code is governed by a ALv2-style
// license that can be found in the LICENSE file.

package set

import (
	"github.com/scylladb/go-set/b16set"
	"github.com/scylladb/go-set/b32set"
	"github.com/scylladb/go-set/b64set"
	"github.com/scylladb/go-set/b8set"
	"github.com/scylladb/go-set/f32set"
	"github.com/scylladb/go-set/f64set"
	"github.com/scylladb/go-set/i16set"
	"github.com/scylladb/go-set/i32set"
	"github.com/scylladb/go-set/i64set"
	"github.com/scylladb/go-set/i8set"
	"github.com/scylladb/go-set/iset"
	"github.com/scylladb/go-set/strset"
	"github.com/scylladb/go-set/u16set"
	"github.com/scylladb/go-set/u32set"
	"github.com/scylladb/go-set/u64set"
	"github.com/scylladb/go-set/u8set"
	"github.com/scylladb/go-set/uset"
)

//go:generate ./gen_set.sh [8]byte b8set

// NewByte8Set is a convenience function to create a new b8set.Set
func NewByte8Set(ts ...[8]byte) *b8set.Set {
	return b8set.New(ts...)
}

// NewByte8SetWithSize is a convenience function to create a new b8set.Set.
func NewByte8SetWithSize(size int) *b8set.Set {
	return b8set.NewWithSize(size)
}

//go:generate ./gen_set.sh [16]byte b16set

// NewByte16Set is a convenience function to create a new b16set.Set
func NewByte16Set(ts ...[16]byte) *b16set.Set {
	return b16set.New(ts...)
}

// NewByte16SetWithSize is a convenience function to create a new b16set.Set.
func NewByte16SetWithSize(size int) *b16set.Set {
	return b16set.NewWithSize(size)
}

//go:generate ./gen_set.sh [32]byte b32set

// NewByte32Set is a convenience function to create a new b32set.Set
func NewByte32Set(ts ...[32]byte) *b32set.Set {
	return b32set.New(ts...)
}

// NewByte32SetWithSize is a convenience function to create a new b32set.Set.
func NewByte32SetWithSize(size int) *b32set.Set {
	return b32set.NewWithSize(size)
}

//go:generate ./gen_set.sh [64]byte b64set

// NewByte64Set is a convenience function to create a new b64set.Set
func NewByte64Set(ts ...[64]byte) *b64set.Set {
	return b64set.New(ts...)
}

// NewByte64SetWithSize is a convenience function to create a new b64set.Set.
func NewByte64SetWithSize(size int) *b64set.Set {
	return b64set.NewWithSize(size)
}

//go:generate ./gen_set.sh float32 f32set

// NewFloat32Set is a convenience function to create a new f32set.Set
func NewFloat32Set(ts ...float32) *f32set.Set {
	return f32set.New(ts...)
}

// NewFloat32SetWithSize is a convenience function to create a new f32set.Set.
func NewFloat32SetWithSize(size int) *f32set.Set {
	return f32set.NewWithSize(size)
}

//go:generate ./gen_set.sh float64 f64set

// NewFloat64Set is a convenience function to create a new f64set.Set
func NewFloat64Set(ts ...float64) *f64set.Set {
	return f64set.New(ts...)
}

// NewFloat64SetWithSize is a convenience function to create a new f64set.Set.
func NewFloat64SetWithSize(size int) *f64set.Set {
	return f64set.NewWithSize(size)
}

//go:generate ./gen_set.sh int iset

// NewIntSet is a convenience function to create a new iset.Set
func NewIntSet(ts ...int) *iset.Set {
	return iset.New(ts...)
}

// NewIntSetWithSize is a convenience function to create a new iset.Set.
func NewIntSetWithSize(size int) *iset.Set {
	return iset.NewWithSize(size)
}

//go:generate ./gen_set.sh int8 i8set

// NewInt8Set is a convenience function to create a new i8set.Set
func NewInt8Set(ts ...int8) *i8set.Set {
	return i8set.New(ts...)
}

// NewInt8SetWithSize is a convenience function to create a new i8set.Set.
func NewInt8SetWithSize(size int) *i8set.Set {
	return i8set.NewWithSize(size)
}

//go:generate ./gen_set.sh int16 i16set

// NewInt16Set is a convenience function to create a new i16set.Set
func NewInt16Set(ts ...int16) *i16set.Set {
	return i16set.New(ts...)
}

// NewInt16SetWithSize is a convenience function to create a new i16set.Set.
func NewInt16SetWithSize(size int) *i16set.Set {
	return i16set.NewWithSize(size)
}

//go:generate ./gen_set.sh int32 i32set

// NewInt32Set is a convenience function to create a new i32set.Set
func NewInt32Set(ts ...int32) *i32set.Set {
	return i32set.New(ts...)
}

// NewInt32SetWithSize is a convenience function to create a new i32set.Set.
func NewInt32SetWithSize(size int) *i32set.Set {
	return i32set.NewWithSize(size)
}

//go:generate ./gen_set.sh int64 i64set

// NewInt64Set is a convenience function to create a new i64set.Set
func NewInt64Set(ts ...int64) *i64set.Set {
	return i64set.New(ts...)
}

// NewInt64SetWithSize is a convenience function to create a new i64set.Set.
func NewInt64SetWithSize(size int) *i64set.Set {
	return i64set.NewWithSize(size)
}

//go:generate ./gen_set.sh uint uset

// NewUintSet is a convenience function to create a new uset.Set
func NewUintSet(ts ...uint) *uset.Set {
	return uset.New(ts...)
}

// NewUintSetWithSize is a convenience function to create a new uset.Set.
func NewUintSetWithSize(size int) *uset.Set {
	return uset.NewWithSize(size)
}

//go:generate ./gen_set.sh uint8 u8set

// NewUint8Set is a convenience function to create a new u8set.Set
func NewUint8Set(ts ...uint8) *u8set.Set {
	return u8set.New(ts...)
}

// NewUint8SetWithSize is a convenience function to create a new u8set.Set.
func NewUint8SetWithSize(size int) *u8set.Set {
	return u8set.NewWithSize(size)
}

//go:generate ./gen_set.sh uint16 u16set

// NewUint16Set is a convenience function to create a new u16set.Set
func NewUint16Set(ts ...uint16) *u16set.Set {
	return u16set.New(ts...)
}

// NewUint16SetWithSize is a convenience function to create a new u16set.Set.
func NewUint16SetWithSize(size int) *u16set.Set {
	return u16set.NewWithSize(size)
}

//go:generate ./gen_set.sh uint32 u32set

// NewUint32Set is a convenience function to create a new u32set.Set
func NewUint32Set(ts ...uint32) *u32set.Set {
	return u32set.New(ts...)
}

// NewUint32SetWithSize is a convenience function to create a new u32set.Set.
func NewUint32SetWithSize(size int) *u32set.Set {
	return u32set.NewWithSize(size)
}

//go:generate ./gen_set.sh uint64 u64set

// NewUint64Set is a convenience function to create a new u64set.Set
func NewUint64Set(ts ...uint64) *u64set.Set {
	return u64set.New(ts...)
}

// NewUint64SetWithSize is a convenience function to create a new u64set.Set.
func NewUint64SetWithSize(size int) *u64set.Set {
	return u64set.NewWithSize(size)
}

//go:generate ./gen_set.sh string strset

// NewStringSet is a convenience function to create a new strset.Set
func NewStringSet(ts ...string) *strset.Set {
	return strset.New(ts...)
}

// NewStringSetWithSize is a convenience function to create a new strset.Set.
func NewStringSetWithSize(size int) *strset.Set {
	return strset.NewWithSize(size)
}
