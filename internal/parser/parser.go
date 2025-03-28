package parser

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Run() error {
	var structName string
	var outputFile string
	var pkg string

	flag.StringVar(&structName, "name", "ns", "struct name to generate filter for")
	flag.StringVar(&outputFile, "out", "", "output file where generated code will be placed")
	flag.StringVar(&pkg, "pkg", "", "package name for generated filet")

	flag.Parse()

	sourceFile := os.Getenv("GOFILE")
	file, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	structSourceRows := []string{}
	consumeStruct := false
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "}" {
			break
		}

		searchFor := fmt.Sprintf("type %s struct {", structName)
		if strings.Contains(line, searchFor) {
			consumeStruct = true
			continue
		}

		if consumeStruct {
			structSourceRows = append(structSourceRows, strings.TrimSpace(line))
		}
	}

	var fields []_field
	for _, line := range structSourceRows {
		f := consumeField(line)
		parsedTag := parseFilterTag(f._tag)
		f._qf = parsedTag
		fields = append(fields, f)
	}

	code, err := GenerateCode(pkg, structName, fields)
	if err != nil {
		return fmt.Errorf("could not generate code: %w", err)
	}

	out, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}

	defer out.Close()
	if _, err := out.WriteString(code); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	if err := exec.Command("goimports", "-w", outputFile).Run(); err != nil {
		return fmt.Errorf("cannot run 'goimports' command: %w", err)
	}

	return nil
}

type _field struct {
	_originalName string
	_goType       string
	_tag          string
	_qf           utiQueryFilter
}

type _readFieldState int

const (
	_readFieldStateName _readFieldState = iota + 1
	_readFieldStateGoType
	_readFieldStateTag
)

func consumeField(s string) _field {
	f := _field{}
	state := _readFieldStateName
	for i, ch := range s {
		var nextCh byte
		if i+1 < len(s) {
			nextCh = s[i+1]
		}
		switch state {
		case _readFieldStateName:
			if ch == ' ' {
				if nextCh != ' ' {
					state++
				}
				continue
			}
			f._originalName += string(ch)
		case _readFieldStateGoType:
			if ch == ' ' {
				if nextCh != ' ' {
					state++
				}
				continue
			}
			f._goType += string(ch)
		case _readFieldStateTag:
			f._tag += string(ch)
		}
	}
	return f
}

type queryFilterKind string

const (
	_qfKindRange      = queryFilterKind("range")
	_qfKindMultiValue = queryFilterKind("multi-value")
	_qfKindExact      = queryFilterKind("exact")
)

var _qfKindMap = map[queryFilterKind]struct{}{
	_qfKindRange:      {},
	_qfKindMultiValue: {},
	_qfKindExact:      {},
}

func isValidQfKind(kind string) bool {
	_, ok := _qfKindMap[queryFilterKind(kind)]
	return ok
}

const (
	_tagQfKind = "qf-kind"
	_tagQfKey  = "qf-key"
)

type utiQueryFilter struct {
	_kindList []queryFilterKind
	_key      string
}

func parseFilterTag(s string) utiQueryFilter {
	s = strings.Trim(s, "`")
	s = strings.Trim(s, "uti:")
	s = strings.Trim(s, `"`)
	splitted := strings.Split(s, ";")
	res := utiQueryFilter{}
	for _, pair := range splitted {
		splittedPair := strings.Split(pair, "=")
		if len(splittedPair) != 2 {
			log.Printf("ignoring qf-pair: [%s]", pair)
			continue
		}

		key, value := splittedPair[0], splittedPair[1]
		if key == _tagQfKey {
			res._key = value
		}

		if key == _tagQfKind {
			valueSplitted := strings.Split(value, ",")
			for _, kind := range valueSplitted {
				if kind == "" {
					continue
				}
				if !isValidQfKind(kind) {
					log.Printf("ignoring qf-kind: [%s]", kind)
					continue
				}
				res._kindList = append(res._kindList, queryFilterKind(kind))
			}
		}
	}

	return res
}

