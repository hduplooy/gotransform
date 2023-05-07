package gotransform

//the main gotransform

import (
	b64 "encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Transform struct {
	DateFormat   string
	QuoteChars   string
	SkipRows     int
	CommentChars string
	Struct       interface{}
	transformers []*fieldTransformer
	fieldMap     map[string]int
}

type transformer struct {
	name   string
	Setter func(fld reflect.Value, value string) error
	Getter func(fld reflect.Value) (string, error)
}

const (
	TRIM_NONE = 0
	TRIM_LEFT = iota
	TRIM_RIGHT
	TRIM_BOTH
	TRIM_ALL
)

const (
	ALIGN_NONE = 0
	ALIGN_LEFT = iota
	ALIGN_RIGHT
	ALIGN_CENTER
)

const (
	CASE_NONE  = 0
	CASE_LOWER = iota
	CASE_UPPER
	CASE_CAPITALIZE
)

const (
	BYTE_NONE   = 0
	BYTE_BASE64 = iota
	BYTE_HEX
)

// ;trim;rtrim;ltrim;tolower;toupper;capitalize;fill='0';lalign;ralign;center;width=10;pos=20;currency='$';
type fieldTransformer struct {
	transformer
	Col      int // Within line
	Columns  []int
	Field    int // Within structure
	Name     string
	FillChar rune
	Trim     int //0-none 1-left 2-right 3-both 4-all
	Align    int // 0-none 1-left 2-right 3-center
	Case     int // 0-none 1-lower 2-upper 3-capitalize
	Pos      int
	Width    int
	Currency rune
	Format   string
	Trans    string
	Dec      bool
	Decimals int
	JoinChar string
	ByteConv int
}

var globalTransformers map[string]transformer

func init() {
	fmt.Println("Transformer init")
	globalTransformers = make(map[string]transformer)
}

func AddTransformer(name string, setter func(fld reflect.Value, value string) error, getter func(fld reflect.Value) (string, error)) {
	fmt.Println("Adding transformer " + name)
	globalTransformers[name] = transformer{name: name, Setter: setter, Getter: getter}
}

func (tr *fieldTransformer) determineSetterAndGetter(fld reflect.Value) *fieldTransformer {
	if tr.Trans != "" {
		trf, ok := globalTransformers[tr.Trans]
		fmt.Println(ok)
		if ok {
			fmt.Println("Got transformer " + tr.Trans)
			tr.Setter = trf.Setter
			tr.Getter = trf.Getter
			return tr
		}
	}
	switch fld.Kind() {
	case reflect.Bool:
		tr.Setter = SetBool
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
	case reflect.Int:
		tr.Setter = func(fld reflect.Value, value string) error {
			tmp, err := strconv.Atoi(value)
			if err != nil {
				return errors.New("Not an integer.")
			}
			fld.SetInt(int64(tmp))
			return nil
		}
	case reflect.Float64:
		tr.Setter = func(fld reflect.Value, value string) error {
			tmp, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return errors.New("Not a float64.")
			}
			fld.SetFloat(tmp)
			return nil
		}
	case reflect.Float32:
	case reflect.Complex64:
	case reflect.Complex128:
	case reflect.String:
		tr.Setter = func(fld reflect.Value, value string) error {
			fld.SetString(value)
			return nil
		}
	default:
		if fld.Type().String() == "time.Time" {
			if tr.Format == "" {
				tr.Format = "2006-01-02"
			}
			tr.Setter = func(fld reflect.Value, value string) error {
				tmp, err := time.Parse(tr.Format, value)
				if err != nil {
					return errors.New("Not a date time")
				}
				fld.Set(reflect.ValueOf(tmp))
				return nil
			}
		} else if fld.Type().String() == "[]uint8" {
			if tr.ByteConv == BYTE_BASE64 {
				tr.Setter = func(fld reflect.Value, value string) error {
					dest, err := b64.StdEncoding.DecodeString(value)
					if err != nil {
						return err
					}
					fld.Set(reflect.ValueOf(dest))
					return nil
				}
			} else if tr.ByteConv == BYTE_HEX {
				tr.Setter = func(fld reflect.Value, value string) error {
					dest, err := hex.DecodeString(value)
					if err != nil {
						return err
					}
					fld.Set(reflect.ValueOf(dest))
					return nil
				}
			}
		}
	}
	return tr
}

func fixTagValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 0 && value[0] == '\'' {
		value = value[1:]
	}
	if len(value) > 0 && value[len(value)-1] == '\'' {
		value = value[:len(value)-1]
	}
	return value
}

