package generator

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"sort"
	"strings"
)

// WriteProtobufHeader writes the file header for a generated protocol buffer
// definition.
func WriteProtobufHeader(out io.Writer, src, pkg string) error {
	if _, err := fmt.Fprintln(out, "// Code generated by go generate."); err != nil {
		return err
	}
	fmt.Fprintln(out, "// source:", src)
	fmt.Fprintln(out, "// DO NOT EDIT!")
	fmt.Fprintln(out)

	fmt.Fprintf(out, "package %s;", pkg)
	fmt.Fprintln(out)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "import \"github.com/gogo/protobuf/gogoproto/gogo.proto\";")
	_, err := fmt.Fprintln(out)

	return err
}

var nonNameChars = regexp.MustCompile(`[^A-Za-z0-9_]`)
var beginsNumeric = regexp.MustCompile(`^_*[0-9]`)

// ProtobufEnumName mangles a name to make it a legal (and style guide
// approved) identifier within a protocol buffer definition.
//
// The return value will consist only of upper case letters, numbers, and
// underscores.  Any non-alphanumeric characters in the input are converted to
// underscores.
//
// If the first character of the input is numeric, an underscore is prepended,
// to differentiate the identifier from a numeric constant.  If the input
// begins with one or more non-alphanumeric characters followed by a number,
// an underscore is also prepended.  This latter rule is to ensure that the
// name mangling doesn't produce duplicate identifiers, however it means that
// this function should not be called multiple times over the same string.
func ProtobufEnumName(origName string) string {
	name := strings.ToUpper(nonNameChars.ReplaceAllString(origName, "_"))
	if beginsNumeric.MatchString(name) {
		name = "_" + name
	}
	return name
}

// MakeStableEnum assigns stable integer values to enum names, for use in a
// protocol buffer declaration.  This is used if in the future the enum may
// have its entries reordered or new entries added, and we don't want existing
// protocol buffers to break.
//
// A list of "special" entries can be provided, which must be a subset of the
// main list.  Values 1 to 127 (which encode as a single byte) are reserved
// for the members of this list.  It's normally used for the most common
// entries, although it can also be a workaround for hash collisions, by
// moving one of colliding entries to this list.
//
// N.B. the order of the special list can never change!  Additional entries
// can be appended later, but if any entries are removed, existing protocol
// buffers will be incompatible with the new definition.
func MakeStableEnum(values, special []string) (map[int]string, error) {
	valuesMap := make(map[int]string, len(values)+len(special))

	if len(special) > 127 {
		return nil, errors.New("too many special values")
	}
	specialSeen := make(map[string]bool, len(special))
	for n, v := range special {
		if _, exists := specialSeen[v]; exists {
			return nil, fmt.Errorf("duplicate special value %q", v)
		}
		specialSeen[v] = false

		valuesMap[n+1] = v
	}

	h := fnv.New32a()

	seen := make(map[string]bool, len(values))
	for _, v := range values {
		duplicate, isSpecial := specialSeen[v]
		if !isSpecial {
			_, duplicate = seen[v]
		}
		if duplicate {
			return nil, fmt.Errorf("duplicate value %q", v)
		}
		if isSpecial {
			specialSeen[v] = true
			continue
		}
		seen[v] = true

		h.Reset()
		h.Write([]byte(v))

		// Protocol buffers used *signed* int32 for enums, so
		// drop high bit.
		n := (h.Sum32() + 128) & 0x7fffffff
		if n < 128 {
			return nil, fmt.Errorf("hash value was unlucky for %q", v)
		}
		if collision, found := valuesMap[int(n)]; found {
			return nil, fmt.Errorf("hash collision between %q and %q", collision, v)
		}

		valuesMap[int(n)] = v
	}

	for v, exists := range specialSeen {
		if !exists {
			return nil, fmt.Errorf("special %q missing from main list", v)
		}
	}

	return valuesMap, nil
}

// WriteProtobufEnum writes a protocol buffer enum definition to the specified
// io.Writer.
//
// This function can be passed a map[int]string, where the enum value is
// assigned, or a []string, in which case the enum value will be allocated
// automatically.
//
// A list of "special" entries can be provided, which may be a subset of the
// main list.  Values 1 to 127 (which encode as a single byte) are reserved
// for the members of this list.  It's normally used for the most common
// entries.
//
// N.B. the order of any []string input to this function can never change!
// Additional entries can be appended later, but if any entries are removed,
// existing protocol buffers will be incompatible with the new definition.
// MakeStableEnum can be used to convert a []string to a map[int]string with
// stable assignments.
func WriteProtobufEnum(out io.Writer, name string, values interface{}, special []string) error {
	if len(special) > 127 {
		return errors.New("too many special values")
	}
	specialSeen := make(map[string]bool, len(special))
	for _, v := range special {
		if _, exists := specialSeen[v]; exists {
			return fmt.Errorf("duplicate special value %q", v)
		}
		specialSeen[v] = false
	}

	fmt.Fprintf(out, "enum %s {", name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "\toption (gogoproto.goproto_enum_stringer) = false;")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "\tNil%s = 0;", name)
	fmt.Fprintln(out)

	if valuesMap, ok := values.(map[int]string); ok {
		for n, v := range special {
			n = n + 1
			if other, exists := valuesMap[n]; exists {
				if other == v {
					continue
				}
				return fmt.Errorf("entry %q conflicts with special %q for value %d", other, v, n)
			}
			valuesMap[n] = v
		}

		// Entries are written in ascending order by value.
		vs := make([]int, 0, len(valuesMap))
		for v := range valuesMap {
			vs = append(vs, v)
		}
		sort.Ints(vs)

		for _, v := range vs {
			fmt.Fprintf(out, "\t%s = %d;", ProtobufEnumName(valuesMap[v]), v)
			fmt.Fprintln(out)
		}
	} else if valuesSlice, ok := values.([]string); ok {
		n := 1

		for _, v := range special {
			fmt.Fprintf(out, "\t%s = %d;", ProtobufEnumName(v), n)
			n++
			fmt.Fprintln(out)
		}
		if len(special) > 0 {
			n = 128
		}

		seen := make(map[string]bool, len(valuesSlice))
		fmt.Fprintln(out)
		for _, v := range valuesSlice {
			if _, exists := seen[v]; exists {
				return fmt.Errorf("duplicate value %q", v)
			}
			seen[v] = true

			fmt.Fprintf(out, "\t%s = %d;", ProtobufEnumName(v), n)
			fmt.Fprintln(out)
			n++
		}
	} else {
		return errors.New("map[int]string or []string required")
	}

	_, err := fmt.Fprintln(out, "}")
	return err
}