func generateQueryValueParser(variable string, fieldName, qfKeyConstName, staticParserFuncName, fieldType string) string {
	const tmpl = `
if q.Has($key) {
	$keyRaw:=q.Get($key)
	$keyParsed:=$gotypeParser[$fieldType]($keyRaw)
	$var.$fieldName=&$keyParsed
}
`
	return namedReplace(tmpl, map[string]string{
		"$var":          variable,
		"$fieldName":    fieldName,
		"$key":          qfKeyConstName,
		"$gotypeParser": staticParserFuncName,
		"$fieldType":    strings.TrimLeft(fieldType, "[]"),
	})
}

func generateFieldGetterFunc(structRcv, structName, originalFieldName, parserFieldName, postfix, gotype string) string {
	const tmpl = `
func ($rcv *$structName) Get$origField$postfix() $gotype {
	if $rcv.$parserField != nil {
		return *$rcv.$parserField
	}
	return *new($gotype)
}
`
	return namedReplace(tmpl, map[string]string{
		"$rcv":         structRcv,
		"$gotype":      gotype,
		"$structName":  structName,
		"$origField":   originalFieldName,
		"$parserField": parserFieldName,
		"$postfix":     postfix,
	})
}

type parserField struct {
	_name      string
	_kind      queryFilterKind
	isRangeLte bool
	isRangeGte bool
}

func generateFilterStructDef(structName string, fields []_field) (string, map[string][]parserField) {
	var rows []string
	fieldMap := make(map[string][]parserField)
	const tmpl = `$fieldName $goType`
	rows = append(rows, fmt.Sprintf("type %s struct{", structName))
	for _, field := range fields {
		for _, kind := range field._qf._kindList {
			switch kind {
			case _qfKindExact:
				parserName := "_" + field._originalName + "Exact"
				fieldMap[field._originalName] = append(fieldMap[field._originalName], parserField{
					_name: parserName,
					_kind: kind,
				})
				rows = append(rows, namedReplace(tmpl, map[string]string{
					"$fieldName": parserName,
					"$goType":    "*" + field._goType,
				}))
			case _qfKindMultiValue:
				parserName := "_" + field._originalName + "MultiValue"
				fieldMap[field._originalName] = append(fieldMap[field._originalName], parserField{
					_name: parserName,
					_kind: kind,
				})
				rows = append(rows, namedReplace(tmpl, map[string]string{
					"$fieldName": parserName,
					"$goType":    "*[]" + field._goType,
				}))
			case _qfKindRange:
				parserNameLte := "_" + field._originalName + "Lte"
				fieldMap[field._originalName] = append(fieldMap[field._originalName], parserField{
					_name:      parserNameLte,
					_kind:      kind,
					isRangeLte: true,
				})
				rows = append(rows, namedReplace(tmpl, map[string]string{
					"$fieldName": parserNameLte,
					"$goType":    "*" + field._goType,
				}))

				parserNameGte := "_" + field._originalName + "Gte"
				rows = append(rows, namedReplace(tmpl, map[string]string{
					"$fieldName": parserNameGte,
					"$goType":    "*" + field._goType,
				}))
				fieldMap[field._originalName] = append(fieldMap[field._originalName], parserField{
					_name:      parserNameGte,
					isRangeGte: true,
					_kind:      kind,
				})
			}
		}
	}
	rows = append(rows, "}")
	return strings.Join(rows, "\n"), fieldMap
}

func namedReplace(s string, fields map[string]string) string {
	for fieldName, value := range fields {
		s = strings.ReplaceAll(s, fieldName, value)
	}
	return s
}

const constTmpl = `const $constName = "$key"`

type fieldToConstKeyMap map[string]map[queryFilterKind][]string

