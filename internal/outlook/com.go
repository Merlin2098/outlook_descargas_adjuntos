//go:build windows

package outlook

import (
	"fmt"
	"math"
	"time"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// Helpers de acceso a propiedades y métodos COM. Todos liberan el VARIANT intermedio; el
// *ole.IDispatch que devuelven getObj/callObj pasa a ser del llamador, que debe hacer release.
// Solo se usan dentro del STA.

func getObj(d *ole.IDispatch, name string, args ...any) (*ole.IDispatch, error) {
	v, err := oleutil.GetProperty(d, name, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return dispatchOf(v, name)
}

func callObj(d *ole.IDispatch, method string, args ...any) (*ole.IDispatch, error) {
	v, err := oleutil.CallMethod(d, method, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return dispatchOf(v, method)
}

// dispatchOf toma el IDispatch del VARIANT sin AddRef extra: la referencia del VARIANT pasa al llamador.
func dispatchOf(v *ole.VARIANT, name string) (*ole.IDispatch, error) {
	if v.VT != ole.VT_DISPATCH || v.Val == 0 {
		_ = v.Clear()
		return nil, fmt.Errorf("%s no es un objeto (VT %d)", name, v.VT)
	}
	return v.ToIDispatch(), nil
}

// value lee una propiedad, la convierte con conv y libera el VARIANT.
func value[T any](d *ole.IDispatch, name string, args []any, conv func(*ole.VARIANT) (T, error)) (T, error) {
	var zero T
	v, err := oleutil.GetProperty(d, name, args...)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", name, err)
	}
	defer v.Clear()
	return conv(v)
}

func getString(d *ole.IDispatch, name string, args ...any) (string, error) {
	return value(d, name, args, func(v *ole.VARIANT) (string, error) {
		switch v.VT {
		case ole.VT_BSTR:
			return v.ToString(), nil
		case ole.VT_EMPTY, ole.VT_NULL:
			return "", nil
		default:
			return fmt.Sprint(v.Value()), nil
		}
	})
}

func getInt(d *ole.IDispatch, name string, args ...any) (int, error) {
	return value(d, name, args, func(v *ole.VARIANT) (int, error) {
		switch x := v.Value().(type) {
		case int8:
			return int(x), nil
		case int16:
			return int(x), nil
		case int32:
			return int(x), nil
		case int64:
			return int(x), nil
		case uint8:
			return int(x), nil
		case uint16:
			return int(x), nil
		case uint32:
			return int(x), nil
		case uint64:
			return int(x), nil
		case int:
			return x, nil
		case float64:
			return int(x), nil
		case float32:
			return int(x), nil
		default:
			return 0, fmt.Errorf("%s no es un número (VT %d)", name, v.VT)
		}
	})
}

func getBool(d *ole.IDispatch, name string, args ...any) (bool, error) {
	return value(d, name, args, func(v *ole.VARIANT) (bool, error) {
		switch x := v.Value().(type) {
		case bool:
			return x, nil
		case nil:
			return false, nil
		default:
			return false, fmt.Errorf("%s no es booleano (VT %d)", name, v.VT)
		}
	})
}

// getDate lee una fecha VT_DATE desde su float64 OLE (no usa VARIANT.Value: ver oleDateToLocal).
func getDate(d *ole.IDispatch, name string) (time.Time, error) {
	return value(d, name, nil, func(v *ole.VARIANT) (time.Time, error) {
		if v.VT != ole.VT_DATE {
			return time.Time{}, fmt.Errorf("%s no es una fecha (VT %d)", name, v.VT)
		}
		return oleDateToLocal(math.Float64frombits(uint64(v.Val))), nil
	})
}

func call(d *ole.IDispatch, method string, args ...any) error {
	v, err := oleutil.CallMethod(d, method, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return v.Clear()
}

// release libera los objetos dados; ignora nil.
func release(ds ...*ole.IDispatch) {
	for _, d := range ds {
		if d != nil {
			d.Release()
		}
	}
}
