package base

import (
	"errors"
	"fmt"
	"qrcode/constants"
)

var EXP_TABLE = make([]int, 256)
var LOG_TABLE = make([]int, 256)

func init() {
	for i := 0; i < 8; i++ {
		EXP_TABLE[i] = 1 << i
	}
	for i := 8; i < 256; i++ {
		EXP_TABLE[i] = EXP_TABLE[i-4] ^ EXP_TABLE[i-5] ^ EXP_TABLE[i-6] ^ EXP_TABLE[i-8]
	}
	for i := 0; i < 255; i++ {
		LOG_TABLE[EXP_TABLE[i]] = i
	}
}

var RS_BLOCK_OFFSET = map[int]int{
	constants.ERROR_CORRECT_L: 0,
	constants.ERROR_CORRECT_M: 1,
	constants.ERROR_CORRECT_Q: 2,
	constants.ERROR_CORRECT_H: 3,
}

var RS_BLOCK_TABLE = [][]int{
	{1, 26, 19}, {1, 26, 16}, {1, 26, 13}, {1, 26, 9},
	{1, 44, 34}, {1, 44, 28}, {1, 44, 22}, {1, 44, 16},
	{1, 70, 55}, {1, 70, 44}, {2, 35, 17}, {2, 35, 13},
	{1, 100, 80}, {2, 50, 32}, {2, 50, 24}, {4, 25, 9},
	{1, 134, 108}, {2, 67, 43}, {2, 33, 15, 2, 34, 16}, {2, 33, 11, 2, 34, 12},
	{2, 86, 68}, {4, 43, 27}, {4, 43, 19}, {4, 43, 15},
	{2, 98, 78}, {4, 49, 31}, {2, 32, 14, 4, 33, 15}, {4, 39, 13, 1, 40, 14},
	{2, 121, 97}, {2, 60, 38, 2, 61, 39}, {4, 40, 18, 2, 41, 19}, {4, 40, 14, 2, 41, 15},
	{2, 146, 116}, {3, 58, 36, 2, 59, 37}, {4, 36, 16, 4, 37, 17}, {4, 36, 12, 4, 37, 13},
	{2, 86, 68, 2, 87, 69}, {4, 69, 43, 1, 70, 44}, {6, 43, 19, 2, 44, 20}, {6, 43, 15, 2, 44, 16},
	{4, 101, 81}, {1, 80, 50, 4, 81, 51}, {4, 50, 22, 4, 51, 23}, {3, 36, 12, 8, 37, 13},
	{2, 116, 92, 2, 117, 93}, {6, 58, 36, 2, 59, 37}, {4, 46, 20, 6, 47, 21}, {7, 42, 14, 4, 43, 15},
	{4, 133, 107}, {8, 59, 37, 1, 60, 38}, {8, 44, 20, 4, 45, 21}, {12, 33, 11, 4, 34, 12},
	{3, 145, 115, 1, 146, 116}, {4, 64, 40, 5, 65, 41}, {11, 36, 16, 5, 37, 17}, {11, 36, 12, 5, 37, 13},
	{5, 109, 87, 1, 110, 88}, {5, 65, 41, 5, 66, 42}, {5, 54, 24, 7, 55, 25}, {11, 36, 12, 7, 37, 13},
	{5, 122, 98, 1, 123, 99}, {7, 73, 45, 3, 74, 46}, {15, 43, 19, 2, 44, 20}, {3, 45, 15, 13, 46, 16},
	{1, 135, 107, 5, 136, 108}, {10, 74, 46, 1, 75, 47}, {1, 50, 22, 15, 51, 23}, {2, 42, 14, 17, 43, 15},
	{5, 150, 120, 1, 151, 121}, {9, 69, 43, 4, 70, 44}, {17, 50, 22, 1, 51, 23}, {2, 42, 14, 19, 43, 15},
	{3, 141, 113, 4, 142, 114}, {3, 70, 44, 11, 71, 45}, {17, 47, 21, 4, 48, 22}, {9, 39, 13, 16, 40, 14},
	{3, 135, 107, 5, 136, 108}, {3, 67, 41, 13, 68, 42}, {15, 54, 24, 5, 55, 25}, {15, 43, 15, 10, 44, 16},
	{4, 144, 116, 4, 145, 117}, {17, 68, 42}, {17, 50, 22, 6, 51, 23}, {19, 46, 16, 6, 47, 17},
	{2, 139, 111, 7, 140, 112}, {17, 74, 46}, {7, 54, 24, 16, 55, 25}, {34, 37, 13},
	{4, 151, 121, 5, 152, 122}, {4, 75, 47, 14, 76, 48}, {11, 54, 24, 14, 55, 25}, {16, 45, 15, 14, 46, 16},
	{6, 147, 117, 4, 148, 118}, {6, 73, 45, 14, 74, 46}, {11, 54, 24, 16, 55, 25}, {30, 46, 16, 2, 47, 17},
	{8, 132, 106, 4, 133, 107}, {8, 75, 47, 13, 76, 48}, {7, 54, 24, 22, 55, 25}, {22, 45, 15, 13, 46, 16},
	{10, 142, 114, 2, 143, 115}, {19, 74, 46, 4, 75, 47}, {28, 50, 22, 6, 51, 23}, {33, 46, 16, 4, 47, 17},
	{8, 152, 122, 4, 153, 123}, {22, 73, 45, 3, 74, 46}, {8, 53, 23, 26, 54, 24}, {12, 45, 15, 28, 46, 16},
	{3, 147, 117, 10, 148, 118}, {3, 73, 45, 23, 74, 46}, {4, 54, 24, 31, 55, 25}, {11, 45, 15, 31, 46, 16},
	{7, 146, 116, 7, 147, 117}, {21, 73, 45, 7, 74, 46}, {1, 53, 23, 37, 54, 24}, {19, 45, 15, 26, 46, 16},
	{5, 145, 115, 10, 146, 116}, {19, 75, 47, 10, 76, 48}, {15, 54, 24, 25, 55, 25}, {23, 45, 15, 25, 46, 16},
	{13, 145, 115, 3, 146, 116}, {2, 74, 46, 29, 75, 47}, {42, 54, 24, 1, 55, 25}, {23, 45, 15, 28, 46, 16},
	{17, 145, 115}, {10, 74, 46, 23, 75, 47}, {10, 54, 24, 35, 55, 25}, {19, 45, 15, 35, 46, 16},
	{17, 145, 115, 1, 146, 116}, {14, 74, 46, 21, 75, 47}, {29, 54, 24, 19, 55, 25}, {11, 45, 15, 46, 46, 16},
	{13, 145, 115, 6, 146, 116}, {14, 74, 46, 23, 75, 47}, {44, 54, 24, 7, 55, 25}, {59, 46, 16, 1, 47, 17},
	{12, 151, 121, 7, 152, 122}, {12, 75, 47, 26, 76, 48}, {39, 54, 24, 14, 55, 25}, {22, 45, 15, 41, 46, 16},
	{6, 151, 121, 14, 152, 122}, {6, 75, 47, 34, 76, 48}, {46, 54, 24, 10, 55, 25}, {2, 45, 15, 64, 46, 16},
	{17, 152, 122, 4, 153, 123}, {29, 74, 46, 14, 75, 47}, {49, 54, 24, 10, 55, 25}, {24, 45, 15, 46, 46, 16},
	{4, 152, 122, 18, 153, 123}, {13, 74, 46, 32, 75, 47}, {48, 54, 24, 14, 55, 25}, {42, 45, 15, 32, 46, 16},
	{20, 147, 117, 4, 148, 118}, {40, 75, 47, 7, 76, 48}, {43, 54, 24, 22, 55, 25}, {10, 45, 15, 67, 46, 16},
	{19, 148, 118, 6, 149, 119}, {18, 75, 47, 31, 76, 48}, {34, 54, 24, 34, 55, 25}, {20, 45, 15, 61, 46, 16},
}

