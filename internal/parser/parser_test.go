package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseFilterTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  utiQueryFilter
	}{
		{
			name:  "basic",
			input: "`uti:\"qf-kind=range;qf-key=createdAt\"",
			want: utiQueryFilter{
				_kindList: []queryFilterKind{_qfKindRange},
				_key:      "createdAt",
			},
		},
		{
			name:  "multiple kinds",
			input: "`uti:\"qf-kind=range,exact;qf-key=createdAt\"",
			want: utiQueryFilter{
				_kindList: []queryFilterKind{_qfKindRange, _qfKindExact},
				_key:      "createdAt",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := parseFilterTag(test.input)

			// Assert
			require.Equal(t, test.want, got)
		})
	}
}

func Test_namedReplace(t *testing.T) {
	t.Parallel()

	fields := map[string]string{
		"$key":   "bob",
		"$range": "1,2,3,4,5",
	}

	// Act
	got := namedReplace("my key is: $key, range is: $range", fields)

	// Assert
	require.Equal(t, `my key is: bob, range is: 1,2,3,4,5`, got)
}

func Test_generateConstKeys(t *testing.T) {
	t.Parallel()

	pfCreatedAt := parserField{_name: "_MySuperCreatedAtExact", _kind: _qfKindExact}
	skusMultiValue := parserField{_name: "_SKUsMultiValue", _kind: _qfKindMultiValue}
	skusLte := parserField{_name: "_SKUsLte", _kind: _qfKindRange, isRangeLte: true}
	skusGte := parserField{_name: "_SKUsGte", _kind: _qfKindRange, isRangeGte: true}

	// Act
	got, gotConstMap := generateConstKeys([]_field{{
		_originalName: "MySuperCreatedAt",
		_goType:       "time.Time",
		_qf: utiQueryFilter{
			_kindList: []queryFilterKind{_qfKindExact},
			_key:      "createdAt",
		},
	}, {
		_originalName: "SKUs",
		_goType:       "[]uint64",
		_qf: utiQueryFilter{
			_kindList: []queryFilterKind{_qfKindMultiValue, _qfKindRange},
			_key:      "skus",
		},
	}}, map[string][]parserField{
		"MySuperCreatedAt": {pfCreatedAt},
		"SKUs": {
			skusMultiValue,
			skusLte,
			skusGte,
		},
	})

	// Assert
	want := []string{
		`const _MySuperCreatedAtKey = "createdAt"`,
		`const _SKUsKey = "skus"`,
		`const _SKUsKey_lte = "skus-to"`,
		`const _SKUsKey_gte = "skus-from"`,
	}
	require.Equal(t, strings.Join(want, "\n"), got)
	require.Equal(t, map[parserField]string{
		pfCreatedAt:    "_MySuperCreatedAtKey",
		skusGte:        "_SKUsKey_gte",
		skusLte:        "_SKUsKey_lte",
		skusMultiValue: "_SKUsKey",
	}, gotConstMap)
}

func Test_generateFilterStructDef(t *testing.T) {
	t.Parallel()

	// Act
	got, gotFieldMap := generateFilterStructDef("Product", []_field{
		{
			_originalName: "name",
			_goType:       "string",
			_qf: utiQueryFilter{
				_kindList: []queryFilterKind{_qfKindExact},
				_key:      "name",
			},
		},
		{
			_originalName: "price",
			_goType:       "float64",
			_qf: utiQueryFilter{
				_kindList: []queryFilterKind{_qfKindRange, _qfKindMultiValue},
				_key:      "price",
			},
		},
	})

	// Assert
	want := []string{
		`type Product struct{`,
		`_nameExact *string`,
		`_priceLte *float64`,
		`_priceGte *float64`,
		`_priceMultiValue *[]float64`,
		`}`,
	}
	require.Equal(t, strings.Join(want, "\n"), got)
	require.Equal(t, map[string][]parserField{
		"name": {{_name: "_nameExact", _kind: _qfKindExact}},
		"price": {
			{_name: "_priceLte", _kind: _qfKindRange, isRangeLte: true},
			{_name: "_priceGte", _kind: _qfKindRange, isRangeGte: true},
			{_name: "_priceMultiValue", _kind: _qfKindMultiValue},
		},
	}, gotFieldMap)
}

/*func Test_generateQueryValueGetter(t *testing.T) {
	t.Parallel()

	// Act
	got := generateQueryValueParser("pf", "_nameExact", "_nameKey", "parseStrVal")

	// Assert
	require.Equal(t, `
if q.Has(_nameKey) {
	_nameKeyRaw:=q.Get(_nameKey)
	_nameKeyParsed:=parseStrVal(_nameKeyRaw)
	pf._nameExact=_nameKeyParsed
}
`, got)
}*/

func Test_generateFieldGetterFunc(t *testing.T) {
	t.Parallel()

	// Act
	got := generateFieldGetterFunc("pf", "_filterProduct", "Name", "_nameExact", "Exact", "string")

	require.Equal(t, `
func (pf *_filterProduct) GetNameExact() string {
	if pf._nameExact != nil {
		return *pf._nameExact
	}
	return *new(string)
}
`, got)
}

func TestGenerateCode(t *testing.T) {
	t.Parallel()

	fields := []_field{{
		_originalName: "name",
		_goType:       "string",
		_qf: utiQueryFilter{
			_kindList: []queryFilterKind{_qfKindExact},
			_key:      "name",
		},
	}, {
		_originalName: "price",
		_goType:       "float64",
		_qf: utiQueryFilter{
			_kindList: []queryFilterKind{_qfKindRange, _qfKindMultiValue},
			_key:      "price",
		},
	}}

	// Act
	got, err := GenerateCode("my_package", "Product", fields)

	// Assert
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(`
// CODE IS GENERATED AUTOMATICALLY, DO NOT EDIT!!!
package my_package
const _nameKey = "name"
const _priceKey = "price"
const _priceKeyGte = "price-from"
const _priceKeyLte = "price-to"
type _filterProduct struct {
	_nameExact *string
	_priceLte  *float64
	_priceGte  *float64
	_priceMultiValue []float64
}
`), got)
}
