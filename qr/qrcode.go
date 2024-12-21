package qr

import (
	"fmt"
	"os"
	"qrcode/base"
	"qrcode/constants"
	"qrcode/image"
	"qrcode/utils"
	"reflect"
)

type ModulesType [][]*bool

type QRCode struct {
	modules         ModulesType
	modulesCount    int
	version         int
	errorCorrection int
	BoxSize         int
	border          int
	maskPattern     int
	imageFactory    image.PilImage
	DataList        []utils.QRData
	dataCache       []int
}

type ActiveWithNeighbors struct {
	NW bool
	N  bool
	NE bool
	W  bool
	me bool
	E  bool
	SW bool
	S  bool
	SE bool
}

// Cache modules generated just based on the QR Code version
var precomputedQRBlanks = make(map[int]ModulesType)

func Make(data interface{}, kwargs map[string]interface{}) (image.PilImage, error) {
	version := kwargs["version"].(int)
	errorCorrection := kwargs["error_correction"].(int)
	boxSize := kwargs["box_size"].(int)
	border := kwargs["border"].(int)
	maskPattern := kwargs["mask_pattern"].(int)
	qr, err := NewQRCode(version, errorCorrection, boxSize, border, image.PilImage{}, maskPattern)
	if err != nil {
		return image.PilImage{}, err
	}
	qr.SetVersion(version)

	if err := qr.AddData(data, 0); err != nil {
		return image.PilImage{}, err
	}

	return qr.MakeImage(image.PilImage{}, kwargs)
}

func CheckBoxSize(size int) error {
	if size < 0 {
		return fmt.Errorf("Invalid box size: %d", size)
	}
	return nil
}

func CheckBorder(border int) error {
	if border < 0 {
		return fmt.Errorf("Invalid border size: %d", border)
	}
	return nil
}

func CheckMaskPattern(pattern int) error {
	if pattern < 0 || pattern > 7 {
		return fmt.Errorf("Invalid mask pattern: %d", pattern)
	}
	return nil
}

func Copy2DArray(src ModulesType) ModulesType {
	dst := make(ModulesType, len(src))
	for i := range src {
		dst[i] = append([]*bool(nil), src[i]...)
	}
	return dst
}

// dataCount returns the data count of an RSBLOCK
func dataCount(block base.RSBlock) int {
	return block.DataCount
}

// BIT_LIMIT_TABLE precomputes bit count limits, indexed by error correction level and code size
var BIT_LIMIT_TABLE = func() [][]int {
	table := make([][]int, 4)
	for errorCorrection := 0; errorCorrection < 4; errorCorrection++ {
		table[errorCorrection] = make([]int, 41)
		for version := 1; version <= 40; version++ {
			rsBlocks, err := base.RSBlocks(version, errorCorrection)
			if err != nil {
				panic(err)
			}
			bitCount := 0
			for _, block := range rsBlocks {
				bitCount += 8 * dataCount(block)
			}
			table[errorCorrection][version] = bitCount
		}
	}
	return table
}()

func NewQRCode(version int, errorCorrection, boxSize, border int, imageFactory image.PilImage, maskPattern int) (*QRCode, error) {
	if err := CheckBoxSize(boxSize); err != nil {
		return nil, err
	}
	if err := CheckBorder(border); err != nil {
		return nil, err
	}
	if maskPattern != 0 {
		if err := CheckMaskPattern(maskPattern); err != nil {
			return nil, err
		}
	}

	qr := &QRCode{
		version:         version,
		errorCorrection: errorCorrection,
		BoxSize:         boxSize,
		border:          border,
		maskPattern:     maskPattern,
		imageFactory:    imageFactory,
	}
	qr.SetVersion(version)

	qr.Clear()
	return qr, nil
}

func (q *QRCode) Clear() {
	// Reset the internal data
	q.modules = make(ModulesType, 0)
	q.modulesCount = 0
	q.dataCache = nil
	q.DataList = make([]utils.QRData, 0)
}

func (q *QRCode) Version() int {
	if q.version == 0 {
		q.BestFit(q.version)
	}
	return q.version
}

func (q *QRCode) SetVersion(value int) int {
	if value != 0 && !utils.CheckVersion(value) {
		panic(fmt.Sprintf("Invalid version: %d", value))
	}
	q.version = value
	return q.version
}

func (q *QRCode) MaskPattern() int {
	return q.maskPattern
}

func (q *QRCode) SetMaskPattern(value int) {
	if err := CheckMaskPattern(value); err != nil {
		panic(err)
	}
	q.maskPattern = value
}