func (ft *fieldTransformer) processTag(tag string, col int) (int, error) {
	var err error
	if tag == "+" {
		col++
	} else {
		ents := strings.Split(tag, ";")
		for _, ent := range ents {
			tmp2 := strings.Split(ent, "=")

			key := strings.TrimSpace(tmp2[0])
			value := ""
			if len(tmp2) > 1 {
				value = fixTagValue(tmp2[1])
			}
			switch key {
			case "col":
				if value == "+" {
					col++
				} else {
					col, err = strconv.Atoi(value)
					if err != nil {
						return col, err
					}
				}
			case "format":
				ft.Format = value
			case "trans":
				ft.Trans = value
			case "trim": // for input
				ft.Trim = TRIM_BOTH
			case "ltrim": // for input
				ft.Trim = TRIM_LEFT
			case "rtrim": // for input
				ft.Trim = TRIM_RIGHT
			case "trimall": // for input
				ft.Trim = TRIM_ALL
			case "tolower": // for input
				ft.Case = CASE_LOWER
			case "toupper": // for input
				ft.Case = CASE_UPPER
			case "capitalize": // for input
				ft.Case = CASE_CAPITALIZE
			case "fill": // for output
				ft.FillChar = rune(value[0])
			case "lalign": // for output
				ft.Align = ALIGN_LEFT
			case "ralign": // for output
				ft.Align = ALIGN_RIGHT
			case "center": // for output
				ft.Align = ALIGN_CENTER
			case "width":
				ft.Width, err = strconv.Atoi(value)
				if err != nil {
					return col, err
				}
			case "pos":
				ft.Pos, err = strconv.Atoi(value)
				if err != nil {
					return col, err
				}
			case "currency":
				ft.FillChar = rune(value[0])
			case "dec":
				ft.Dec = value == "true"
			case "decimals":
				ft.Decimals, err = strconv.Atoi(value)
				if err != nil {
					return col, err
				}
			case "join":
				ft.JoinChar = value
			case "base64":
				ft.ByteConv = BYTE_BASE64
			case "hex":
				ft.ByteConv = BYTE_HEX
			default:
				if len(tmp2) == 1 {
					tmp3 := strings.Split(key, ",")
					ft.Columns = make([]int, len(tmp3))
					for i, cs := range tmp3 {
						c, err := strconv.Atoi(cs)
						if err != nil {
							return col, err
						}
						ft.Columns[i] = c
					}
					col = ft.Columns[0]
				}
			}
		}
	}

	ft.Col = col
	return col, nil
}

func (tr *Transform) prepareTransformers() error {
	var err error
	valr := reflect.ValueOf(tr.Struct)
	tmp := reflect.TypeOf(tr.Struct)
	col := 0
	for i := 0; i < tmp.NumField(); i++ {
		fld := tmp.Field(i)
		fmt.Printf("Field %d. %s\n", fld.Name)
		trf := fld.Tag.Get("trf") // trf:"-" trf:"+" trf:"23" trf:"23;format='2016-01-19'"
		if trf != "-" {
			tr.fieldMap[fld.Name] = i
			tmp := &fieldTransformer{
				Col:   col,
				Field: i,
			}
			col, err = tmp.processTag(trf, col)
			if err != nil {
				return err
			}
			tmp.determineSetterAndGetter(valr.Field(i))

			tr.transformers = append(tr.transformers, tmp)
		}
	}
	return nil
}

func NewTransform(str interface{}) (*Transform, error) {
	tmp := Transform{Struct: str}
	tmp.transformers = make([]*fieldTransformer, 0)
	tmp.QuoteChars = "'\""
	tmp.fieldMap = make(map[string]int)
	err := tmp.prepareTransformers()
	return &tmp, err
}

func (tr *Transform) SetDateFormat(format string) *Transform {
	tr.DateFormat = format
	return tr
}

func (tr *Transform) SetQuoteChars(qc string) *Transform {
	tr.QuoteChars = qc
	return tr
}

func (tr *Transform) SetSkipRows(sr int) *Transform {
	tr.SkipRows = sr
	return tr
}

func (tr *Transform) SetCommentChars(cr string) *Transform {
	tr.CommentChars = cr
	return tr
}

func (tr *Transform) PopulateStruct(values []string) (interface{}, error) {
	return nil, nil
}

func SetBool(fld reflect.Value, value string) error {
	tmp, err := strconv.ParseBool(value)
	if err != nil {
		return errors.New("Not an integer.")
	}
	fld.SetBool(bool(tmp))
	return nil
}

func (tr *Transform) SetSetter(name string, setter func(fld reflect.Value, value string) error) *Transform {
	fld, ok := tr.fieldMap[name]
	if !ok {
		return tr
	}
	for _, trf := range tr.transformers {
		if trf.Field == fld {
			trf.Setter = setter
			return tr
		}
	}
	return tr
}

func (tr *Transform) SetGetter(name string, getter func(fld reflect.Value) (string, error)) *Transform {
	fld, ok := tr.fieldMap[name]
	if !ok {
		return tr
	}
	for _, trf := range tr.transformers {
		if trf.Field == fld {
			trf.Getter = getter
			return tr
		}
	}
	return tr
}

func (ft *fieldTransformer) prepareInput(value string) string {
	fmt.Printf("TRIM %s with %d\n", value, ft.Trim)
	switch ft.Trim {
	case TRIM_LEFT:
		value = strings.TrimLeft(value, " \t\n\r")
	case TRIM_RIGHT:
		value = strings.TrimRight(value, " \t\r\n")
	case TRIM_BOTH:
		fmt.Println("TRIM BOTH of " + value)
		value = strings.TrimSpace(value)
	case TRIM_ALL:
		value = strings.ReplaceAll(value, " ", "")
	}
	switch ft.Case {
	case CASE_LOWER:
		value = strings.ToLower(value)
	case CASE_UPPER:
		value = strings.ToUpper(value)
	case CASE_CAPITALIZE:
		tmp := strings.Split(value, " ")
		for i := 0; i < len(tmp); i++ {
			tmp2 := tmp[i]
			tmp2 = strings.ToUpper(tmp2[:1]) + strings.ToLower(tmp2[1:])
			tmp[i] = tmp2
		}
		value = strings.Join(tmp, "")
	}
	fmt.Println(value)
	return value
}

func (ent *fieldTransformer) prepareInputValue(values []string) string {
	value := values[ent.Col]
	if len(ent.Columns) > 1 {
		bld := strings.Builder{}
		for i, c := range ent.Columns {
			if i > 0 {
				bld.WriteString(ent.JoinChar)
			}
			bld.WriteString(values[c])
		}
		value = bld.String()
	}
	return value
}
