package sliceutil

func SliceMapper[T any, V any](items []T, mpr func(T) V) []V {
	if items == nil {
		return nil
	}
	slice, _ := SliceMapperError(items, func(i T) (V, error) {
		return mpr(i), nil
	})
	return slice
}

func SliceMapperError[T any, V any](items []T, mpr func(T) (V, error)) (slice []V, err error) {
	if items == nil {
		return nil, nil
	}
	slice = make([]V, len(items))
	for i, v := range items {
		slice[i], err = mpr(v)
		if err != nil {
			return nil, err
		}
	}
	return slice, nil
}

func SliceMapperFilter[T any, V any](items []T, mpr func(T) (bool, V)) []V {
	if items == nil {
		return nil
	}
	slice := make([]V, 0)
	for _, i := range items {
		keep, val := mpr(i)
		if !keep {
			continue
		}
		slice = append(slice, val)
	}
	return slice
}

func SliceUnique[T any, C comparable](arr []T, eq func(T) C) []T {
	unique := make([]T, 0)
	seen := map[C]struct{}{}
	for _, v := range arr {
		k := eq(v)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			unique = append(unique, v)
		}
	}
	return unique
}

func SliceContains[T comparable](arr []T, val T) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func SliceCopy[T any](arr []T) []T {
	if arr == nil {
		return nil
	}
	return append([]T(nil), arr...)
}