func (q *QRCode) AddData(data any, optimize int) error {
	if optimize < 0 {
		return fmt.Errorf("Invalid optimize value: %d", optimize)
	}

	switch v := data.(type) {
	case utils.QRData:
		q.DataList = append(q.DataList, v)
	case string:
		fmt.Printf("String: %s\n", v)
		if optimize > 0 {
			chunks, err := utils.OptimalDataChunks([]byte(v), optimize)
			if err != nil {
				return err
			}
			for _, chunk := range chunks {
				q.DataList = append(q.DataList, *chunk)
			}
		} else {
			data, err := utils.NewQRData([]byte(v), 0, true)
			if err != nil {
				return err
			}
			q.DataList = append(q.DataList, *data)
		}
	default:
		return fmt.Errorf("Unsupported data type: %T", v)
	}

	q.dataCache = nil
	return nil
}

func (q *QRCode) Make(fit bool) error {
	if fit || q.Version() == 0 {
		q.BestFit(q.Version())
	}
	fmt.Printf("VersionMake: %d\n", q.Version())
	if q.maskPattern == 0 {
		q.MakeImpl(false, q.BestMaskPattern())
	} else {
		q.MakeImpl(false, q.maskPattern)
	}
	return nil
}

func (q *QRCode) MakeImpl(test bool, maskPattern int) {
	q.modulesCount = q.Version()*4 + 17

	if precomputedModules, ok := precomputedQRBlanks[q.Version()]; ok {
		q.modules = Copy2DArray(precomputedModules)
	} else {
		q.modules = make(ModulesType, q.modulesCount)
		for i := range q.modules {
			q.modules[i] = make([]*bool, q.modulesCount)
		}
		q.SetupPositionProbePattern(0, 0)
		q.SetupPositionProbePattern(q.modulesCount-7, 0)
		q.SetupPositionProbePattern(0, q.modulesCount-7)
		q.SetupPositionAdjustPattern()
		q.SetupTimingPattern()

		precomputedQRBlanks[q.Version()] = Copy2DArray(q.modules)
	}

	q.SetupTypeInfo(test, maskPattern)

	if q.Version() >= 7 {
		q.SetupTypeNumber(test)
	}

	if q.dataCache == nil {
		qrDataList := make([]*utils.QRData, len(q.DataList))
		for i := range q.DataList {
			qrDataList[i] = &q.DataList[i]
		}
		dataCache, err := utils.CreateData(q.Version(), q.errorCorrection, qrDataList)
		if err != nil {
			panic(err)
		}
		q.dataCache = make([]int, len(dataCache))
		for i, b := range dataCache {
			q.dataCache[i] = int(b)
		}
	}
	dataCacheBytes := make([]byte, len(q.dataCache))
	for i, v := range q.dataCache {
		dataCacheBytes[i] = byte(v)
	}
	q.MapData(dataCacheBytes, maskPattern)
}

func (q *QRCode) SetupTypeNumber(test bool) {
	bits := utils.BCHTypeNumber(q.version)

	for i := 0; i < 18; i++ {
		mod := !test && ((bits>>i)&1) == 1
		q.modules[i/3][i%3+q.modulesCount-8-3] = &mod
	}

	for i := 0; i < 18; i++ {
		mod := !test && ((bits>>i)&1) == 1
		q.modules[i%3+q.modulesCount-8-3][i/3] = &mod
	}
}

func (q *QRCode) SetupTypeInfo(test bool, maskPattern int) {
	data := (q.errorCorrection << 3) | maskPattern
	bits := utils.BCHTypeInfo(data)

	// vertical
	for i := 0; i < 15; i++ {
		mod := !test && ((bits>>i)&1) == 1

		if i < 6 {
			q.modules[i][8] = &mod
		} else if i < 8 {
			q.modules[i+1][8] = &mod
		} else {
			q.modules[q.modulesCount-15+i][8] = &mod
		}
	}

	// horizontal
	for i := 0; i < 15; i++ {
		mod := !test && ((bits>>i)&1) == 1

		if i < 8 {
			q.modules[8][q.modulesCount-i-1] = &mod
		} else if i < 9 {
			q.modules[8][15-i-1+1] = &mod
		} else {
			q.modules[8][15-i-1] = &mod
		}
	}

	// fixed module
	mod := !test
	q.modules[q.modulesCount-8][8] = &mod
}

func (q *QRCode) SetupTimingPattern() {
	for r := 8; r < q.modulesCount-8; r++ {
		if q.modules[r][6] != nil {
			continue
		}
		val := r%2 == 0
		q.modules[r][6] = &val
	}

	for c := 8; c < q.modulesCount-8; c++ {
		if q.modules[6][c] != nil {
			continue
		}
		val := c%2 == 0
		q.modules[6][c] = &val
	}
}

