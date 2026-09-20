package config

import "reflect"

type MapT interface {
	*AIServiceT | any
}

// MergeMap merges newer into older and returns older. Nested maps are merged;
// all other values in newer replace the corresponding values in older.
func MergeMap[M MapT](older, newer map[string]M) map[string]M {
	if older == nil {
		older = make(map[string]M)
	}

	mergeMapValues(reflect.ValueOf(older), reflect.ValueOf(newer))
	return older
}

func mergeMapValues(older, newer reflect.Value) {
	older = indirectValue(older)
	newer = indirectValue(newer)
	if !older.IsValid() || !newer.IsValid() || older.Kind() != reflect.Map || newer.Kind() != reflect.Map {
		return
	}
	if older.IsNil() {
		if !older.CanSet() {
			return
		}
		older.Set(reflect.MakeMap(older.Type()))
	}

	for entries := newer.MapRange(); entries.Next(); {
		key := indirectValue(entries.Key())
		if !key.IsValid() {
			continue
		}
		if !key.Type().AssignableTo(older.Type().Key()) {
			if !key.Type().ConvertibleTo(older.Type().Key()) {
				continue
			}
			key = key.Convert(older.Type().Key())
		}

		newValue := entries.Value()
		oldValue := older.MapIndex(key)
		if oldValue.IsValid() {
			mergedValue := mergeMapValue(oldValue, newValue, older.Type().Elem())
			if mergedValue.IsValid() {
				older.SetMapIndex(key, mergedValue)
			}
			continue
		}

		if value, ok := mapValueForType(newValue, older.Type().Elem()); ok {
			older.SetMapIndex(key, value)
		}
	}
}

func mergeMapValue(older, newer reflect.Value, valueType reflect.Type) reflect.Value {
	olderValue := indirectValue(older)
	newerValue := indirectValue(newer)
	if olderValue.IsValid() && newerValue.IsValid() &&
		olderValue.Kind() == reflect.Map && newerValue.Kind() == reflect.Map {
		mergeMapValues(olderValue, newerValue)
		return older
	}
	if olderValue.IsValid() && newerValue.IsValid() &&
		olderValue.Kind() == reflect.Struct && newerValue.Kind() == reflect.Struct &&
		olderValue.Type() == newerValue.Type() {
		mergeStructValues(olderValue, newerValue)
		return older
	}

	if value, ok := mapValueForType(newer, valueType); ok {
		return value
	}
	return reflect.Value{}
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func mapValueForType(value reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Zero(targetType), true
	}
	if value.Type().AssignableTo(targetType) {
		return value, true
	}
	if value.Type().ConvertibleTo(targetType) {
		return value.Convert(targetType), true
	}
	return reflect.Value{}, false
}

func mergeStructValues(older, newer reflect.Value) {
	for fieldIndex := 0; fieldIndex < older.NumField(); fieldIndex++ {
		oldField := older.Field(fieldIndex)
		newField := newer.Field(fieldIndex)
		if !oldField.CanSet() || isZeroValue(newField) {
			continue
		}

		oldFieldValue := indirectValue(oldField)
		newFieldValue := indirectValue(newField)
		if oldFieldValue.IsValid() && newFieldValue.IsValid() &&
			oldFieldValue.Kind() == reflect.Map && newFieldValue.Kind() == reflect.Map {
			mergeMapValues(oldFieldValue, newFieldValue)
			continue
		}
		if oldFieldValue.IsValid() && newFieldValue.IsValid() &&
			oldFieldValue.Kind() == reflect.Struct && newFieldValue.Kind() == reflect.Struct &&
			oldFieldValue.Type() == newFieldValue.Type() {
			mergeStructValues(oldFieldValue, newFieldValue)
			continue
		}
		if newField.Type().AssignableTo(oldField.Type()) {
			oldField.Set(newField)
		}
	}
}

func isZeroValue(value reflect.Value) bool {
	return !value.IsValid() || value.IsZero()
}