func generateConstKeys(fields []_field, structFieldMap map[string][]parserField) (string, map[parserField]string) {
	parserFieldToConst := make(map[parserField]string)
	var rows []string
	for _, field := range fields {
		for _, pf := range structFieldMap[field._originalName] {
			if pf.isRangeLte {
				lteValues := map[string]string{
					"$constName": fmt.Sprintf("_%sKey_lte", field._originalName),
					"$key":       fmt.Sprintf("%s-to", field._qf._key),
					"$kind":      strings.ReplaceAll(string(pf._kind), "-", "_"),
				}
				parserFieldToConst[pf] = lteValues["$constName"]
				rows = append(rows, namedReplace(constTmpl, lteValues))
			}
			if pf.isRangeGte {
				gteValues := map[string]string{
					"$constName": fmt.Sprintf("_%sKey_gte", field._originalName),
					"$key":       fmt.Sprintf("%s-from", field._qf._key),
					"$kind":      strings.ReplaceAll(string(pf._kind), "-", "_"),
				}
				parserFieldToConst[pf] = gteValues["$constName"]
				rows = append(rows, namedReplace(constTmpl, gteValues))
			}
			if pf._kind == _qfKindMultiValue || pf._kind == _qfKindExact {
				multiValueOrExactValues := map[string]string{
					"$constName": fmt.Sprintf("_%sKey", field._originalName),
					"$key":       fmt.Sprintf("%s", field._qf._key),
					"$kind":      strings.ReplaceAll(string(pf._kind), "-", "_"),
				}
				parserFieldToConst[pf] = multiValueOrExactValues["$constName"]
				rows = append(rows, namedReplace(constTmpl, multiValueOrExactValues))
			}
		}
	}
	return strings.Join(rows, "\n"), parserFieldToConst
}

func generateParserFunc(structName string, fields []_field, structFieldsMap map[string][]parserField, qfConstKeyMap map[parserField]string) string {
	var queryParserRows []string
	for _, field := range fields {
		parserFields, ok := structFieldsMap[field._originalName]
		if !ok {
			continue
		}
		for _, pf := range parserFields {
			goType := ternary(pf._kind == _qfKindMultiValue, "[]"+field._goType, field._goType)
			queryParserRows = append(queryParserRows, generateQueryValueParser(
				"res",
				pf._name,
				qfConstKeyMap[pf],
				goTypeParserFuncs[goType],
				goType))
		}
	}
	const parseFuncTmpl = `
func ParseFilters(input string) (*$structName, error) {
	inpAsUri, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url: %w", err)
	}
	q := inpAsUri.Query()
	res := new($structName)
	$queryParsers
	return res, nil
}`
	return namedReplace(parseFuncTmpl, map[string]string{
		"$structName":   structName,
		"$queryParsers": strings.Join(queryParserRows, "\n"),
	})
}