func (q *QRCode) SetupPositionProbePattern(row, col int) {
	for r := -1; r <= 7; r++ {
		if q.isOutOfBounds(row + r) {
			continue
		}

		for c := -1; c <= 7; c++ {
			if q.isOutOfBounds(col + c) {
				continue
			}

			q.setModuleValue(row+r, col+c, r, c)
		}
	}
}

func (q *QRCode) isOutOfBounds(index int) bool {
	return index <= -1 || q.modulesCount <= index
}

func (q *QRCode) setModuleValue(row, col, r, c int) {
	if (0 <= r && r <= 6 && (c == 0 || c == 6)) ||
		(0 <= c && c <= 6 && (r == 0 || r == 6)) ||
		(2 <= r && r <= 4 && 2 <= c && c <= 4) {
		val := true
		q.modules[row][col] = &val
	} else {
		val := false
		q.modules[row][col] = &val
	}
}

func (q *QRCode) SetupPositionAdjustPattern() {
	pos := utils.PatternPosition(q.Version())

	for i := range pos {
		row := pos[i]

		for j := range pos {
			col := pos[j]

			if q.modules[row][col] != nil {
				continue
			}

			q.setPositionAdjustPattern(row, col)
		}
	}
}

func (q *QRCode) setPositionAdjustPattern(row, col int) {
	for r := -2; r <= 2; r++ {
		for c := -2; c <= 2; c++ {
			q.setPositionAdjustPatternValue(row, col, r, c)
		}
	}
}

func (q *QRCode) setPositionAdjustPatternValue(row, col, r, c int) {
	if r == -2 || r == 2 || c == -2 || c == 2 || (r == 0 && c == 0) {
		val := true
		q.modules[row+r][col+c] = &val
	} else {
		val := false
		q.modules[row+r][col+c] = &val
	}
}

func (q *QRCode) BestFit(start int) int {
	if !utils.CheckVersion(start) {
		panic(fmt.Sprintf("Invalid version: %d", start))
	}

	modeSizes := utils.ModeSizeVersion(start)
	buffer := utils.NewBitBuffer()
	for _, data := range q.DataList {
		buffer.Put(data.GetMode(), 4)
		buffer.Put(data.Len(), modeSizes[data.GetMode()])
		data.Write(buffer)
	}

	// FIXME: This is a hack to work around the fact that the bisect_left function is not working as expected
	// bisect := utils.BisectLeft(BIT_LIMIT_TABLE[q.errorCorrection], start)

	if q.Version() == 41 {
		panic("Data overflow error")
	}

	// Now check whether we need more bits for the mode sizes, recursing if our guess was too low
	status := reflect.DeepEqual(modeSizes, utils.ModeSizeVersion(q.Version()))
	if !status {
		q.BestFit(q.Version())
	}
	return q.Version()
}

func (q *QRCode) BestMaskPattern() int {
	minLostPoint := int(^uint(0) >> 1)
	bestPattern := 0

	for pattern := 0; pattern < 8; pattern++ {
		q.MakeImpl(true, pattern)
		lostPoint := utils.LostPoint(q.modules)

		if pattern == 0 || minLostPoint > lostPoint {
			minLostPoint = lostPoint
			bestPattern = pattern
		}
	}
	return bestPattern
}

func (q *QRCode) PrintASCII(out *os.File, tty bool, invert bool) error {
	if out == nil {
		out = os.Stdout
	}

	if tty && !utils.OutIsTTY(out) {
		return fmt.Errorf("not a tty")
	}

	if q.dataCache == nil {
		q.Make(true)
	}

	modcount := q.modulesCount
	codes := []string{"█", "▄", "▀", " "}

	if tty {
		invert = true
	}
	if invert {
		codes = []string{" ", "▄", "▀", "█"}
	}

	getModule := func(x, y int) int {
		if invert && q.border > 0 && (x >= modcount+q.border || y >= modcount+q.border) {
			return 1
		}
		if x < 0 || y < 0 || x >= modcount || y >= modcount {
			return 0
		}
		if q.modules[x][y] != nil && *q.modules[x][y] {
			return 1
		}
		return 0
	}

	for r := -q.border; r < modcount+q.border; r += 2 {
		if tty {
			if !invert || r < modcount+q.border-1 {
				fmt.Fprint(out, "\x1b[48;5;232m") // Background black
			}
			fmt.Fprint(out, "\x1b[38;5;255m") // Foreground white
		}
		for c := -q.border; c < modcount+q.border; c++ {
			pos := getModule(r, c) + (getModule(r+1, c) << 1)
			fmt.Fprint(out, codes[pos])
		}
		if tty {
			fmt.Fprint(out, "\x1b[0m")
		}
		fmt.Fprintln(out)
	}

	return nil
}

