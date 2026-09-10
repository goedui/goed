package windows

// SetClearType selects ClearType for fonts subsequently set on this context.
// Use it for opaque UI surfaces copied without scaling; grayscale remains the
// default for terminal bitmaps and other surfaces that may be composited.
func (g *GDIContext) SetClearType(enabled bool) {
	g.fontQuality = fontAntialiased
	if enabled {
		g.fontQuality = fontClearType
	}
}
