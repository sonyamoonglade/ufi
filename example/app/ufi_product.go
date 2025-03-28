// CODE IS GENERATED AUTOMATICALLY, DO NOT EDIT!!!
package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const _SKUKey = "skus"
const _NameKey = "name"
const _CreatedAtKey_lte = "createdAt-to"
const _CreatedAtKey_gte = "createdAt-from"
const _CreatedAtKey = "createdAt"
const _AgeKey_lte = "age-to"
const _AgeKey_gte = "age-from"

type _ProductFilter struct {
	_SKUMultiValue  *[]uint64
	_NameExact      *string
	_CreatedAtLte   *time.Time
	_CreatedAtGte   *time.Time
	_CreatedAtExact *time.Time
	_AgeLte         *uint
	_AgeGte         *uint
}

func ParseFilters(input string) (*_ProductFilter, error) {
	inpAsUri, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url: %w", err)
	}
	q := inpAsUri.Query()
	res := new(_ProductFilter)

	if q.Has(_SKUKey) {
		_SKUKeyRaw := q.Get(_SKUKey)
		_SKUKeyParsed := gsliceuintparse[uint64](_SKUKeyRaw)
		res._SKUMultiValue = &_SKUKeyParsed
	}

	if q.Has(_NameKey) {
		_NameKeyRaw := q.Get(_NameKey)
		_NameKeyParsed := vstrparse[string](_NameKeyRaw)
		res._NameExact = &_NameKeyParsed
	}

	if q.Has(_CreatedAtKey_lte) {
		_CreatedAtKey_lteRaw := q.Get(_CreatedAtKey_lte)
		_CreatedAtKey_lteParsed := vtimeparse[time.Time](_CreatedAtKey_lteRaw)
		res._CreatedAtLte = &_CreatedAtKey_lteParsed
	}

	if q.Has(_CreatedAtKey_gte) {
		_CreatedAtKey_gteRaw := q.Get(_CreatedAtKey_gte)
		_CreatedAtKey_gteParsed := vtimeparse[time.Time](_CreatedAtKey_gteRaw)
		res._CreatedAtGte = &_CreatedAtKey_gteParsed
	}

	if q.Has(_CreatedAtKey) {
		_CreatedAtKeyRaw := q.Get(_CreatedAtKey)
		_CreatedAtKeyParsed := vtimeparse[time.Time](_CreatedAtKeyRaw)
		res._CreatedAtExact = &_CreatedAtKeyParsed
	}

	if q.Has(_AgeKey_lte) {
		_AgeKey_lteRaw := q.Get(_AgeKey_lte)
		_AgeKey_lteParsed := guintparse[uint](_AgeKey_lteRaw)
		res._AgeLte = &_AgeKey_lteParsed
	}

	if q.Has(_AgeKey_gte) {
		_AgeKey_gteRaw := q.Get(_AgeKey_gte)
		_AgeKey_gteParsed := guintparse[uint](_AgeKey_gteRaw)
		res._AgeGte = &_AgeKey_gteParsed
	}

	return res, nil
}

func (_Pr *_ProductFilter) GetSKUArray() []uint64 {
	if _Pr._SKUMultiValue != nil {
		return *_Pr._SKUMultiValue
	}
	return *new([]uint64)
}

func (_Pr *_ProductFilter) GetNameExact() string {
	if _Pr._NameExact != nil {
		return *_Pr._NameExact
	}
	return *new(string)
}

func (_Pr *_ProductFilter) GetCreatedAtLte() time.Time {
	if _Pr._CreatedAtLte != nil {
		return *_Pr._CreatedAtLte
	}
	return *new(time.Time)
}

func (_Pr *_ProductFilter) GetCreatedAtGte() time.Time {
	if _Pr._CreatedAtGte != nil {
		return *_Pr._CreatedAtGte
	}
	return *new(time.Time)
}

func (_Pr *_ProductFilter) GetCreatedAtExact() time.Time {
	if _Pr._CreatedAtExact != nil {
		return *_Pr._CreatedAtExact
	}
	return *new(time.Time)
}

func (_Pr *_ProductFilter) GetAgeLte() uint {
	if _Pr._AgeLte != nil {
		return *_Pr._AgeLte
	}
	return *new(uint)
}

func (_Pr *_ProductFilter) GetAgeGte() uint {
	if _Pr._AgeGte != nil {
		return *_Pr._AgeGte
	}
	return *new(uint)
}

func guintparse[I uint | uint32 | uint64](inp string) I {
	v, err := strconv.ParseUint(inp, 10, 64)
	if err != nil {
		return *new(I)
	}
	return I(v)
}

func gsliceuintparse[I uint | uint32 | uint64](inp string) []I {
	splitted := strings.Split(inp, ",")
	result := make([]I, 0, len(splitted))
	for _, v := range splitted {
		result = append(result, guintparse[I](v))
	}
	return result
}
func vstrparse[T any](inp string) string { return inp }

func vtimeparse[T any](inp string) time.Time {
	v, err := time.Parse(time.RFC3339, inp)
	if err != nil {
		return time.Time{}
	}
	return v
}