func (q *QRCode) MakeImage(imageFactory image.PilImage, kwargs map[string]interface{}) (image.PilImage, error) {
	if embeddedImagePath, ok := kwargs["embedded_image_path"]; ok && embeddedImagePath != nil {
		if q.errorCorrection != constants.ERROR_CORRECT_H {
			return image.PilImage{}, fmt.Errorf("Error correction level must be ERROR_CORRECT_H if an embedded image is provided")
		}
	}
	if embeddedImage, ok := kwargs["embedded_image"]; ok && embeddedImage != nil {
		if q.errorCorrection != constants.ERROR_CORRECT_H {
			return image.PilImage{}, fmt.Errorf("Error correction level must be ERROR_CORRECT_H if an embedded image is provided")
		}
	}

	if err := CheckBoxSize(q.BoxSize); err != nil {
		return image.PilImage{}, err
	}

	if q.dataCache == nil {
		if err := q.Make(true); err != nil {
			return image.PilImage{}, err
		}
	}

	if reflect.ValueOf(imageFactory).IsNil() {
		if !reflect.ValueOf(q.imageFactory).IsNil() {
			imageFactory = q.imageFactory
		} else {
			imageFactory = *image.NewPilImage(q.border, q.modulesCount, q.BoxSize, nil, nil)
		}
	}

	modules := make([][]bool, len(q.modules))
	for i := range q.modules {
		modules[i] = make([]bool, len(q.modules[i]))
		for j := range q.modules[i] {
			if q.modules[i][j] != nil {
				modules[i][j] = *q.modules[i][j]
			}
		}
	}
	im := image.NewPilImage(q.border, q.modulesCount, q.BoxSize, modules, nil)

	if im.NeedsDrawRect {
		for r := 0; r < q.modulesCount; r++ {
			for c := 0; c < q.modulesCount; c++ {
				if q.modules[r][c] != nil && *q.modules[r][c] {
					if im.NeedsContext {
						im.DrawRectContext(r, c, q)
					} else {
						im.DrawRect(r, c)
					}
				}
			}
		}
	}

	if im.NeedsProcessing {
		im.Process()
	}

	return *im, nil
}

func (q *QRCode) IsConstrained(row, col int) bool {
	return row >= 0 &&
		row < len(q.modules) &&
		col >= 0 &&
		col < len(q.modules[row])
}

func (q *QRCode) MapData(data []byte, maskPattern int) {
	inc := -1
	row := q.modulesCount - 1
	bitIndex := 7
	byteIndex := 0

	maskFunc := utils.MaskFunc(maskPattern)

	dataLen := len(data)

	for col := q.modulesCount - 1; col > 0; col -= 2 {
		if col <= 6 {
			col--
		}

		colRange := []int{col, col - 1}

		for {
			for _, c := range colRange {
				if q.modules[row][c] == nil {
					dark := false

					if byteIndex < dataLen {
						dark = ((data[byteIndex] >> bitIndex) & 1) == 1
					}

					if maskFunc(row, c) {
						dark = !dark
					}

					q.modules[row][c] = &dark
					bitIndex--

					if bitIndex == -1 {
						byteIndex++
						bitIndex = 7
					}
				}
			}

			row += inc

			if row < 0 || row >= q.modulesCount {
				row -= inc
				inc = -inc
				break
			}
		}
	}
}

func (q *QRCode) GetMatrix() [][]bool {
	if q.dataCache == nil {
		q.Make(true)
	}

	if q.border == 0 {
		matrix := make([][]bool, len(q.modules))
		for i, row := range q.modules {
			matrix[i] = make([]bool, len(row))
			for j, cell := range row {
				if cell != nil {
					matrix[i][j] = *cell
				}
			}
		}
		return matrix
	}

	width := len(q.modules) + q.border*2
	code := make([][]bool, width)
	for i := 0; i < q.border; i++ {
		code[i] = make([]bool, width)
		code[width-1-i] = make([]bool, width)
	}

	for i, module := range q.modules {
		row := make([]bool, width)
		for j, cell := range module {
			if cell != nil {
				row[j+q.border] = *cell
			}
		}
		code[i+q.border] = row
	}

	return code
}

func (q *QRCode) ActiveWithNeighbors(row, col int) ActiveWithNeighbors {
	context := make([]bool, 0, 9)
	for r := row - 1; r <= row+1; r++ {
		for c := col - 1; c <= col+1; c++ {
			context = append(context, q.IsConstrained(r, c) && q.modules[r][c] != nil && *q.modules[r][c])
		}
	}
	return ActiveWithNeighbors{
		NW: context[0],
		N:  context[1],
		NE: context[2],
		W:  context[3],
		me: context[4],
		E:  context[5],
		SW: context[6],
		S:  context[7],
		SE: context[8],
	}
}