func GenerateCode(pkg, structName string, fields []_field) (string, error) {
	structName = fmt.Sprintf("_%sFilter", structName)
	structRcv := structrcv(structName)
	structDef, structFieldMap := generateFilterStructDef(structName, fields)
	// Generate getters
	var getters []string
	for originalField, parserFields := range structFieldMap {
		for _, field := range fields {
			if originalField != field._originalName {
				continue
			}

			for _, pf := range parserFields {
				var postfix string
				if pf.isRangeGte {
					postfix = "Gte"
				}
				if pf.isRangeLte {
					postfix = "Lte"
				}
				if pf._kind == _qfKindExact {
					postfix = "Exact"
				}
				if pf._kind == _qfKindMultiValue {
					postfix = "Array"
				}

				getters = append(getters, generateFieldGetterFunc(
					structRcv,
					structName,
					originalField,
					pf._name,
					postfix,
					ternary(pf._kind == _qfKindMultiValue, "[]"+field._goType, field._goType),
				))
			}

		}
	}

	uniqueTypes := make(map[string]struct{})
	for _, field := range fields {
		for _, kind := range field._qf._kindList {
			field._goType = strings.Trim(field._goType, "64")
			field._goType = strings.Trim(field._goType, "32")
			uniqueTypes[field._goType] = struct{}{}
			if kind == _qfKindMultiValue {
				uniqueTypes["[]"+field._goType] = struct{}{}
			}
		}
	}

	var parsers []string
	for t := range uniqueTypes {
		parsers = append(parsers, generateQueryValueParserForGoType(t))
	}

	constantsDef, parserFieldToConstMap := generateConstKeys(fields, structFieldMap)
	parserFunc := generateParserFunc(structName, fields, structFieldMap, parserFieldToConstMap)
	rows := []string{
		"// CODE IS GENERATED AUTOMATICALLY, DO NOT EDIT!!!",
		fmt.Sprintf(`package %s`, pkg),
		constantsDef,
		structDef,
		parserFunc,
	}

	rows = append(rows, getters...)
	rows = append(rows, parsers...)

	result := strings.Builder{}
	result.Grow(10000)
	result.WriteString(strings.Join(rows, "\n"))

	return result.String(), nil
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func structrcv(structName string) string {
	return structName[:3]
}

var goTypeParserFuncs = map[string]string{
	"string":    "vstrparse",
	"bool":      "vboolparse",
	"int":       "gintparse",
	"[]int":     "gsliceintparse",
	"int32":     "gintparse",
	"[]int32":   "gsliceintparse",
	"int64":     "gintparse",
	"[]int64":   "gsliceintparse",
	"uint":      "guintparse",
	"[]uint":    "gsliceuintparse",
	"uint32":    "guintparse",
	"[]uint32":  "gsliceuintparse",
	"uint64":    "guintparse",
	"[]uint64":  "gsliceuintparse",
	"float32":   "gfloatparse",
	"float64":   "gfloatparse",
	"time.Time": "vtimeparse",
}

const (
	genericIntParseFunc = `
func gintparse[I int|int32|int64](inp string) I {
	v, err := strconv.ParseInt(inp, 10, 64)
	if err != nil {
		return *new(I)
	}
	return I(v)
}`
	genericUintParseFunc = `
func guintparse[I uint|uint32|uint64](inp string) I {
	v, err := strconv.ParseUint(inp, 10, 64)
	if err != nil {
		return *new(I)
	}
	return I(v)
}`
	genericFloatParseFunc = `
func gfloatparse[I float32 | float64](inp string) I {
	v, err := strconv.ParseFloat(inp, 64)
	if err != nil {
		return *new(I)
	}
	return I(v)
}`

	genericIntSliceParseFunc = `
func gsliceintparse[I int|int32|int64](inp string) []I {
	splitted := strings.Split(inp, ",")
	result := make([]I, 0, len(splitted))
	for _, v := range splitted {
		result = append(result, gintparse[I](v))
	}
	return result
}`
	genericUintSliceParseFunc = `
func gsliceuintparse[I uint|uint32|uint64](inp string) []I {
	splitted := strings.Split(inp, ",")
	result := make([]I, 0, len(splitted))
	for _, v := range splitted {
		result = append(result, guintparse[I](v))
	}
	return result
}`
	genericFloatSliceParseFunc = `
func gslicefloatparse[I float64|float32](inp string) []I {
	splitted := strings.Split(inp, ",")
	result := make([]I, 0, len(splitted))
	for _, v := range splitted {
		result = append(result, gfloatparse[I](v))
	}
	return result
}`
)

func generateQueryValueParserForGoType(goType string) string {
	var funcTemplate string
	switch goType {
	case "int", "int64", "int32":
		return genericIntParseFunc
	case "[]int", "[]int64", "[]int32":
		return genericIntSliceParseFunc
	case "uint", "uint64", "uint32":
		return genericUintParseFunc
	case "[]uint", "[]uint64", "[]uint32":
		return genericUintSliceParseFunc
	case "float32", "float64":
		return genericFloatParseFunc
	case "[]float32", "[]float64":
		return genericFloatSliceParseFunc
	case "string":
		funcTemplate = `func %s[T any](inp string) string {return inp}`
	case "bool":
		funcTemplate = `
func %s[T any](inp string) bool {
	v, err := strconv.ParseBool(inp)
	if err != nil {
		return false
	}
	return v
}`
	case "time.Time":
		funcTemplate = `
func %s[T any](inp string) time.Time {
	v, err := time.Parse(time.RFC3339, inp)
	if err != nil {
		return time.Time{}
	}
	return v
}`
	}
	return fmt.Sprintf(funcTemplate, goTypeParserFuncs[goType])
}
