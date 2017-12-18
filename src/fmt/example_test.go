// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fmt_test

import (
	"fmt"
)

// Animal has a Name and an Age to represent an animal.
type Animal struct {
	Name string
	Age  uint
}

// String makes Animal satisfy the Stringer interface.
func (a Animal) String() string {
	return fmt.Sprintf("%v (%d)", a.Name, a.Age)
}

func ExampleStringer() {
	a := Animal{
		Name: "Gopher",
		Age:  2,
	}
	fmt.Println(a)
	// Output: Gopher (2)
}

func ExamplePrintf_flagV() {
	type X struct {
		A int
		B string
	}
	type Y struct {
		D X
		E []int
		F [2]string
	}
	type Z struct {
		G Y
		H string
		I []string
		J map[string]int
	}
	var z = Z{
		G: Y{
			D: X{
				A: 123,
				B: `"b" = 1`,
			},
			E: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			F: [2]string{
				`aaa`,
				`bbb`,
			},
		},
		H: `zzz`,
		I: []string{
			`c:\x\y\z`,
			`d:\a\b\c`,
		},
		J: map[string]int{
			`abc`: 456,
		},
	}
	fmt.Printf("-------\n\"%%v\":\n%v\n", z)
	fmt.Printf("-------\n\"%%@v\":\n%@v\n", z)
	fmt.Printf("-------\n\"%%#v\":\n%#v\n", z)
	fmt.Printf("-------\n\"%%@#v\":\n%@#v\n", z)
	fmt.Printf("-------\n\"%%+v\":\n%+v\n", z)
	fmt.Printf("-------\n\"%%@+v\":\n%@+v\n", z)

	// Output:
	// -------
	// "%v":
	// {{{123 "b" = 1} [1 2 3 4 5 6 7 8 9 10 11 12] [aaa bbb]} zzz [c:\x\y\z d:\a\b\c] map[abc:456]}
	// -------
	// "%@v":
	// {
	//     {
	//         {
	//             123
	//             "b" = 1
	//         }
	//         [
	//             1 2 3 4 5 6 7 8 9 10
	//             11 12
	//         ]
	//         [
	//             aaa
	//             bbb
	//         ]
	//     }
	//     zzz
	//     [
	//         c:\x\y\z
	//         d:\a\b\c
	//     ]
	//     map[
	//         abc: 456
	//     ]
	// }
	//
	// -------
	// "%#v":
	// fmt_test.Z{G:fmt_test.Y{D:fmt_test.X{A:123, B:"\"b\" = 1"}, E:[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, F:[2]string{"aaa", "bbb"}}, H:"zzz", I:[]string{"c:\\x\\y\\z", "d:\\a\\b\\c"}, J:map[string]int{"abc":456}}
	// -------
	// "%@#v":
	// fmt_test.Z{
	//     G: fmt_test.Y{
	//         D: fmt_test.X{
	//             A: 123,
	//             B: `"b" = 1`,
	//         },
	//         E: []int{
	//             1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
	//             11, 12,
	//         },
	//         F: [2]string{
	//             `aaa`,
	//             `bbb`,
	//         },
	//     },
	//     H: `zzz`,
	//     I: []string{
	//         `c:\x\y\z`,
	//         `d:\a\b\c`,
	//     },
	//     J: map[string]int{
	//         `abc`: 456,
	//     },
	// }
	//
	// -------
	// "%+v":
	// {G:{D:{A:123 B:"b" = 1} E:[1 2 3 4 5 6 7 8 9 10 11 12] F:[aaa bbb]} H:zzz I:[c:\x\y\z d:\a\b\c] J:map[abc:456]}
	// -------
	// "%@+v":
	// {
	//     G: {
	//         D: {
	//             A: 123
	//             B: "b" = 1
	//         }
	//         E: [
	//             1 2 3 4 5 6 7 8 9 10
	//             11 12
	//         ]
	//         F: [
	//             aaa
	//             bbb
	//         ]
	//     }
	//     H: zzz
	//     I: [
	//         c:\x\y\z
	//         d:\a\b\c
	//     ]
	//     J: map[
	//         abc: 456
	//     ]
	// }
	//
}
