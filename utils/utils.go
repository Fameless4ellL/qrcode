package utils

import (
	"fmt"
	"log"
	"math"
	"os"
	"qrcode/base"
	"regexp"
	"strings"
)

// QR encoding modes
const (
	ModeNumeric      = 1 << iota
	ModeAlphanumeric = 1 << 1
	ModeByte         = 1 << 2
	ModeKanji        = 1 << 3
)

// Encoding mode sizes
var ModeSizeSmall = map[int]int{
	ModeNumeric:      10,
	ModeAlphanumeric: 9,
	ModeByte:         8,
	ModeKanji:        8,
}
var ModeSizeMedium = map[int]int{
	ModeNumeric:      12,
	ModeAlphanumeric: 11,
	ModeByte:         16,
	ModeKanji:        10,
}
var ModeSizeLarge = map[int]int{
	ModeNumeric:      14,
	ModeAlphanumeric: 13,
	ModeByte:         16,
	ModeKanji:        12,
}

const (
	AlphanumericChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"
)

var ReAlphaNumeric = regexp.MustCompile("^[" + regexp.QuoteMeta(AlphanumericChars) + "]*$")

// Number of bits for numeric delimited data lengths
var NumberLength = map[int]int{
	3: 10,
	2: 7,
	1: 4,
}

// PatternPositionTable defines the position of alignment patterns for each version of the QR code
var PatternPositionTable = [][]int{
	{},
	{6, 18},
	{6, 22},
	{6, 26},
	{6, 30},
	{6, 34},
	{6, 22, 38},
	{6, 24, 42},
	{6, 26, 46},
	{6, 28, 50},
	{6, 30, 54},
	{6, 32, 58},
	{6, 34, 62},
	{6, 26, 46, 66},
	{6, 26, 48, 70},
	{6, 26, 50, 74},
	{6, 30, 54, 78},
	{6, 30, 56, 82},
	{6, 30, 58, 86},
	{6, 34, 62, 90},
	{6, 28, 50, 72, 94},
	{6, 26, 50, 74, 98},
	{6, 30, 54, 78, 102},
	{6, 28, 54, 80, 106},
	{6, 32, 58, 84, 110},
	{6, 30, 58, 86, 114},
	{6, 34, 62, 90, 118},
	{6, 26, 50, 74, 98, 122},
	{6, 30, 54, 78, 102, 126},
	{6, 26, 52, 78, 104, 130},
	{6, 30, 56, 82, 108, 134},
	{6, 34, 60, 86, 112, 138},
	{6, 30, 58, 86, 114, 142},
	{6, 34, 62, 90, 118, 146},
	{6, 30, 54, 78, 102, 126, 150},
	{6, 24, 50, 76, 102, 128, 154},
	{6, 28, 54, 80, 106, 132, 158},
	{6, 32, 58, 84, 110, 136, 162},
	{6, 26, 54, 82, 110, 138, 166},
	{6, 30, 58, 86, 114, 142, 170},
}

const G15 = (1 << 10) | (1 << 8) | (1 << 5) | (1 << 4) | (1 << 2) | (1 << 1) | (1 << 0)
const G18 = (1 << 12) | (1 << 11) | (1 << 10) | (1 << 9) | (1 << 8) | (1 << 5) | (1 << 2) | (1 << 0)
const G15_MASK = (1 << 14) | (1 << 12) | (1 << 10) | (1 << 4) | (1 << 1)

const PAD0 = 0xEC
const PAD1 = 0x11

// BCHTypeInfo calculates BCH code for type information.
func BCHTypeInfo(data int) int {
	d := data << 10
	for BCHDigit(d)-BCHDigit(G15) >= 0 {
		d ^= G15 << (BCHDigit(d) - BCHDigit(G15))
	}
	return ((data << 10) | d) ^ G15_MASK
}

// BCHDigit returns the number of bits in the binary representation of the number.
func BCHDigit(data int) int {
	digit := 0
	for data != 0 {
		digit++
		data >>= 1
	}
	return digit
}

// BCHTypeNumber calculates BCH code for type number.
func BCHTypeNumber(data int) int {
	d := data << 12
	for BCHDigit(d)-BCHDigit(G18) >= 0 {
		d ^= G18 << (BCHDigit(d) - BCHDigit(G18))
	}
	return (data << 12) | d
}

