package converter

import (
	"errors"
	"fmt"
	"reflect"
)

type Converter struct {
	registry map[reflect.Type]map[reflect.Type]reflect.Value
}

func New() *Converter {
	converter := new(Converter)
	converter.registry = make(map[reflect.Type]map[reflect.Type]reflect.Value)
	return converter
}

// Register a convert handler, handler must be the func(inType) (outType, error)
func (c *Converter) Register(handler interface{}) error {
	inType := reflect.TypeOf(handler).In(0)
	outType := reflect.TypeOf(handler).Out(0)
	handlerValue := reflect.ValueOf(handler)
	if err := c.verifyHandler(handlerValue.Type()); err != nil {
		return err
	}

	outMap, ok := c.registry[inType]
	if !ok {
		outMap = make(map[reflect.Type]reflect.Value)
		c.registry[inType] = outMap
	}
	outMap[outType] = handlerValue
	return nil
}

// verifyHandler will check the input handler type
func (c *Converter) verifyHandler(handlerType reflect.Type) error {
	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be func(T1) (T2, error)")
	}
	if handlerType.NumIn() != 1 || handlerType.NumOut() != 2 || handlerType.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("%s doesn't match func(T1) (T2, error)", handlerType.Name())
	}
	return nil
}

func (c *Converter) Convert(in, out interface{}) error {
	result, err := c.getResult(reflect.ValueOf(in), getOutActualType(out))
	if err != nil {
		return err
	}
	reflect.ValueOf(out).Elem().Set(result)
	return nil
}

func getOutActualType(out interface{}) reflect.Type {
	v := reflect.ValueOf(out).Elem()
	for {
		if v.Kind() != reflect.Interface {
			break
		}
		v = v.Elem()
	}
	return v.Type()
}

func (c *Converter) getResult(inValue reflect.Value, outType reflect.Type) (reflect.Value, error) {
	inType := inValue.Type()
	handler, err := c.getHandler(inType, outType)
	if err != nil {
		if inType.Kind() != reflect.Slice || outType.Kind() != reflect.Slice {
			return reflect.Value{}, err
		}
		return c.getSliceResult(inValue, outType)
	}
	return executeHandler(handler, inValue)
}

func (c *Converter) getSliceResult(inValue reflect.Value, outType reflect.Type) (reflect.Value, error) {
	l := inValue.Len()
	s := reflect.MakeSlice(outType, l, l)
	handler, err := c.getHandler(inValue.Type().Elem(), outType.Elem())
	if err != nil {
		return reflect.Value{}, err
	}
	for i := 0; i < l; i++ {
		result, err := executeHandler(handler, inValue.Index(i))
		if err != nil {
			return reflect.Value{}, err
		}
		s.Index(i).Set(result)
	}
	return s, nil
}

func (c *Converter) getHandler(in, out reflect.Type) (reflect.Value, error) {
	outMap, ok := c.registry[in]
	if !ok {
		return reflect.Value{}, errors.New("can't find any matches")
	}
	handler, ok := outMap[out]
	if !ok {
		return reflect.Value{}, errors.New("can't find any matches")
	}
	return handler, nil
}

func executeHandler(handler reflect.Value, in reflect.Value) (reflect.Value, error) {
	result := handler.Call([]reflect.Value{in})
	if result[1].Interface() != nil {
		return reflect.Value{}, result[1].Interface().(error)
	}
	return result[0], nil
}