func glog(n int) (int, error) {
	if n < 1 {
		return 0, fmt.Errorf("glog(%d)", n)
	}
	return LOG_TABLE[n], nil
}

func Gexp(n int) int {
	return EXP_TABLE[n%255]
}

type Polynomial struct {
	num []int
}

func NewPolynomial(num []int, shift int) (*Polynomial, error) {
	if len(num) == 0 {
		return nil, errors.New(fmt.Sprintf("%d/%d", len(num), shift))
	}

	offset := 0
	for offset = range num {
		if num[offset] != 0 {
			break
		}
	}

	p := &Polynomial{
		num: append(num[offset:], make([]int, shift)...),
	}
	return p, nil
}

func (p *Polynomial) Get(index int) int {
	return p.num[index]
}

func (p *Polynomial) Len() int {
	return len(p.num)
}

func (p *Polynomial) Mul(other *Polynomial) (*Polynomial, error) {
	num := make([]int, p.Len()+other.Len()-1)

	for i, item := range p.num {
		for j, otherItem := range other.num {
			glogItem, err := glog(item)
			if err != nil {
				return nil, err
			}
			glogOtherItem, err := glog(otherItem)
			if err != nil {
				return nil, err
			}
			num[i+j] ^= Gexp(glogItem + glogOtherItem)
		}
	}

	return NewPolynomial(num, 0)
}

func (p *Polynomial) Mod(other *Polynomial) (*Polynomial, error) {
	for p.Len() >= other.Len() {
		difference := p.Len() - other.Len()
		if difference < 0 {
			return p, nil
		}

		glogP0, err := glog(p.Get(0))
		if err != nil {
			return nil, err
		}
		glogOther0, err := glog(other.Get(0))
		if err != nil {
			return nil, err
		}
		ratio := glogP0 - glogOther0

		num := make([]int, len(p.num))
		copy(num, p.num)
		for i := range other.num {
			glogOtherItem, err := glog(other.Get(i))
			if err != nil {
				return nil, err
			}
			num[i] ^= Gexp(glogOtherItem + ratio)
		}
		// Remove leading zeros
		for len(num) > 0 && num[0] == 0 {
			num = num[1:]
		}

		modPoly, err := NewPolynomial(num, 0)
		if err != nil {
			return nil, err
		}
		p = modPoly
	}
	return p, nil
}

type RSBlock struct {
	TotalCount int
	DataCount  int
}

func RSBlocks(version int, errorCorrection int) ([]RSBlock, error) {
	offset, ok := RS_BLOCK_OFFSET[errorCorrection]
	if !ok {
		return nil, fmt.Errorf("bad rs block @ version: %d / error_correction: %d", version, errorCorrection)
	}
	rsBlock := RS_BLOCK_TABLE[(version-1)*4+offset]

	var blocks []RSBlock
	for i := 0; i < len(rsBlock); i += 3 {
		count := rsBlock[i]
		totalCount := rsBlock[i+1]
		dataCount := rsBlock[i+2]
		for j := 0; j < count; j++ {
			blocks = append(blocks, RSBlock{TotalCount: totalCount, DataCount: dataCount})
		}
	}

	return blocks, nil
}
