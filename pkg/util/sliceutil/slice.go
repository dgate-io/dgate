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


// BinarySearch searches for a value in a sorted slice and returns the index of the value.
// If the value is not found, it returns -1
func BinarySearch[T any](slice []T, val T, less func(T, T) bool) int {
	low, high := 0, len(slice)-1
	for low <= high {
		mid := low + (high-low)/2
		if less(slice[mid], val) {
			low = mid + 1
		} else if less(val, slice[mid]) {
			high = mid - 1
		} else {
			return mid
		}
	}
	return -1
}