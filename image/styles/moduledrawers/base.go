package moduledrawers

type Rectangle struct {
	X, Y, Width, Height int
}

type QRModuleDrawer interface {
	// DrawRect draws a rectangle in the given box. If isActive is true, the box is "active".
	DrawRect(box Rectangle, isActive bool)

	// Initialize sets up values that only the containing Image class knows about.
	Initialize(img any)

	// NeedsNeighbors indicates whether the drawer needs neighbor information.
	NeedsNeighbors() bool
}

type qrModuleDrawer struct {
	img any
}

func (d *qrModuleDrawer) Initialize(img any) {
	d.img = img
}

func (d *qrModuleDrawer) DrawRect(box Rectangle, isActive bool) {
	// Implementation goes here
}

func (d *qrModuleDrawer) NeedsNeighbors() bool {
	return false
}
