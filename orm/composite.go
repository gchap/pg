package orm

import (
	"fmt"
	"reflect"

	"github.com/go-pg/pg/types"
)

func compositeScanner(typ reflect.Type) types.ScannerFunc {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var table *Table
	return func(v reflect.Value, rd types.Reader, n int) error {
		if !v.CanSet() {
			return fmt.Errorf("pg: Scan(nonsettable %s)", v.Type())
		}

		if n == -1 {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}

		if table == nil {
			table = GetTable(typ)
		}
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}

		p := newCompositeParser(rd)
		var elemReader *types.BytesReader

		var firstErr error
		for i := 0; ; i++ {
			elem, err := p.NextElem()
			if err != nil {
				if err == endOfComposite {
					break
				}
				return err
			}

			if i >= len(table.allFields) {
				if firstErr == nil {
					firstErr = fmt.Errorf(
						"%s has %d fields, but composite at least %d values",
						table, len(table.allFields), i)
				}
				continue
			}

			if elemReader == nil {
				elemReader = types.NewBytesReader(elem)
			} else {
				elemReader.Reset(elem)
			}

			field := table.allFields[i]
			err = field.ScanValue(v, elemReader, len(elem))
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}

		return firstErr
	}
}

func compositeAppender(typ reflect.Type) types.AppenderFunc {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var table *Table
	return func(b []byte, v reflect.Value, quote int) []byte {
		if table == nil {
			table = GetTable(typ)
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		b = append(b, '(')
		for i, f := range table.Fields {
			if i > 0 {
				b = append(b, ',')
			}
			b = f.AppendValue(b, v, quote)
		}
		b = append(b, ')')
		return b
	}
}
