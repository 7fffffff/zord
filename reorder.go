package zord

// reorder reads a JSON object from src and transforms it by moving the
// key-value pairs named in firstKeys to the beginning of the object. Only top
// level keys are moved. reorder does not change any nested objects within the
// object.
//
// reorder does not deduplicate keys. If there are duplicate keys matching
// firstKeys, they are all moved to the front, with their relative ordering
// preserved.
//
// reorder appends the transformed object to dest, then returns the extended
// dest and the number of bytes read from src. If the length of firstKeys is 0,
// src is appended as-is.
func reorder(dest, src []byte, firstKeys []string) ([]byte, int, error) {
	if len(firstKeys) == 0 {
		return append(dest, src...), len(src), nil
	}
	parser := &parser{}
	pairs, n, err := parser.parse(src)
	if err != nil {
		return dest, n, err
	}
	keyPositions := map[string][]int{}
	for i, pair := range pairs {
		keyPositions[pair.keyUnquoted] = append(keyPositions[pair.keyUnquoted], i)
	}
	pairsWritten := 0
	skip := map[int]struct{}{}
	dest = append(dest, '{')
	for _, key := range firstKeys {
		for _, i := range keyPositions[key] {
			if _, ok := skip[i]; ok {
				continue
			}
			pair := pairs[i]
			if pairsWritten > 0 {
				dest = append(dest, ',')
			}
			dest = append(dest, pair.keyBytes...)
			dest = append(dest, ':')
			dest = append(dest, pair.valueBytes...)
			skip[i] = struct{}{}
			pairsWritten++
		}
	}
	for i, pair := range pairs {
		if _, ok := skip[i]; ok {
			continue
		}
		if pairsWritten > 0 {
			dest = append(dest, ',')
		}
		dest = append(dest, pair.keyBytes...)
		dest = append(dest, ':')
		dest = append(dest, pair.valueBytes...)
		pairsWritten++
	}
	dest = append(dest, '}')
	return dest, n, nil
}