func PatternPosition(version int) []int {
	return PatternPositionTable[version-1]
}

// Mask Return the mask function for the given mask pattern.
func MaskFunc(pattern int) func(int, int) bool {
	switch pattern {
	case 0: // 000
		return func(i, j int) bool { return (i+j)%2 == 0 }
	case 1: // 001
		return func(i, j int) bool { return i%2 == 0 }
	case 2: // 010
		return func(i, j int) bool { return j%3 == 0 }
	case 3: // 011
		return func(i, j int) bool { return (i+j)%3 == 0 }
	case 4: // 100
		return func(i, j int) bool { return (i/2+j/3)%2 == 0 }
	case 5: // 101
		return func(i, j int) bool { return (i*j)%2+(i*j)%3 == 0 }
	case 6: // 110
		return func(i, j int) bool { return ((i*j)%2+(i*j)%3)%2 == 0 }
	case 7: // 111
		return func(i, j int) bool { return ((i*j)%3+(i+j)%2)%2 == 0 }
	default:
		panic("Bad mask pattern")
	}
}

func ModeSizeVersion(version int) map[int]int {
	if version <= 9 {
		return ModeSizeSmall
	} else if version <= 26 {
		return ModeSizeMedium
	} else {
		return ModeSizeLarge
	}
}

func CheckVersion(version int) bool {
	return version >= 1 && version <= 40
}

