package gotransform

import (
	"errors"
	"fmt"
	"reflect"
)

func (tr *Transform) StructToArr(input interface{}) ([][]string, error) {
	return nil, nil
}

func (tr *Transform) ArrToStruct(values [][]string) (interface{}, error) {
	if valr := reflect.ValueOf(tr.Struct); valr.Kind() != reflect.Struct {
		return nil, errors.New("the provided record must be a structure")
	}

	slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(tr.Struct)), 0, 100)
	for _, row := range values {
		newStruct := reflect.Indirect(reflect.New(reflect.TypeOf(tr.Struct)))
		for fld, ent := range tr.transformers {
			if ent.Col > len(row) {
				return nil, errors.New(fmt.Sprintf("Not enough columns in row %d", fld))
			}
			fld2 := newStruct.Field(ent.Field)
			value := ent.prepareInputValue(row)
			err := ent.Setter(fld2, ent.prepareInput(value))
			if err != nil {
				fmt.Printf("Problem with %s\n", row[ent.Col])
				return nil, err
			}
		}
		slice = reflect.Append(slice, newStruct)
	}

	return slice.Interface(), nil
}
