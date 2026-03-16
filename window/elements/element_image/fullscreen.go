package element_image


// fullscreen displays the image in a fullscreen overlay for WebKit.
//
// The image asset is extracted and displayed in a scrollable, fullscreen overlay
// in the frontend (JavaScript). The overlay can be closed by pressing ESC or
// clicking outside the image. All keystrokes are captured by the overlay.
//
// This implementation is portable and can be reused for other overlays (e.g., notes).
func (el *ElementImage) fullscreen() error {
	if el.image == nil {
		return nil
	}

	// Extract data URL and dimensions from the image
	dataURL, ok := el.image.Asset().(string)
	if !ok || dataURL == "" {
		return nil
	}

	size := el.image.Size()
	if size == nil {
		return nil
	}

	el.renderer.DisplayImageFullscreen(dataURL, size.X, size.Y)
	// Display the fullscreen overlay through the renderer
	el.renderer.DisplayImageFullscreen(dataURL, size.X, size.Y)
	return nil
}