func OutIsTTY(out *os.File) bool {
	stat, err := out.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func LostPoint(modules [][]*bool) int {
	modulesCount := len(modules)

	lostPoint := 0

	lostPoint += lostPointLevel1(modules, modulesCount)
	lostPoint += lostPointLevel2(modules, modulesCount)
	lostPoint += lostPointLevel3(modules, modulesCount)
	lostPoint += lostPointLevel4(modules, modulesCount)

	return lostPoint
}

func lostPointLevel1(modules [][]*bool, modulesCount int) int {
	lostPoint := 0
	container := make([]int, modulesCount+1)

	for row := 0; row < modulesCount; row++ {
		thisRow := modules[row]
		previousColor := thisRow[0]
		length := 0
		for col := 0; col < modulesCount; col++ {
			if thisRow[col] == previousColor {
				length++
			} else {
				if length >= 5 {
					container[length]++
				}
				length = 1
				previousColor = thisRow[col]
			}
		}
		if length >= 5 {
			container[length]++
		}
	}

	for col := 0; col < modulesCount; col++ {
		previousColor := modules[0][col]
		length := 0
		for row := 0; row < modulesCount; row++ {
			if modules[row][col] == previousColor {
				length++
			} else {
				if length >= 5 {
					container[length]++
				}
				length = 1
				previousColor = modules[row][col]
			}
		}
		if length >= 5 {
			container[length]++
		}
	}

	for eachLength := 5; eachLength <= modulesCount; eachLength++ {
		lostPoint += container[eachLength] * (eachLength - 2)
	}

	return lostPoint
}

func lostPointLevel2(modules [][]*bool, modulesCount int) int {
	lostPoint := 0

	for row := 0; row < modulesCount-1; row++ {
		thisRow := modules[row]
		nextRow := modules[row+1]
		for col := 0; col < modulesCount-1; col++ {
			topRight := thisRow[col+1]
			if topRight != nextRow[col+1] {
				col++ // Skip the next column
			} else if topRight != thisRow[col] {
				continue
			} else if topRight != nextRow[col] {
				continue
			} else {
				lostPoint += 3
			}
		}
	}

	return lostPoint
}

func lostPointLevel3(modules [][]*bool, modulesCount int) int {
	lostPoint := 0

	for row := 0; row < modulesCount; row++ {
		thisRow := modules[row]
		for col := 0; col < modulesCount-10; col++ {
			if !*thisRow[col+1] &&
				*thisRow[col+4] &&
				!*thisRow[col+5] &&
				*thisRow[col+6] &&
				!*thisRow[col+9] &&
				((*thisRow[col+0] &&
					*thisRow[col+2] &&
					*thisRow[col+3] &&
					!*thisRow[col+7] &&
					!*thisRow[col+8] &&
					!*thisRow[col+10]) ||
					(!*thisRow[col+0] &&
						!*thisRow[col+2] &&
						!*thisRow[col+3] &&
						*thisRow[col+7] &&
						*thisRow[col+8] &&
						*thisRow[col+10])) {
				lostPoint += 40
			}
			if *thisRow[col+10] {
				col++ // Skip the next column
			}
		}
	}

	for col := 0; col < modulesCount; col++ {
		for row := 0; row < modulesCount-10; row++ {
			if !*modules[row+1][col] &&
				*modules[row+4][col] &&
				!*modules[row+5][col] &&
				*modules[row+6][col] &&
				!*modules[row+9][col] &&
				((*modules[row+0][col] &&
					*modules[row+2][col] &&
					*modules[row+3][col] &&
					!*modules[row+7][col] &&
					!*modules[row+8][col] &&
					!*modules[row+10][col]) ||
					(!*modules[row+0][col] &&
						!*modules[row+2][col] &&
						!*modules[row+3][col] &&
						*modules[row+7][col] &&
						*modules[row+8][col] &&
						*modules[row+10][col])) {
				lostPoint += 40
			}
			if *modules[row+10][col] {
				row++ // Skip the next row
			}
		}
	}

	return lostPoint
}

func lostPointLevel4(modules [][]*bool, modulesCount int) int {
	darkCount := 0
	for _, row := range modules {
		for _, module := range row {
			if *module {
				darkCount++
			}
		}
	}

	percent := float64(darkCount) / float64(modulesCount*modulesCount)
	rating := int(math.Abs(percent*100-50) / 5)
	return rating * 10
}

func LengthInBits(mode int, version int) int {
	if mode != ModeNumeric && mode != ModeAlphanumeric && mode != ModeByte && mode != ModeKanji {
		panic(fmt.Sprintf("Invalid mode (%d)", mode))
	}

	if !CheckVersion(version) {
		panic(fmt.Sprintf("Invalid version (%d)", version))
	}

	return ModeSizeVersion(version)[mode]
}

func Contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// BitBuffer is a structure to hold bits and manage them.
type BitBuffer struct {
	buffer []int
	length int
}

// NewBitBuffer creates a new BitBuffer.
func NewBitBuffer() *BitBuffer {
	return &BitBuffer{
		buffer: make([]int, 0),
		length: 0,
	}
}

// String returns a string representation of the BitBuffer.
func (b *BitBuffer) String() string {
	str := ""
	for _, n := range b.buffer {
		str += fmt.Sprintf("%08b", n)
	}
	return str
}

// Get returns the bit at the specified index.
func (b *BitBuffer) Get(index int) bool {
	bufIndex := index / 8
	return ((b.buffer[bufIndex] >> (7 - index%8)) & 1) == 1
}

// Put adds the specified number of bits to the buffer.
func (b *BitBuffer) Put(num, length int) {
	for i := 0; i < length; i++ {
		b.PutBit(((num >> (length - i - 1)) & 1) == 1)
	}
}

// Len returns the number of bits in the buffer.
func (b *BitBuffer) Len() int {
	return b.length
}

// PutBit adds a single bit to the buffer.
func (b *BitBuffer) PutBit(bit bool) {
	bufIndex := b.length / 8
	if len(b.buffer) <= bufIndex {
		b.buffer = append(b.buffer, 0)
	}
	if bit {
		b.buffer[bufIndex] |= 0x80 >> (b.length % 8)
	}
	b.length++
}

// BisectLeft finds the insertion point for x in a to maintain sorted order.
// The return value is the index where to insert x to keep a sorted.
func BisectLeft(a []int, x int) int {
	lo, hi := 0, len(a)
	for lo < hi {
		mid := (lo + hi) / 2
		if a[mid] < x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

type QRData struct {
	data []byte
	mode int
}

// NewQRData creates a new QRData instance.
func NewQRData(data []byte, mode int, checkData bool) (*QRData, error) {
	if checkData {
		data = toBytes(data)
	}

	if mode == 0 {
		mode = OptimalMode(data)
	} else {
		if mode != ModeNumeric && mode != ModeAlphanumeric && mode != ModeByte {
			return nil, fmt.Errorf("invalid mode (%d)", mode)
		}
		if checkData && mode < OptimalMode(data) {
			return nil, fmt.Errorf("provided data cannot be represented in mode %d", mode)
		}
	}

	return &QRData{
		data: data,
		mode: mode,
	}, nil
}

// Len returns the length of the data.
func (q *QRData) Len() int {
	return len(q.data)
}

func (q *QRData) GetMode() int {
	return q.mode
}

// Write writes the data to the buffer.
func (q *QRData) Write(buffer *BitBuffer) {
	if q.mode == ModeNumeric {
		for i := 0; i < len(q.data); i += 3 {
			chars := q.data[i:min(i+3, len(q.data))]
			bitLength := NumberLength[len(chars)]
			buffer.Put(bytesToInt(chars), bitLength)
		}
	} else if q.mode == ModeAlphanumeric {
		for i := 0; i < len(q.data); i += 2 {
			chars := q.data[i:min(i+2, len(q.data))]
			if len(chars) > 1 {
				buffer.Put(
					alphanumericIndex(chars[0])*45+alphanumericIndex(chars[1]), 11)
			} else {
				buffer.Put(alphanumericIndex(chars[0]), 6)
			}
		}
	} else {
		for _, c := range q.data {
			buffer.Put(int(c), 8)
		}
	}
}

// String returns a string representation of the QRData.
func (q *QRData) String() string {
	return string(q.data)
}

// Calculate the optimal mode for this chunk of data.
func OptimalMode(data []byte) int {
	if isDigit(data) {
		return ModeNumeric
	}
	if ReAlphaNumeric.Match(data) {
		return ModeAlphanumeric
	}
	return ModeByte
}

func isDigit(data []byte) bool {
	for _, b := range data {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// toBytes converts data to a byte slice if it isn't a byte slice already.
func toBytes(data interface{}) []byte {
	switch v := data.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}

// min returns the smaller of x or y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// alphanumericIndex returns the index of the character in the alphanumeric set.
func alphanumericIndex(c byte) int {
	return strings.IndexByte(AlphanumericChars, c)
}

// bytesToInt converts a byte slice to an integer.
func bytesToInt(bytes []byte) int {
	result := 0
	for _, b := range bytes {
		result = result*10 + int(b-'0')
	}
	return result
}

// OptimalDataChunks returns an iterator of QRData chunks optimized to the data content.
func OptimalDataChunks(data []byte, minimum int) ([]*QRData, error) {
	if minimum <= 0 {
		minimum = 4
	}
	data = toBytes(data)
	numPattern := regexp.MustCompile(`\d`)
	alphaPattern := regexp.MustCompile("[" + regexp.QuoteMeta(AlphanumericChars) + "]")
	if len(data) <= minimum {
		numPattern = regexp.MustCompile(`^\d+$`)
		alphaPattern = regexp.MustCompile("^[" + regexp.QuoteMeta(AlphanumericChars) + "]+$")
	} else {
		repeat := fmt.Sprintf("{%d,}", minimum)
		numPattern = regexp.MustCompile(`\d` + repeat)
		alphaPattern = regexp.MustCompile("[" + regexp.QuoteMeta(AlphanumericChars) + "]" + repeat)
	}
	return optimalSplit(data, numPattern, alphaPattern)
}

func optimalSplit(data []byte, numPattern, alphaPattern *regexp.Regexp) ([]*QRData, error) {
	var result []*QRData
	for len(data) > 0 {
		numMatch := numPattern.FindIndex(data)
		if numMatch == nil {
			break
		}
		start, end := numMatch[0], numMatch[1]
		if start > 0 {
			result = append(result, &QRData{data: data[:start], mode: ModeByte})
		}
		result = append(result, &QRData{data: data[start:end], mode: ModeNumeric})
		data = data[end:]
	}
	if len(data) > 0 {
		alphaMatch := alphaPattern.FindIndex(data)
		if alphaMatch != nil {
			start, end := alphaMatch[0], alphaMatch[1]
			if start > 0 {
				result = append(result, &QRData{data: data[:start], mode: ModeByte})
			}
			result = append(result, &QRData{data: data[start:end], mode: ModeAlphanumeric})
			data = data[end:]
		} else {
			result = append(result, &QRData{data: data, mode: ModeByte})
		}
	}
	return result, nil
}

func CreateBytes(buffer *BitBuffer, rsBlocks []base.RSBlock) []byte {
	offset := 0

	maxDcCount := 0
	maxEcCount := 0

	dcdata := make([][]int, len(rsBlocks))
	ecdata := make([][]int, len(rsBlocks))

	for r := 0; r < len(rsBlocks); r++ {
		rsBlock := rsBlocks[r]
		dcCount := rsBlock.DataCount
		ecCount := rsBlock.TotalCount - dcCount

		if maxDcCount < dcCount {
			maxDcCount = dcCount
		}
		if maxEcCount < ecCount {
			maxEcCount = ecCount
		}

		dcdata[r] = make([]int, dcCount)
		for i := 0; i < len(dcdata[r]); i++ {
			dcdata[r][i] = 0xff & buffer.buffer[i+offset]
		}
		offset += dcCount

		// Get error correction polynomial.
		rsPoly, err := base.NewPolynomial([]int{1}, 0)
		if err != nil {
			log.Printf("Failed to create polynomial: %v", err)
			return nil
		}

		for i := 0; i < ecCount; i++ {
			child, err := base.NewPolynomial([]int{1, base.Gexp(i)}, 0)
			if err != nil {
				log.Printf("Failed to create polynomial: %v", err)
				return nil
			}
			rsPoly, err = rsPoly.Mul(child)
			if err != nil {
				log.Printf("Failed to multiply polynomials: %v", err)
				return nil
			}
		}

		rawPoly, err := base.NewPolynomial(dcdata[r], rsPoly.Len()-1)
		if err != nil {
			log.Printf("Failed to create raw polynomial: %v", err)
			return nil
		}

		modPoly, err := rawPoly.Mod(rsPoly)
		if err != nil {
			log.Printf("Failed to mod polynomial: %v", err)
			return nil
		}

		ecdata[r] = make([]int, rsPoly.Len()-1)
		modOffset := modPoly.Len() - len(ecdata[r])
		for i := 0; i < len(ecdata[r]); i++ {
			modIndex := i + modOffset
			if modIndex >= 0 {
				ecdata[r][i] = modPoly.Get(modIndex)
			} else {
				ecdata[r][i] = 0
			}
		}
	}

	totalCodewords := 0
	for _, rsBlock := range rsBlocks {
		totalCodewords += rsBlock.TotalCount
	}

	data := make([]byte, totalCodewords)
	index := 0

	for i := 0; i < maxDcCount; i++ {
		for r := 0; r < len(rsBlocks); r++ {
			if i < len(dcdata[r]) {
				data[index] = byte(dcdata[r][i])
				index++
			}
		}
	}

	for i := 0; i < maxEcCount; i++ {
		for r := 0; r < len(rsBlocks); r++ {
			if i < len(ecdata[r]) {
				data[index] = byte(ecdata[r][i])
				index++
			}
		}
	}

	return data
}

func CreateData(version int, errorCorrection int, dataList []*QRData) ([]byte, error) {
	buffer := NewBitBuffer()
	for _, data := range dataList {
		buffer.Put(data.mode, 4)
		buffer.Put(len(data.data), LengthInBits(data.mode, version))
		data.Write(buffer)
	}

	// Calculate the maximum number of bits for the given version.
	rsBlocks, err := base.RSBlocks(version, errorCorrection)
	if err != nil {
		return nil, err
	}

	bitLimit := 0
	for _, block := range rsBlocks {
		bitLimit += block.DataCount * 8
	}
	if buffer.Len() > bitLimit {
		return nil, fmt.Errorf("code length overflow. Data size (%d) > size available (%d)", buffer.Len(), bitLimit)
	}

	// Terminate the bits (add up to four 0s).
	for i := 0; i < min(bitLimit-buffer.Len(), 4); i++ {
		buffer.PutBit(false)
	}

	// Delimit the string into 8-bit words, padding with 0s if necessary.
	delimit := buffer.Len() % 8
	if delimit != 0 {
		for i := 0; i < 8-delimit; i++ {
			buffer.PutBit(false)
		}
	}

	// Add special alternating padding bitstrings until buffer is full.
	bytesToFill := (bitLimit - buffer.Len()) / 8
	for i := 0; i < bytesToFill; i++ {
		if i%2 == 0 {
			buffer.Put(PAD0, 8)
		} else {
			buffer.Put(PAD1, 8)
		}
	}

	return CreateBytes(buffer, rsBlocks), nil
}
